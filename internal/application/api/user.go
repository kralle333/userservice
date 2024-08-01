package api

import (
	"context"
	"errors"
	"userservice/internal/domain/model/updateuser"
	"userservice/internal/domain/user"
	"userservice/internal/util/converter"
	"userservice/proto/grpc"
)

var ErrRequestIsRequired = errors.New("request is required")
var ErrModifiedUserRequestIsRequired = errors.New("user to be modified is required")

type UserController struct {
	grpc.UserServiceServer
	userComponent user.Component
}

func NewUserController(userComponent user.Component) *UserController {
	return &UserController{userComponent: userComponent}
}

func (s UserController) AddUser(ctx context.Context, request *grpc.AddUserRequest) (*grpc.AddUserResponse, error) {
	if request == nil {
		return nil, ErrRequestIsRequired
	}

	addedUser, err := s.userComponent.AddUser(ctx, converter.ToAddUserRequest(request))
	if err != nil {
		return nil, err
	}
	responseUser := converter.FromDomainUserToResponseUser(addedUser)
	return &grpc.AddUserResponse{User: responseUser}, nil
}

func (s UserController) RemoveUser(ctx context.Context, request *grpc.RemoveUserRequest) (*grpc.RemoveUserResponse, error) {
	if request == nil {
		return nil, ErrRequestIsRequired
	}

	removedUser, err := s.userComponent.RemoveUser(ctx, request.UserID)
	if err != nil {
		return nil, err
	}

	return &grpc.RemoveUserResponse{User: converter.FromDomainUserToResponseUser(removedUser)}, nil
}

func (s UserController) UpdateUser(ctx context.Context, request *grpc.UpdateUserRequest) (*grpc.UpdateUserResponse, error) {
	if request == nil {
		return nil, ErrRequestIsRequired
	}

	if request.User == nil {
		return nil, ErrModifiedUserRequestIsRequired
	}

	modifiedUser, err := s.userComponent.UpdateUser(ctx, request.UserID, updateuser.Request{
		FirstName: request.User.FirstName,
		LastName:  request.User.LastName,
		Nickname:  request.User.Nickname,
		Email:     request.User.Email,
		Password:  request.User.Password,
		Country:   request.User.Country,
	})
	if err != nil {
		return nil, err
	}
	responseUser := converter.FromDomainUserToResponseUser(modifiedUser)
	return &grpc.UpdateUserResponse{User: responseUser}, nil
}

func (s UserController) ListUsers(ctx context.Context, request *grpc.ListUsersRequest) (*grpc.ListUsersResponse, error) {
	input := converter.ConvertListUsersRequest(request)
	if input == nil {
		return nil, ErrRequestIsRequired
	}

	userResponse, err := s.userComponent.ListUsers(ctx, *input)
	if err != nil {
		return nil, err
	}

	return converter.FromListUsersToResponseListUsers(userResponse), nil
}
