package mock

import (
	"context"
	"errors"
	"userservice/internal/domain/model"
	"userservice/internal/domain/model/listusers"
	"userservice/internal/domain/model/updateuser"
)

const userNotFoundIndex = -1

type UserRepoMock struct {
	Users         []model.User
	mockListUsers []model.User
}

func (u *UserRepoMock) AddUser(ctx context.Context, user model.User) (model.User, error) {
	u.Users = append(u.Users, user)
	return user, nil
}

func (u *UserRepoMock) UpdateUser(ctx context.Context, userID string, updateUser updateuser.Request) (model.User, error) {
	storedUserIndex := userNotFoundIndex
	for i, storedUser := range u.Users {
		if storedUser.ID != userID {
			continue
		}
		storedUserIndex = i
	}
	if storedUserIndex == userNotFoundIndex {
		return model.User{}, errors.New("user not found")
	}
	return model.User{}, nil
}

func NewUserRepoMock() *UserRepoMock {
	return &UserRepoMock{
		Users: []model.User{},
	}
}

func (u *UserRepoMock) SetMockListResult(result []model.User) {
	u.mockListUsers = result
}

func (u *UserRepoMock) RemoveUser(ctx context.Context, id string) (model.User, error) {
	for i := len(u.Users) - 1; i >= 0; i-- {
		if u.Users[i].ID != id {
			continue
		}
		toRemove := u.Users[i]
		u.Users = append(u.Users[:i], u.Users[i+1:]...)
		return toRemove, nil
	}
	return model.User{}, errors.New("user not found")
}

func (u *UserRepoMock) GetUser(ctx context.Context, id string) (model.User, error) {

	for _, storedUser := range u.Users {
		if storedUser.ID != id {
			continue
		}
		return storedUser, nil
	}
	return model.User{}, errors.New("user not found")
}

func (u *UserRepoMock) ListUsers(ctx context.Context, listRequest listusers.Request) (listusers.Response, error) {
	users := u.mockListUsers

	return listusers.Response{
		Next: listusers.PageInfo{
			Limit:  listRequest.Paging.Limit,
			Cursor: listRequest.Paging.Cursor,
		},
		Users: users,
	}, nil
}
