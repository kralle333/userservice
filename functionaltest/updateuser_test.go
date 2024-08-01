package functionaltest

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"testing"
	"time"
	"userservice/internal/infrastructure/mongodb"
	"userservice/internal/util/crypto"
	proto "userservice/proto/grpc"
)

func pointerString(s string) *string { return &s }

func TestUpdateUser(t *testing.T) {
	ctx := context.Background()
	defer testerApp.clearDB()

	userID := uuid.New().String()
	testUser := mongodb.DBUser{
		ID:        userID,
		FirstName: "Tylar",
		LastName:  "Jewel",
		Nickname:  "dollar",
		Password:  "randomPassword",
		Email:     "tylerJ@example.com",
		Country:   "YE",
		Salt:      crypto.GenerateSalt(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	coll := testerApp.dbConn.Database("functional").Collection("user")
	_, err := coll.InsertOne(ctx, testUser)
	require.NoError(t, err)

	modifiedUser := proto.UpdateUserRequestUser{
		FirstName: pointerString("Rain"),
		LastName:  pointerString("Kinsley"),
		Nickname:  pointerString("Regn"),
		Email:     pointerString("rk@example.com"),
		Password:  pointerString("Password1234!"),
		Country:   pointerString("DK"),
	}

	resp, err := testerApp.grpcClient.UpdateUser(ctx, &proto.UpdateUserRequest{
		UserID: userID,
		User:   &modifiedUser,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.User)

	respUser := resp.User
	require.Equal(t, userID, respUser.Id)
	require.Equal(t, *modifiedUser.Country, respUser.Country)
	require.Equal(t, *modifiedUser.Nickname, respUser.Nickname)
	require.Equal(t, *modifiedUser.FirstName, respUser.FirstName)
	require.Equal(t, *modifiedUser.LastName, respUser.LastName)
	require.Equal(t, *modifiedUser.Email, respUser.Email)

	var userInDB mongodb.DBUser
	err = coll.FindOne(ctx, bson.M{"id": respUser.Id}).Decode(&userInDB)
	require.NoError(t, err)
	require.Equal(t, userID, userInDB.ID)
	require.Equal(t, *modifiedUser.Country, userInDB.Country)
	require.Equal(t, *modifiedUser.Nickname, userInDB.Nickname)
	require.Equal(t, *modifiedUser.FirstName, userInDB.FirstName)
	require.Equal(t, *modifiedUser.LastName, userInDB.LastName)
	require.Equal(t, *modifiedUser.Email, userInDB.Email)

	// nil request
	resp, err = testerApp.grpcClient.UpdateUser(ctx, &proto.UpdateUserRequest{
		UserID: userID,
		User:   nil,
	})
	require.Error(t, err)
	require.Nil(t, resp)

	// empty request
	resp, err = testerApp.grpcClient.UpdateUser(ctx, &proto.UpdateUserRequest{
		UserID: userID,
		User:   &proto.UpdateUserRequestUser{},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

}
