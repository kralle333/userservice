package db

import (
	"context"
	"encoding/json"
	"time"
	"userservice/internal/kafkamessage"
	timeutil "userservice/internal/util/time"
)

type KafkaMessageSender interface {
	PutMessageInOutbox(ctx context.Context, msg kafkamessage.InternalMessage) error
}

func (m *MongoDBRepo) PutMessageInOutbox(ctx context.Context, msg kafkamessage.InternalMessage) error {

	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = m.kafkaCollection.InsertOne(ctx, &KafkaOutboxMessage{
		MsgID:     msg.ID,
		Data:      msgData,
		NextRetry: timeutil.DBNow().Add(time.Duration(m.config.InitialRetryDelaySeconds)),
		Retries:   0,
		State:     KafkaOutboxMessageStateWaiting.String(),
	})
	if err != nil {
		return err
	}
	return nil
}
