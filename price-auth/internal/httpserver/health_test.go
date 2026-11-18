package httpserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"priceauth/internal/httpserver"
	"priceauth/internal/testutil"
)

func TestHealth_ReturnsOKWhenDatabaseReachable(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "ok", body["status"])
}
