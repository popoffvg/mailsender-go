package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EntryId string

type MailingStatus int // enum

const (
	StatusMailingPending MailingStatus = iota // Pending
	StatusMailingDone // Done
	StatusMailingError // Error
)

const (
	EmptyEntryId = ""
	AttemptsCount = 3
)

type QueueEntry struct {
	Id        EntryId       `bson:"_id,omitempty" json:"id" swaggerignore:"true` // fill from db driver
	Receivers []Receiver    `bson:"receivers" json:"receivers"` 
	Subject   string        `bson:"subject" json:"subject"`
	Text      string        `bson:"text" json:"text"`
	Timestamp time.Time     `bson:"timestamp" swaggerignore:"true"`

	// Status:
	// * 0 - Pending.
	// * 1 - Done.
	// * 2 - Error.
	Status    MailingStatus `bson:"status" enums:"0,1,2"`
	Attempts  int           
}

type Receiver struct {
	Addr string `bson:"addr" json:"addr"`
	IsSended bool `bson:"isSended" json:"isSended"`
}

// Marshal / unmarshal mailing id for decoupling packages.
func (id EntryId) MarshalBSONValue() (bsontype.Type, []byte, error) {

	p, err := primitive.ObjectIDFromHex(string(id))
	// if string(id) == "" {
	// 	p = string(primitive.NilObjectID)
	// }
	if err != nil {
		return bsontype.Null, nil, err
	}

	return bson.MarshalValue(p)
}

func (entry *QueueEntry) StatusToString() string {
	switch {
	case entry.Status == StatusMailingDone:
		return "Done"

	case entry.Status == StatusMailingError:
		return "Error"

	case entry.Status == StatusMailingPending:
		return "Pending"

	default:
		return ""
	}
}

func (m *QueueEntry) IsNew() bool {
	return m.Id == ""
}