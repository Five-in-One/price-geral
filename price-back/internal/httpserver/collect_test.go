package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"priceback/internal/httpserver"
	"priceback/internal/messaging"
	"priceback/internal/models"
	"priceback/internal/testutil"
)

// fakePublisher records published events instead of talking to RabbitMQ.
type fakePublisher struct {
	mu     sync.Mutex
	keys   []string
	events []any
	err    error
}

func (f *fakePublisher) Publish(_ context.Context, routingKey string, event any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.keys = append(f.keys, routingKey)
	f.events = append(f.events, event)
	return nil
}

type collectResponse struct {
	Market models.Market     `json:"market"`
	Job    models.CollectJob `json:"job"`
	Error  string            `json:"error"`
}

func postCollect(t *testing.T, router http.Handler, body map[string]string) (*httptest.ResponseRecorder, collectResponse) {
	t.Helper()
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/collect", bytes.NewReader(buf))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var resp collectResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, resp
}

func TestCollect_SavesMarketAndJob_PublishesEvent_Returns202(t *testing.T) {
	pool := testutil.NewTestPool(t)
	pub := &fakePublisher{}
	router := httpserver.NewRouter(pool, httpserver.WithPublisher(pub))

	rec, resp := postCollect(t, router, map[string]string{
		"market_name": "Mercado Novo",
		"url":         "https://mercadonovo.com.br/ofertas",
		"city":        "Teresina",
	})

	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	require.Equal(t, "Mercado Novo", resp.Market.Name)
	require.Equal(t, "bot", resp.Market.Source)
	require.Contains(t, resp.Market.Links, "https://mercadonovo.com.br/ofertas")
	require.Equal(t, models.CollectJobQueued, resp.Job.Status)
	require.Equal(t, resp.Market.ID, resp.Job.MarketID)

	require.Equal(t, []string{messaging.CollectRequestedKey}, pub.keys)
	ev, ok := pub.events[0].(messaging.CollectRequested)
	require.True(t, ok)
	require.Equal(t, resp.Job.ID, ev.JobID)
	require.Equal(t, resp.Market.ID, ev.MarketID)
	require.Equal(t, "Mercado Novo", ev.MarketName)
	require.Equal(t, "https://mercadonovo.com.br/ofertas", ev.URL)
	require.NotEmpty(t, ev.EventID)

	// The job is readable for the panel's polling.
	req := httptest.NewRequest(http.MethodGet, "/api/collect/jobs/"+resp.Job.ID, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, req)
	require.Equal(t, http.StatusOK, getRec.Code)
}

func TestCollect_SameMarketAndURLInFlight_Returns409WithExistingJob(t *testing.T) {
	pool := testutil.NewTestPool(t)
	pub := &fakePublisher{}
	router := httpserver.NewRouter(pool, httpserver.WithPublisher(pub))
	body := map[string]string{"market_name": "Mercado Novo", "url": "https://mercadonovo.com.br/"}

	_, first := postCollect(t, router, body)
	rec, second := postCollect(t, router, body)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, "collect_in_progress", second.Error)
	require.Equal(t, first.Job.ID, second.Job.ID)
	require.Len(t, pub.events, 1)
}

func TestCollect_MissingFields_Returns400(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, httpserver.WithPublisher(&fakePublisher{}))

	rec, _ := postCollect(t, router, map[string]string{"market_name": "Mercado Novo"})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rec, _ = postCollect(t, router, map[string]string{"market_name": "Mercado Novo", "url": "ftp://x"})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCollect_BrokerDown_Returns503AndClosesJobAsFailed(t *testing.T) {
	pool := testutil.NewTestPool(t)
	pub := &fakePublisher{err: errors.New("connection refused")}
	router := httpserver.NewRouter(pool, httpserver.WithPublisher(pub))

	rec, _ := postCollect(t, router, map[string]string{"market_name": "Mercado Novo", "url": "https://mercadonovo.com.br/"})
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var status string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT status FROM collect_jobs ORDER BY created_at DESC LIMIT 1`).Scan(&status))
	require.Equal(t, models.CollectJobFailed, status)

	// The market itself is kept, so the collection can be retried later.
	var markets int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM markets WHERE name = 'Mercado Novo'`).Scan(&markets))
	require.Equal(t, 1, markets)
}

func TestCollect_NoPublisherConfigured_Returns503(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec, _ := postCollect(t, router, map[string]string{"market_name": "Mercado Novo", "url": "https://mercadonovo.com.br/"})
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestReloadMarket_OneJobPerLink(t *testing.T) {
	pool := testutil.NewTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `INSERT INTO market_links (id, market_id, url) VALUES
		('l1', 'af', 'https://af.com.br/page=1'),
		('l2', 'af', 'https://af.com.br/page=2')`)
	require.NoError(t, err)

	pub := &fakePublisher{}
	router := httpserver.NewRouter(pool, httpserver.WithPublisher(pub))

	req := httptest.NewRequest(http.MethodPost, "/api/markets/af/collect", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	var resp struct {
		MarketID string              `json:"market_id"`
		Jobs     []models.CollectJob `json:"jobs"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "af", resp.MarketID)
	require.Len(t, resp.Jobs, 2)
	require.Len(t, pub.events, 2)
}

func TestReloadMarket_NoLinks_Returns422(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, httpserver.WithPublisher(&fakePublisher{}))

	req := httptest.NewRequest(http.MethodPost, "/api/markets/se/collect", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestGetJob_Unknown_Returns404(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/collect/jobs/nope", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}
