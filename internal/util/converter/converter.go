package converter

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"userservice/internal/domain/model"
	"userservice/internal/domain/model/listusers"
	"userservice/proto/grpc"
	"userservice/proto/kafkaschema"
)

func FromDomainUserToResponseUser(user model.User) *grpc.ResponseUser {
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

func FromListUsersToResponseListUsers(response listusers.Response) *grpc.ListUsersResponse {

	var grpcUsers []*grpc.ResponseUser

	for _, user := range response.Users {
		grpcUsers = append(grpcUsers, FromDomainUserToResponseUser(user))
	}

	return &grpc.ListUsersResponse{
		Next: &grpc.PageInfo{
			Limit:  response.Next.Limit,
			Cursor: response.Next.Cursor,
		},
		Users: grpcUsers,
	}
}

func FromRepoUserToKafkaAddedUserMessage(user model.User) *kafkaschema.UserAddedMessage {
	return &kafkaschema.UserAddedMessage{
		Id:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Country:   user.Country,
	}
}
