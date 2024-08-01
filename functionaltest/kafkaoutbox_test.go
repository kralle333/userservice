package functionaltest

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
	"userservice/internal/infrastructure/messaging"
	"userservice/internal/infrastructure/mongodb"
)

func TestInsertAndWait(t *testing.T) {
	ctx := context.Background()
	defer testerApp.clearDB()

	internalMsg := messaging.NewInternalMessage(testerApp.serverConfig.Kafka.Topics.UserAddedTopicName, []byte("hello"), []byte("value"))

	data, err := json.Marshal(&internalMsg)
	require.NoError(t, err)

	msg := mongodb.Message{
		MsgID:     internalMsg.ID,
		Data:      data,
		NextRetry: time.Now().UTC(),
		SentAt:    time.Time{},
		Retries:   0,
		State:     mongodb.StateWaiting.String(),
	}

	_, err = testerApp.kafkaOutboxCollection.InsertOne(ctx, msg)
	require.NoError(t, err)

	// sleep a bit, waiting for message to get picked up and sent
	time.Sleep(10 * time.Second)

	entries := getAllKafkaDBEntries(ctx, testerApp.kafkaOutboxCollection)
	require.Len(t, entries, 1)
	require.Equal(t, mongodb.StateFinished.String(), entries[0].State)
}
