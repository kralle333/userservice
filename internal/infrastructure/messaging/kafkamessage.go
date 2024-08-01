package messaging

import (
	"github.com/google/uuid"
)

type KafkaInternalMessage struct {
	ID      string
	TopicID string
	Key     []byte
	Value   []byte
}

func NewInternalMessage(topicID string, key []byte, value []byte) *KafkaInternalMessage {
	return &KafkaInternalMessage{ID: uuid.New().String(), TopicID: topicID, Key: key, Value: value}
}
