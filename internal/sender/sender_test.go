//go:build !integration
// +build !integration

package sender

import (
	"context"
	"sync"
	"testing"

	"go.uber.org/zap"
	"mts.teta.mailsender/internal/mail"
	"mts.teta.mailsender/internal/model"
)

// Test then sender is receive data form db
// also can do Up.
func Test_NonBlockingWhileServe(t *testing.T) {
	logger, _ := zap.NewProduction()
	log := logger.Sugar()
	queue := &queueMock{logger: log}
	sender := New(queue, log, mail.New(nil, log))
	defer sender.Stop()

	queue.wg.Add(2)

	go sender.Serve()
	sender.Up()
	sender.Up()

	logger.Info("Finish")
	queue.wg.Wait()
}

type queueMock struct {
	wg     sync.WaitGroup
	logger *zap.SugaredLogger
}

// call from serve
func (q *queueMock) Get(context.Context) (model.QueueEntry, bool, error) {
	q.wg.Done()
	q.logger.Info("Start wait...")
	return model.QueueEntry{}, false, nil
}

func (q *queueMock) Save(context.Context, model.QueueEntry) (model.EntryId, error) {
	return model.EmptyEntryId, nil
}

func (q *queueMock) Find(ctx context.Context, id model.EntryId) (m model.QueueEntry, err error) {
	return model.QueueEntry{}, nil
}

func (q *queueMock) FindAll(ctx context.Context, skip int64, pageSize int64) (m []model.QueueEntry, err error) {
	return nil, nil
}

func (q *queueMock) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
