//go:build integration
// +build integration

package store

import (
	"context"
	"testing"
	"time"

	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"mts.teta.mailsender/internal/model"
	"mts.teta.mailsender/pkg/mongodb"
)

func Test_SaveFind(t *testing.T) {
	var err error
	pool, resource, db := prepareContainer(t)
	// When you're done, kill and remove the container
	defer func() {
		if err = pool.Purge(resource); err != nil {
			t.Logf("Could not purge resource: %s", err)
		}
	}()

	logger := zap.NewNop().Sugar()

	queue, err := NewMongoQueue(db, logger)
	if err != nil {
		t.Fatal(err)
	}

	etalon := model.QueueEntry{
		Subject: "test",
	}
	id, err := queue.Save(context.Background(), etalon)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, id)

	entry, err := queue.Find(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}

	// not compare id and timestamp
	etalon.Id = id
	etalon.Timestamp = time.Now()
	entry.Timestamp = etalon.Timestamp
	assert.Equal(t, etalon, entry)
}

func Test_FindAll(t *testing.T) {
	var err error
	pool, resource, db := prepareContainer(t)
	// When you're done, kill and remove the container
	defer func() {
		if err = pool.Purge(resource); err != nil {
			t.Logf("Could not purge resource: %s", err)
		}
	}()

	logger := zap.NewNop().Sugar()

	queue, err := NewMongoQueue(db, logger)
	if err != nil {
		t.Fatal(err)
	}

	count, _ := queue.Count(context.Background())
	if count != 0 {
		t.Fatalf("Not empty DB")
	}

	// get few pages
	pauseTime := time.Second * 1
	for i := 0; i < 20; i++ {
		_ = save(t, queue, model.QueueEntry{
			Subject: "empty",
			Status:  model.StatusMailingPending,
		})
	}
	// sleep for separate empty data and not
	time.Sleep(pauseTime)
	id1 := save(t, queue, model.QueueEntry{
		Subject: "test",
		Status:  model.StatusMailingDone,
	})
	time.Sleep(pauseTime)
	id2 := save(t, queue, model.QueueEntry{
		Subject: "test2",
		Status:  model.StatusMailingPending,
	})
	time.Sleep(pauseTime)

	entries, err := queue.FindAll(context.Background(), 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 20, len(entries))
	assert.Equal(t, id2, entries[0].Id)
	assert.Equal(t, id1, entries[1].Id)

	entries, err = queue.FindAll(context.Background(), 20, 20)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(entries))
}

func Test_SaveGet(t *testing.T) {
	var err error
	pool, resource, db := prepareContainer(t)
	// When you're done, kill and remove the container
	defer func() {
		if err = pool.Purge(resource); err != nil {
			t.Logf("Could not purge resource: %s", err)
		}
	}()

	logger := zap.NewNop().Sugar()

	queue, err := NewMongoQueue(db, logger)
	if err != nil {
		t.Fatal(err)
	}

	count, _ := queue.Count(context.Background())
	if count != 0 {
		t.Fatalf("Not empty DB")
	}

	_, ok, _ := queue.Get(context.Background())
	assert.False(t, ok)

	pauseTime := time.Second * 1
	// sleep for separate empty data and not
	time.Sleep(pauseTime)
	id1 := save(t, queue, model.QueueEntry{
		Subject: "test",
		Status:  model.StatusMailingPending,
	})
	time.Sleep(pauseTime)
	save(t, queue, model.QueueEntry{
		Subject: "test2",
		Status:  model.StatusMailingPending,
	})

	entry, ok, err := queue.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, id1, entry.Id)
}

func save(t *testing.T, queue MailingQueue, entry model.QueueEntry) model.EntryId {
	id1, err := queue.Save(context.Background(), entry)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, id1)

	return id1
}

func prepareContainer(t *testing.T) (*dockertest.Pool, *dockertest.Resource, *mongo.Database) {
	var db *mongo.Database
	var err error

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatal(err)
	}
	resource, err := pool.Run("mongo", "5.0.3", nil)
	if err != nil {
		t.Fatalf("Could not start resource: %s", err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		db, err = mongodb.NewClient(context.Background(), "localhost", resource.GetPort("27017/tcp"), "", "", "test", "")
		return err
	}); err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}

	return pool, resource, db
}
