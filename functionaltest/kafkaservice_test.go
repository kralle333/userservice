//go:build needs_kafka
// +build needs_kafka

package functionaltest

import (
	"FACEITBackendTest/internal/db"
	"FACEITBackendTest/internal/util/crypto"
	proto "FACEITBackendTest/proto/grpc"
	"FACEITBackendTest/proto/kafkaschema"
	"context"
	"encoding/json"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
	"testing"
	"time"
)

func TestKafkaServiceAddUser(t *testing.T) {
	ctx := context.Background()
	defer testerApp.clearDB()

	// KAFKA STATE
	waitingChannel := make(chan kafka.Message)
	testerApp.consumer.setCommunicationChannel(waitingChannel)
	defer testerApp.consumer.clearCommunicationChannel()

	// to make it easier for me to debug
	randomNickname := rand.String(6)
	requestUser := &proto.AddUserRequestUser{
		FirstName: "Jessica",
		LastName:  "Testerson",
		Nickname:  randomNickname,
		Email:     "jess@example.com",
		Password:  "superDuper",
		Country:   "DK",
	}

	user, err := testerApp.grpcClient.AddUser(ctx, &proto.AddUserRequest{
		User: requestUser,
	})
	require.NoError(t, err)
	respUser := user.User

	timeOutTimer := time.NewTimer(5 * time.Second)
	select {
	case <-timeOutTimer.C:
		require.FailNow(t, "timed out waiting for kafka message to arrive!")
	case msg := <-waitingChannel:
		messageTopic := *msg.TopicPartition.Topic
		require.Equal(t, testerApp.serverConfig.Kafka.UserAddedTopicName, messageTopic)
		var parsed kafkaschema.UserAddedMessage
		err := json.Unmarshal(msg.Value, &parsed)
		require.NoError(t, err)
		require.Equal(t, parsed.Id, respUser.Id)
		require.Equal(t, parsed.Country, respUser.Country)
		require.Equal(t, parsed.Nickname, respUser.Nickname)
		require.Equal(t, parsed.FirstName, respUser.FirstName)
		require.Equal(t, parsed.LastName, respUser.LastName)
		require.Equal(t, parsed.Email, respUser.Email)
	}
}

func TestKafkaServiceRemoveUser(t *testing.T) {
	ctx := context.Background()
	defer testerApp.clearDB()

	waitingChannel := make(chan kafka.Message)
	testerApp.consumer.setCommunicationChannel(waitingChannel)
	defer testerApp.consumer.clearCommunicationChannel()

	// setting up data for later removal
	testUser := db.User{
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

	user, err := testerApp.grpcClient.RemoveUser(ctx, &proto.RemoveUserRequest{
		UserID: testUser.ID,
	})
	require.NoError(t, err)
	respUser := user.User

	// KAFKA STATE
	timeOutTimer := time.NewTimer(5 * time.Second)
	select {
	case <-timeOutTimer.C:
		require.FailNow(t, "timed out waiting for kafka message to arrive!")
	case msg := <-waitingChannel:
		messageTopic := *msg.TopicPartition.Topic
		require.Equal(t, testerApp.serverConfig.Kafka.UserRemovedTopicName, messageTopic)
		var parsed kafkaschema.UserRemovedMessage
		err := json.Unmarshal(msg.Value, &parsed)
		require.NoError(t, err)
		require.Equal(t, parsed.Id, respUser.Id)
	}
}
