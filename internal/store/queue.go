package store

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"mts.teta.mailsender/internal/model"
)

const (
	timeout = 5 * time.Second
)

type ErrNotFound struct {
	id model.EntryId
}

func (err *ErrNotFound) Error() string {
	return fmt.Sprintf("Object not found with id: %v", err.id)
}

type MailingQueue interface {
	Find(ctx context.Context, id model.EntryId) (m model.QueueEntry, err error)
	FindAll(ctx context.Context, skip int64, pageSize int64) (m []model.QueueEntry, err error)
	Get(context.Context) (model.QueueEntry, bool, error)           // get last mailing for send
	Save(context.Context, model.QueueEntry) (model.EntryId, error) // add mailing to queue
	Count(ctx context.Context) (int64, error)
}

type mailingQueueMongo struct {
	collection *mongo.Collection
	logger     *zap.SugaredLogger
}

func NewMongoQueue(
	db *mongo.Database,
	logger *zap.SugaredLogger,
) (MailingQueue, error) {
	collection := db.Collection("mailing_queue")

	ctx, _ := context.WithTimeout(context.Background(), timeout)
	index := mongo.IndexModel{
		Keys: bson.D{{
			"status",
			"timestamp",
		}},
	}

	collection.Indexes().CreateOne(ctx, index)
	return &mailingQueueMongo{
		collection: collection,
		logger:     logger,
	}, nil
}

func (r *mailingQueueMongo) Find(ctx context.Context, id model.EntryId) (m model.QueueEntry, err error) {
	oid, err := primitive.ObjectIDFromHex(string(id))
	if err != nil {
		return m, fmt.Errorf("failed to convert hex to objectid. hex: %s", id)
	}

	filter := bson.M{"_id": oid}
	result := r.collection.FindOne(ctx, filter)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return m, &ErrNotFound{id}
		}
		return m, fmt.Errorf("failed to find one user by id: %s due to error: %v", id, err)
	}
	if err = result.Decode(&m); err != nil {
		return m, fmt.Errorf("failed to decode user (id:%s) from DB due to error: %v", id, err)
	}

	return m, nil
}

func (r *mailingQueueMongo) FindAll(ctx context.Context, skip int64, pageSize int64) (m []model.QueueEntry, err error) {
	findOptions := options.Find()
	findOptions.SetSkip(skip)
	findOptions.SetLimit(pageSize)
	findOptions.SetSort(bson.D{{"timestamp", -1}})
	cursor, err := r.collection.Find(ctx, bson.D{}, findOptions)
	if cursor.Err() != nil {
		return m, fmt.Errorf("failed to find all users due to error: %v", err)
	}

	if err = cursor.All(ctx, &m); err != nil {
		return m, fmt.Errorf("failed to read all documents from cursor. error: %v", err)
	}

	return m, nil
}

func (q *mailingQueueMongo) Get(ctx context.Context) (model.QueueEntry, bool, error) {
	options := options.FindOne()
	options.SetSort(bson.D{{"timestamp", 1}})
	filter := bson.M{
		"status": model.StatusMailingPending,
	}
	result := q.collection.FindOne(ctx, filter, options)

	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return model.QueueEntry{}, false, nil
		}

		return model.QueueEntry{}, false, result.Err()
	}

	var entry model.QueueEntry
	if err := result.Decode(&entry); err != nil {
		return model.QueueEntry{}, false, err
	}

	return entry, true, nil
}

func (q *mailingQueueMongo) Save(ctx context.Context, entry model.QueueEntry) (model.EntryId, error) {
	var result *mongo.InsertOneResult
	var err error
	var id model.EntryId
	entry.Timestamp = time.Now()

	
	if entry.IsNew() {
		// create
		result, err = q.create(ctx, &entry)
		if err != nil {
			return "", errors.Wrap(err, "failed create mailing")
		}
		oid, ok := result.InsertedID.(primitive.ObjectID)
		if !ok {
			return "", fmt.Errorf("failed to convert objectid to hex. probably oid: %s", oid)
		}
		id = model.EntryId(oid.Hex())
		entry.Id = id

	} else {
		// update
		err = q.update(ctx, &entry)
		if err != nil {
			return "", errors.Wrapf(err, "failed update mailing with id: %v", entry.Id)
		}
		id = entry.Id
	}

	return id, nil
}

func (r *mailingQueueMongo) Count(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.D{}, nil)
}

func (r *mailingQueueMongo) update(ctx context.Context,queue *model.QueueEntry) error {
	objectID, err := primitive.ObjectIDFromHex(string(queue.Id))
	if err != nil {
		return fmt.Errorf("failed to convert user ID to ObjectID. ID=%v", queue.Id)
	}
	filter := bson.M{"_id": objectID}
	userBytes, err := bson.Marshal(queue)
	if err != nil {
		return errors.Wrap(err, "failed to marshal user.")
	}

	var updateUserObj bson.M
	err = bson.Unmarshal(userBytes, &updateUserObj)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal user bytes. error")
	}

	update := bson.M{
		"$set": updateUserObj,
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to execute update user query. error: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("Not found mailing with id: %v", queue.Id)
	}

	return nil
}

func (r *mailingQueueMongo) create(ctx context.Context, queue *model.QueueEntry) (*mongo.InsertOneResult, error) {
	result, err := r.collection.InsertOne(ctx, queue)
	if err != nil {
		return nil, err
	}

	return result, nil
}
