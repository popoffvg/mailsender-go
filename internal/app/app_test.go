package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"mts.teta.mailsender/internal/config"
	"mts.teta.mailsender/internal/mailsender"
	"mts.teta.mailsender/internal/model"
	"mts.teta.mailsender/internal/sender"
	"mts.teta.mailsender/internal/store"
	"mts.teta.mailsender/pkg/mongodb"
)

func Test_Create(t *testing.T) {
	var err error
	pool, resource, db := prepareContainer(t)
	// When you're done, kill and remove the container
	defer func() {
		if err = pool.Purge(resource); err != nil {
			t.Logf("Could not purge resource: %s", err)
		}
	}()

	log, _ := zap.NewProduction()
	logger := log.Sugar()
	// logger := zap.NewNop().Sugar()

	queue, err := store.NewMongoQueue(db, logger)
	if err != nil {
		t.Fatal(err)
	}

	count, _ := queue.Count(context.Background())
	if count != 0 {
		t.Fatalf("Not empty DB")
	}

	sender := sender.New(
		queue,
		logger,
		&testSender{},
	)
	cfg := config.Config{
		Addr: "0.0.0.0:8085",
		Mongo: config.MongoConfig{
			Addr: "localhost",
			Port: "27017",
			DB:   "integration_test",
		},
	}
	app := App{
		logger:     logger,
		db:         db,
		queue:      queue,
		server:     mailsender.New(&cfg, logger, queue, sender),
		mailSender: sender,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		app.Start(context.Background())
		time.Sleep(time.Second) // wait start sender
		wg.Done()
	}()
	wg.Wait()
	defer app.Stop(context.Background())

	mailing := model.QueueEntry{}
	b, err := json.Marshal(mailing)
	if err != nil {
		t.Fatal(err)
	}
	data := send(t, http.MethodPost, "mailing", b)
	id := string(data)
	assert.NotEmpty(t, id)
}

func Test_List(t *testing.T) {
	var err error
	pool, resource, db := prepareContainer(t)
	// When you're done, kill and remove the container
	defer func() {
		if err = pool.Purge(resource); err != nil {
			t.Logf("Could not purge resource: %s", err)
		}
	}()

	log, _ := zap.NewProduction()
	logger := log.Sugar()
	// logger := zap.NewNop().Sugar()

	queue, err := store.NewMongoQueue(db, logger)
	if err != nil {
		t.Fatal(err)
	}

	count, _ := queue.Count(context.Background())
	if count != 0 {
		t.Fatalf("Not empty DB")
	}

	sender := sender.New(
		queue,
		logger,
		&testSender{},
	)
	cfg := config.Config{
		Addr: "0.0.0.0:8085",
		Mongo: config.MongoConfig{
			Addr: "localhost",
			Port: "27017",
			DB:   "integration_test",
		},
	}
	app := App{
		logger:     logger,
		db:         db,
		queue:      queue,
		server:     mailsender.New(&cfg, logger, queue, sender),
		mailSender: sender,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		app.Start(context.Background())
		time.Sleep(time.Second) // wait start sender
		wg.Done()
	}()
	wg.Wait()
	defer app.Stop(context.Background())

	mailing := model.QueueEntry{}
	b, err := json.Marshal(mailing)
	if err != nil {
		t.Fatal(err)
	}

	data := send(t, http.MethodPost, "mailing", b)
	id1 := string(data)
	assert.NotEmpty(t, id1)

	data = send(t, http.MethodPost, "mailing", b)
	id2 := string(data)
	assert.NotEmpty(t, id2)

	time.Sleep(2 * time.Second) // wait sending mail

	data = send(t, http.MethodGet, "mailing", make([]byte,0))

	list := mailsender.ListJson{
		Total: 1,
		Data: []mailsender.Info{
			{
				Id: id2,
				Status: "Done",
			},
			{
				Id: id1,
				Status: "Done",
			},
		},
	}

	var sRes mailsender.ListJson
	json.Unmarshal(data, &sRes)
	assert.Equal(t, list, sRes)
}

func Test_Get(t *testing.T) {
	var err error
	pool, resource, db := prepareContainer(t)
	// When you're done, kill and remove the container
	defer func() {
		if err = pool.Purge(resource); err != nil {
			t.Logf("Could not purge resource: %s", err)
		}
	}()

	log, _ := zap.NewProduction()
	logger := log.Sugar()
	// logger := zap.NewNop().Sugar()

	queue, err := store.NewMongoQueue(db, logger)
	if err != nil {
		t.Fatal(err)
	}

	count, _ := queue.Count(context.Background())
	if count != 0 {
		t.Fatalf("Not empty DB")
	}

	sender := sender.New(
		queue,
		logger,
		&testSender{},
	)
	cfg := config.Config{
		Addr: "0.0.0.0:8085",
		Mongo: config.MongoConfig{
			Addr: "localhost",
			Port: "27017",
			DB:   "integration_test",
		},
	}
	app := App{
		logger:     logger,
		db:         db,
		queue:      queue,
		server:     mailsender.New(&cfg, logger, queue, sender),
		mailSender: sender,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		app.Start(context.Background())
		time.Sleep(time.Second) // wait start sender
		wg.Done()
	}()
	wg.Wait()
	defer app.Stop(context.Background())

	mailing := model.QueueEntry{}
	b, err := json.Marshal(mailing)
	if err != nil {
		t.Fatal(err)
	}

	data := send(t, http.MethodPost, "mailing", b)
	id := string(data)
	assert.NotEmpty(t, id)

	data = send(t, http.MethodGet, "mailing/"+id, make([]byte,0))

	mailing.Id = model.EntryId(id)
	mailing.Timestamp = time.Now()
	mailing.Status = model.StatusMailingDone
	var mailingFromServer model.QueueEntry
	json.Unmarshal(data, &mailingFromServer)
	mailingFromServer.Timestamp = mailing.Timestamp

	assert.Equal(t, mailing, mailingFromServer)
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

func send(t *testing.T, method string, path string,  body []byte) []byte {
	req, err := http.NewRequest(method, "http://localhost:8085/" + path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	c := http.Client{}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type testSender struct{}

func (s *testSender) Send(entry *model.QueueEntry) error {
	return nil
}
