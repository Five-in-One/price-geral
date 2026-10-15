package messaging_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"priceback/internal/messaging"
	"priceback/internal/repository"
)

type fakeFinisher struct {
	id  string
	in  repository.FinishJobInput
	err error
}

func (f *fakeFinisher) Finish(_ context.Context, id string, in repository.FinishJobInput) error {
	f.id, f.in = id, in
	return f.err
}

func TestHandleCollectFinished_ClosesJob(t *testing.T) {
	jobs := &fakeFinisher{}
	body := []byte(`{"event_id":"e1","job_id":"job-1","status":"succeeded",
		"blocks_found":10,"products_extracted":8,"products_inserted":7,"error":null}`)

	requeue, err := messaging.HandleCollectFinished(context.Background(), jobs, body)

	require.NoError(t, err)
	require.False(t, requeue)
	require.Equal(t, "job-1", jobs.id)
	require.Equal(t, "succeeded", jobs.in.Status)
	require.Equal(t, 7, jobs.in.ProductsInserted)
	require.Nil(t, jobs.in.Error)
}

func TestHandleCollectFinished_InvalidPayload_DroppedWithoutRequeue(t *testing.T) {
	for _, body := range []string{`not json`, `{"job_id":"","status":"succeeded"}`, `{"job_id":"j","status":"weird"}`} {
		requeue, err := messaging.HandleCollectFinished(context.Background(), &fakeFinisher{}, []byte(body))
		require.Error(t, err, body)
		require.False(t, requeue, body)
	}
}

func TestHandleCollectFinished_UnknownJob_Dropped(t *testing.T) {
	jobs := &fakeFinisher{err: repository.ErrJobNotFound}
	requeue, err := messaging.HandleCollectFinished(context.Background(), jobs, []byte(`{"job_id":"j","status":"failed"}`))
	require.Error(t, err)
	require.False(t, requeue)
}

func TestHandleCollectFinished_DatabaseError_Requeued(t *testing.T) {
	jobs := &fakeFinisher{err: errors.New("db down")}
	requeue, err := messaging.HandleCollectFinished(context.Background(), jobs, []byte(`{"job_id":"j","status":"failed"}`))
	require.Error(t, err)
	require.True(t, requeue)
}
