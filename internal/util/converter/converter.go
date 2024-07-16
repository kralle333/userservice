package converter

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"userservice/internal/db"
	"userservice/internal/util/crypto"
	"userservice/internal/util/time"
	"userservice/proto/grpc"
	"userservice/proto/kafkaschema"
)

func CreateUserDBEntry(request *grpc.AddUserRequestUser) db.User {

	id := uuid.New()
	now := time.DBNow()

	// I know that this is not part of the assignment,
	// but I couldn't bring myself to store passwords in plaintext,
	// so I figured this is simple solution to avoid that.
	salt := crypto.GenerateSalt()
	hashedPassword := crypto.GenerateHashedPassword(request.Password, salt)

	return db.User{
		ID:        id.String(),
		FirstName: request.FirstName,
		LastName:  request.LastName,
		Nickname:  request.Nickname,
		Password:  hashedPassword,
		Email:     request.Email,
		Country:   request.Country,
		Salt:      salt,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func FromRepoUserToResponseUser(user db.User) *grpc.ResponseUser {
	return &grpc.ResponseUser{
		Id:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Country:   user.Country,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}

func FromRepoUserToKafkaAddedUserMessage(user db.User) *kafkaschema.UserAddedMessage {
	return &kafkaschema.UserAddedMessage{
		Id:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Country:   user.Country,
	}
}
