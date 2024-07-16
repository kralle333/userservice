package db

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	MongoDBID primitive.ObjectID `bson:"_id,omitempty"`
	ID        string             `bson:"id"`
	FirstName string             `bson:"first_name"`
	LastName  string             `bson:"last_name"`
	Nickname  string             `bson:"nickname"`
	Password  string             `bson:"password"`
	Email     string             `bson:"email"`
	Country   string             `bson:"country"`
	Salt      string             `bson:"salt"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

type ListUsersResult struct {
	Cursor string
	Limit  int64
	Users  []User
}

type KafkaOutboxMessageState string

func (k KafkaOutboxMessageState) String() string {
	return string(k)
}

const (
	KafkaOutboxMessageStateWaiting    KafkaOutboxMessageState = "waiting"
	KafkaOutboxMessageStateProcessing KafkaOutboxMessageState = "processing"
	KafkaOutboxMessageStateFinished   KafkaOutboxMessageState = "finished"
)

type KafkaOutboxMessage struct {
	MsgID     string    `bson:"msg_id"` // have an id that is easily accessed without needing to deserialize data
	Data      []byte    `bson:"data"`
	NextRetry time.Time `bson:"next_retry"`
	SentAt    time.Time `bson:"sent_at"`
	Retries   int64     `bson:"retries"`
	State     string    `bson:"state"`
}
