package functionaltest

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"k8s.io/apimachinery/pkg/util/rand"
	"testing"
	"userservice/internal/infrastructure/mongodb"
	proto "userservice/proto/grpc"
)

func TestAddUser(t *testing.T) {
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
	respUser := user.User

	// DB STATE
	coll := testerApp.dbConn.Database("functional").Collection("user")

	var userInDB mongodb.DBUser
	err = coll.FindOne(ctx, bson.M{"id": respUser.Id}).Decode(&userInDB)
	require.NoError(t, err)
	require.Equal(t, userInDB.ID, respUser.Id)
	require.Equal(t, userInDB.Country, respUser.Country)
	require.Equal(t, userInDB.Nickname, respUser.Nickname)
	require.Equal(t, userInDB.FirstName, respUser.FirstName)
	require.Equal(t, userInDB.LastName, respUser.LastName)
	require.Equal(t, userInDB.Email, respUser.Email)

}
