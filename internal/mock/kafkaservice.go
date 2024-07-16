package mock

import (
	"FACEITBackendTest/internal/kafkamessage"
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

type KafkaServiceMock struct {
	Messages []kafkamessage.InternalMessage
}

func NewKafkaServiceMock() *KafkaServiceMock {
	return &KafkaServiceMock{
		Messages: []kafkamessage.InternalMessage{},
	}
}

func (k *KafkaServiceMock) CleanUp() {
}

func (k *KafkaServiceMock) Run(stopChannel chan struct{}) {
	//TODO implement me
	panic("implement me")
}

func (k *KafkaServiceMock) produce(ctx context.Context, msg kafkamessage.InternalMessage) error {
	k.Messages = append(k.Messages, msg)

	return nil
}

func (k *KafkaServiceMock) EnsureHasMessage(t *testing.T, topicID string, userID string) {
	for _, msg := range k.Messages {
		if msg.TopicID == topicID && bytes.Equal(msg.Key, []byte(userID)) {
			return
		}
	}
	t.Fatalf("message with topic id %s and userID as key %s not found in kafka mock", topicID, userID)
}

func (k *KafkaServiceMock) EnsureHasMessageCount(t *testing.T, messageCount int) {
	require.Equal(t, messageCount, len(k.Messages))
}
