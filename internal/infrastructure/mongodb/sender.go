package mongodb

import (
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"time"
	"userservice/internal/infrastructure/messaging"
	timeutil "userservice/internal/util/time"
)

func (c *Connection) putKafkaMessageInOutbox(ctx context.Context, msg messaging.KafkaInternalMessage) error {

	log.Info().Msgf("Putting kafka message in outbox %s", msg.TopicID)
	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = c.outboxCollection.InsertOne(ctx, &Message{
		MsgID:     msg.ID,
		Data:      msgData,
		NextRetry: timeutil.DBNow().Add(time.Duration(c.dbConfig.InitialRetryDelaySeconds)),
		SentAt:    time.Time{},
		Retries:   0,
		State:     StateWaiting.String(),
	})
	if err != nil {
		return err
	}
	return nil
}
