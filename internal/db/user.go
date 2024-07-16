package db

import (
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrUserNotFound = errors.New("user not found")

type UserDBRepo interface {
	AddUser(ctx context.Context, user User) error
	RemoveUser(ctx context.Context, id string) (User, error)
	UpdateUser(ctx context.Context, userID string, updateFilter bson.D) (User, error)
	GetUser(ctx context.Context, id string) (User, error)
	ListUsers(ctx context.Context, limit int64, findFilter bson.D, sortBy bson.D) (ListUsersResult, error)
}

func (m *MongoDBRepo) AddUser(ctx context.Context, user User) error {
	_, err := m.usersCollection.InsertOne(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepo) RemoveUser(ctx context.Context, id string) (User, error) {
	result := m.usersCollection.FindOneAndDelete(ctx, bson.M{m.config.UserIdName: id})
	user := User{}
	err := result.Decode(&user)
	return user, err
}

func (m *MongoDBRepo) UpdateUser(ctx context.Context, userID string, updateFilter bson.D) (User, error) {

	log.Info().Msgf("updating user %s with filter: %s", userID, updateFilter)

	upsert := false
	returnDocument := options.After
	opt := options.FindOneAndUpdateOptions{
		Upsert:         &upsert,
		ReturnDocument: &returnDocument,
	}

	result := m.usersCollection.FindOneAndUpdate(ctx, bson.M{m.config.UserIdName: userID}, updateFilter, &opt)

	updatedUser := User{}
	err := result.Decode(&updatedUser)
	if err != nil {
		return User{}, err
	}

	return updatedUser, nil
}

func (m *MongoDBRepo) GetUser(ctx context.Context, id string) (User, error) {
	returnedUser := User{}
	err := m.usersCollection.FindOne(ctx, bson.M{m.config.UserIdName: id}).Decode(&returnedUser)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return returnedUser, nil
}

func (m *MongoDBRepo) getUsedLimit(limit int64) int64 {
	if limit == 0 {
		return m.config.ListUserDefaultLimit
	}
	if limit > m.config.ListUserMaxLimit {
		return m.config.ListUserMaxLimit
	}
	return limit
}

func (m *MongoDBRepo) ListUsers(ctx context.Context, limit int64, findFilter bson.D, sortBy bson.D) (ListUsersResult, error) {

	usedLimit := m.getUsedLimit(limit)

	findOptions := options.Find()
	findOptions.SetSort(sortBy)
	findOptions.SetLimit(usedLimit)

	log.Info().Msgf("looking for users using filter: %v", findFilter)

	findResult, err := m.usersCollection.Find(ctx, findFilter, findOptions)
	if err != nil {
		return ListUsersResult{}, err
	}
	defer findResult.Close(ctx)

	returnedListResult := ListUsersResult{}
	for findResult.Next(ctx) {
		userResult := User{}
		err = findResult.Decode(&userResult)
		if err != nil {
			log.Err(err).Msgf("failed to decode user result! might indicate bad entries in database: %v", findResult.Current)
		}
		returnedListResult.Users = append(returnedListResult.Users, userResult)
	}

	if len(returnedListResult.Users) > 0 {
		newCursor := returnedListResult.Users[len(returnedListResult.Users)-1].MongoDBID
		returnedListResult.Cursor = newCursor.Hex()
		returnedListResult.Limit = usedLimit
	}

	return returnedListResult, nil
}
