package internal

import (
	"context"
	"errors"
	"userservice/internal/user"
	"userservice/proto/grpc"
)

var ErrRequestIsRequired = errors.New("request is required")
var ErrModifiedUserRequestIsRequired = errors.New("user to be modified is required")

type server struct {
	grpc.UserServiceServer
	userComponent user.Component
}

func newServer(usersComponent user.Component) *server {
	return &server{userComponent: usersComponent}
}

func (s server) AddUser(ctx context.Context, request *grpc.AddUserRequest) (*grpc.AddUserResponse, error) {
	if request == nil {
		return nil, ErrRequestIsRequired
	}

	responseUser, err := s.userComponent.AddUser(ctx, request.User)
	if err != nil {
		return nil, err
	}

	return &grpc.AddUserResponse{User: responseUser}, nil
}

func (s server) RemoveUser(ctx context.Context, request *grpc.RemoveUserRequest) (*grpc.RemoveUserResponse, error) {
	if request == nil {
		return nil, ErrRequestIsRequired
	}

	removedUser, err := s.userComponent.RemoveUser(ctx, request.UserID)
	if err != nil {
		return nil, err
	}

	return &grpc.RemoveUserResponse{User: removedUser}, nil
}

func (s server) ModifyUser(ctx context.Context, request *grpc.ModifyUserRequest) (*grpc.ModifyUserResponse, error) {
	if request == nil {
		return nil, ErrRequestIsRequired
	}

	if request.User == nil {
		return nil, ErrModifiedUserRequestIsRequired
	}

	modifiedUser, err := s.userComponent.ModifyUser(ctx, request.UserID, request.User)
	if err != nil {
		return nil, err
	}
	return &grpc.ModifyUserResponse{User: modifiedUser}, nil
}

func (s server) ListUsers(ctx context.Context, request *grpc.ListUsersRequest) (*grpc.ListUsersResponse, error) {
	if request == nil {
		return nil, ErrRequestIsRequired
	}

	userResponse, err := s.userComponent.ListUsers(ctx, request)
	if err != nil {
		return nil, err
	}

	return userResponse, nil
}
