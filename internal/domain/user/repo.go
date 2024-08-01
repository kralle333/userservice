package user

import (
	"context"
	"userservice/internal/domain/model"
	"userservice/internal/domain/model/listusers"
	"userservice/internal/domain/model/updateuser"
)

type Repo interface {
	AddUser(ctx context.Context, user model.User) (model.User, error)
	RemoveUser(ctx context.Context, id string) (model.User, error)
	UpdateUser(ctx context.Context, userID string, updateUser updateuser.Request) (model.User, error)
	GetUser(ctx context.Context, id string) (model.User, error)
	ListUsers(ctx context.Context, listRequest listusers.Request) (listusers.Response, error)
}
