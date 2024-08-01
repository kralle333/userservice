package functionaltest

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
	"userservice/internal/infrastructure/messaging"
	"userservice/internal/infrastructure/mongodb"
	"userservice/internal/util/crypto"
	proto "userservice/proto/grpc"
	"userservice/proto/kafkaschema"
)

func TestRemoveUser(t *testing.T) {
	ctx := context.Background()
	defer testerApp.clearDB()

	// setting up data for later removal
	testUser := mongodb.DBUser{
		ID:        uuid.New().String(),
		FirstName: "Hest",
		LastName:  "Petersen",
		Nickname:  "Hesty",
		Password:  "Alalal",
		Email:     "hest@example.com",
		Country:   "SWE",
		Salt:      crypto.GenerateSalt(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	coll := testerApp.dbConn.Database("functional").Collection("user")
	_, err := coll.InsertOne(ctx, testUser)
	require.NoError(t, err)

	entries := getAllUserDBEntries(ctx, coll)
	require.Len(t, entries, 1)
	require.Equal(t, entries[0].ID, testUser.ID)

	// testing removal of user
	resp, err := testerApp.grpcClient.RemoveUser(ctx, &proto.RemoveUserRequest{
		UserID: testUser.ID,
	})
	require.NoError(t, err)
	require.Equal(t, testUser.ID, resp.User.Id)
	entries = getAllUserDBEntries(ctx, coll)
	require.Len(t, entries, 0)

	// removing user again should fail
	_, err = testerApp.grpcClient.RemoveUser(ctx, &proto.RemoveUserRequest{
		UserID: testUser.ID,
	})
	require.Error(t, err)

	outboxEntries := getAllKafkaDBEntries(ctx, testerApp.kafkaOutboxCollection)
	require.Len(t, outboxEntries, 1)
	var msg messaging.KafkaInternalMessage
	err = json.Unmarshal(outboxEntries[0].Data, &msg)
	require.NoError(t, err)
	require.Equal(t, outboxEntries[0].MsgID, msg.ID)
	require.Equal(t, testerApp.serverConfig.Kafka.Topics.UserRemovedTopicName, msg.TopicID)
	require.Equal(t, testUser.ID, string(msg.Key))
	var removedMsg kafkaschema.UserRemovedMessage
	err = json.Unmarshal(msg.Value, &removedMsg)
	require.NoError(t, err)
	require.Equal(t, testUser.ID, removedMsg.Id)
}
