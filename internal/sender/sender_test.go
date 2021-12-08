//go:build !integration
// +build !integration

package sender

import (
	"context"
	"testing"
	"time"

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

	go sender.Serve()
	time.Sleep(1 * time.Second)
	sender.Up()
	sender.Up()

	logger.Info("Finish")
}

type queueMock struct {
	logger *zap.SugaredLogger
}

// call from serve
func (q *queueMock) Get(context.Context) (model.Mailing, bool, error) {
	q.logger.Info("Start wait...")
	return model.Mailing{}, false, nil
}

func (q *queueMock) Save(context.Context, model.Mailing) (model.EntryId, error) {
	return model.EmptyEntryId, nil
}

func (q *queueMock) Find(ctx context.Context, id model.EntryId) (m model.Mailing, err error) {
	return model.Mailing{}, nil
}

func (q *queueMock) FindAll(ctx context.Context, skip int64, pageSize int64) (m []model.Mailing, err error) {
	return nil, nil
}

func (q *queueMock) Count(ctx context.Context) (int64, error) {
	return 0, nil
}
