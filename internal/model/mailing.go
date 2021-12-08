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

type Mailing struct {
	Id        EntryId       `bson:"_id,omitempty" json:"-" swaggerignore:"true"` // if empty than mailing will be created
	Receivers []Receiver    `bson:"receivers" json:"receivers"` 
	Subject   string        `bson:"subject" json:"subject"`
	Text      string        `bson:"text" json:"text"`
	Timestamp time.Time     `bson:"timestamp" swaggerignore:"true"` // define queue priority

	// Status:
	// * 0 - Pending.
	// * 1 - Done.
	// * 2 - Error.
	Status    MailingStatus `bson:"status" json:"-" enums:"0,1,2"`
	Attempts  int           `bson:"attempts" json:"attempts"` // attempts send
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

func (entry *Mailing) StatusToString() string {
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

func (m *Mailing) IsNew() bool {
	return m.Id == ""
}