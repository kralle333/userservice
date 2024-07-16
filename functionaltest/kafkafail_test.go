package functionaltest

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
	"testing"
	"userservice/internal/db"
	"userservice/internal/kafkamessage"
	proto "userservice/proto/grpc"
	kafkaproto "userservice/proto/kafkaschema"
)

func TestKafkaFailStorage(t *testing.T) {
	ctx := context.Background()
	defer testerApp.clearDB()

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

	coll := testerApp.kafkaOutboxCollection
	messages := getAllKafkaDBEntries(ctx, coll)
	require.Len(t, messages, 1)
	require.Equal(t, db.KafkaOutboxMessageStateWaiting.String(), messages[0].State)

	var kafkaMessage kafkamessage.InternalMessage
	err = json.Unmarshal(messages[0].Data, &kafkaMessage)
	require.NoError(t, err)
	require.Equal(t, testerApp.serverConfig.Kafka.UserAddedTopicName, kafkaMessage.TopicID)
	require.Equal(t, user.User.Id, string(kafkaMessage.Key))
	var userAddedMessage kafkaproto.UserAddedMessage
	err = json.Unmarshal(kafkaMessage.Value, &userAddedMessage)
	require.NoError(t, err)

	require.Equal(t, user.User.Id, userAddedMessage.Id)
	require.Equal(t, user.User.FirstName, userAddedMessage.FirstName)
	require.Equal(t, user.User.LastName, userAddedMessage.LastName)
	require.Equal(t, user.User.Nickname, userAddedMessage.Nickname)
	require.Equal(t, user.User.Country, userAddedMessage.Country)
	require.Equal(t, user.User.Email, userAddedMessage.Email)
}
