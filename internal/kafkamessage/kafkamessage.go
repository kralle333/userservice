package kafkamessage

import (
	"github.com/google/uuid"
)

type InternalMessage struct {
	ID      string
	TopicID string
	Key     []byte
	Value   []byte
}

func NewInternalMessage(topicID string, key []byte, value []byte) *InternalMessage {
	return &InternalMessage{ID: uuid.New().String(), TopicID: topicID, Key: key, Value: value}
}
