package mock

import (
	"FACEITBackendTest/internal/kafkamessage"
	"context"
)

type KafkaMessageOutboxMock struct {
	OutboxMessages []kafkamessage.InternalMessage
}

func (k *KafkaMessageOutboxMock) PutMessageInOutbox(ctx context.Context, msg kafkamessage.InternalMessage) error {
	k.OutboxMessages = append(k.OutboxMessages, msg)
	return nil
}

func NewKafkaFailStoreMock() *KafkaMessageOutboxMock {
	return &KafkaMessageOutboxMock{}
}
