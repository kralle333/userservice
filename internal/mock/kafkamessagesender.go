package mock

import (
	"context"
	"userservice/internal/infrastructure/messaging"
)

type KafkaMessageOutboxMock struct {
	OutboxMessages []messaging.KafkaInternalMessage
}

func (k *KafkaMessageOutboxMock) PutMessageInOutbox(ctx context.Context, msg messaging.KafkaInternalMessage) error {
	k.OutboxMessages = append(k.OutboxMessages, msg)
	return nil
}

func NewKafkaFailStoreMock() *KafkaMessageOutboxMock {
	return &KafkaMessageOutboxMock{}
}
