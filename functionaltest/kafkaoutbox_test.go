package functionaltest

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
	"userservice/internal/db"
	"userservice/internal/kafkamessage"
)

func TestInsertAndWait(t *testing.T) {
	ctx := context.Background()
	defer testerApp.clearDB()

	internalMsg := kafkamessage.NewInternalMessage(testerApp.serverConfig.Kafka.UserAddedTopicName, []byte("hello"), []byte("value"))

	data, err := json.Marshal(&internalMsg)
	require.NoError(t, err)

	msg := db.KafkaOutboxMessage{
		MsgID:     internalMsg.ID,
		Data:      data,
		NextRetry: time.Now().UTC(),
		SentAt:    time.Time{},
		Retries:   0,
		State:     db.KafkaOutboxMessageStateWaiting.String(),
	}

	_, err = testerApp.kafkaOutboxCollection.InsertOne(ctx, msg)
	require.NoError(t, err)

	// sleep a bit, waiting for message to get picked up and sent
	time.Sleep(15 * time.Second)

	entries := getAllKafkaDBEntries(ctx, testerApp.kafkaOutboxCollection)
	require.Len(t, entries, 1)
	require.Equal(t, db.KafkaOutboxMessageStateFinished.String(), entries[0].State)
}
