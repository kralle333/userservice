package outbox

import (
	"context"
	"errors"
	"userservice/internal/infrastructure/messaging"
)

var ErrNoPendingMessage = errors.New("no pending message found")

type MessageOutboxRepo interface {
	GetPendingMessage(ctx context.Context) (messaging.KafkaInternalMessage, error)
	RetryMessage(ctx context.Context, id string)
	MarkMessageSent(ctx context.Context, id string)
}
