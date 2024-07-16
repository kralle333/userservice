package mock

import (
	"FACEITBackendTest/internal/db"
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
)

const userNotFoundIndex = -1

type UserRepoMock struct {
	Users         []db.User
	mockListUsers []db.User
}

func NewUserRepoMock() *UserRepoMock {
	return &UserRepoMock{
		Users: []db.User{},
	}
}

func (u *UserRepoMock) SetMockListResult(result []db.User) {
	u.mockListUsers = result
}

func (u *UserRepoMock) AddUser(ctx context.Context, user db.User) error {
	u.Users = append(u.Users, user)
	return nil
}

func (u *UserRepoMock) RemoveUser(ctx context.Context, id string) (db.User, error) {
	for i := len(u.Users) - 1; i >= 0; i-- {
		if u.Users[i].ID != id {
			continue
		}
		toRemove := u.Users[i]
		u.Users = append(u.Users[:i], u.Users[i+1:]...)
		return toRemove, nil
	}
	return db.User{}, errors.New("user not found")
}

func (u *UserRepoMock) UpdateUser(ctx context.Context, userID string, updateFilter bson.D) (db.User, error) {
	storedUserIndex := userNotFoundIndex
	for i, storedUser := range u.Users {
		if storedUser.ID != userID {
			continue
		}
		storedUserIndex = i
	}
	if storedUserIndex == userNotFoundIndex {
		return db.User{}, errors.New("user not found")
	}
	return db.User{}, nil
}

func (u *UserRepoMock) GetUser(ctx context.Context, id string) (db.User, error) {

	for _, storedUser := range u.Users {
		if storedUser.ID != id {
			continue
		}
		return storedUser, nil
	}
	return db.User{}, errors.New("user not found")
}

func (u *UserRepoMock) ListUsers(ctx context.Context, limit int64, findFilter bson.D, sortBy bson.D) (db.ListUsersResult, error) {

	users := u.mockListUsers
	return db.ListUsersResult{
		Cursor: users[len(users)-1].MongoDBID.Hex(),
		Limit:  limit,
		Users:  users,
	}, nil
}
