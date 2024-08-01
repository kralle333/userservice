package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
	"userservice/internal/domain/model"
	"userservice/internal/domain/model/listusers"
	"userservice/internal/domain/model/updateuser"
	"userservice/internal/infrastructure/messaging"
	"userservice/internal/util/crypto"
	"userservice/proto/kafkaschema"
)

var errUserNotFound = errors.New("user not found")

type DBUser struct {
	MongoDBID primitive.ObjectID `bson:"_id,omitempty"`
	ID        string             `bson:"id"`
	FirstName string             `bson:"first_name"`
	LastName  string             `bson:"last_name"`
	Nickname  string             `bson:"nickname"`
	Password  string             `bson:"password"`
	Email     string             `bson:"email"`
	Country   string             `bson:"country"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
	Salt      string             `bson:"salt"`
}

func toDomainUser(user DBUser) model.User {
	return model.User{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nickname:  user.Nickname,
		Password:  user.Password,
		Email:     user.Email,
		Country:   user.Country,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func createNewDBUser(request model.User) DBUser {

	id := uuid.New()
	now := time.Now().UTC()

	salt := crypto.GenerateSalt()
	hashedPassword := crypto.GenerateHashedPassword(request.Password, salt)

	return DBUser{
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

func createKafkaMessage(topic string, key string, data interface{}) (messaging.KafkaInternalMessage, error) {

	kafkaValue, err := json.Marshal(data)
	if err != nil {
		return messaging.KafkaInternalMessage{}, err
	}
	return *messaging.NewInternalMessage(
		topic,
		[]byte(key),
		kafkaValue,
	), nil
}

func (c *Connection) AddUser(ctx context.Context, user model.User) (model.User, error) {

	var addedUser model.User
	// first add the user to the database and then add an outbox message for kafka
	err := c.executeInTransaction(ctx, func(innerContext mongo.SessionContext) error {
		userToAdd := createNewDBUser(user)
		_, innerErr := c.usersCollection.InsertOne(innerContext, userToAdd)
		if innerErr != nil {
			return innerErr
		}
		addedUser = toDomainUser(userToAdd)

		messageToSend, innerErr := createKafkaMessage(c.kafkaConfig.Topics.UserAddedTopicName, userToAdd.ID, &kafkaschema.UserAddedMessage{
			Id:        addedUser.ID,
			FirstName: addedUser.FirstName,
			LastName:  addedUser.LastName,
			Nickname:  addedUser.Nickname,
			Email:     addedUser.Email,
			Country:   addedUser.Country,
		})

		innerErr = c.putKafkaMessageInOutbox(innerContext, messageToSend)
		if innerErr != nil {
			return innerErr
		}

		return nil
	})
	return addedUser, err
}

func (c *Connection) RemoveUser(ctx context.Context, userID string) (model.User, error) {

	var removedUser model.User
	// first remove the user from the database and then add an outbox message for kafka
	err := c.executeInTransaction(ctx, func(innerContext mongo.SessionContext) error {
		result := c.usersCollection.FindOneAndDelete(innerContext, bson.M{c.dbConfig.UserIdName: userID})
		user := DBUser{}
		innerErr := result.Decode(&user)
		if innerErr != nil {
			return innerErr
		}

		messageToSend, innerErr := createKafkaMessage(c.kafkaConfig.Topics.UserRemovedTopicName, userID, &kafkaschema.UserRemovedMessage{
			Id: userID,
		})

		innerErr = c.putKafkaMessageInOutbox(innerContext, messageToSend)
		if innerErr != nil {
			return innerErr
		}
		removedUser = toDomainUser(user)
		return nil
	})
	return removedUser, err
}

func (c *Connection) UpdateUser(ctx context.Context, userID string, updateUser updateuser.Request) (model.User, error) {

	storedUser, err := c.getUserInternal(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.User{}, errUserNotFound
		}
		return model.User{}, err
	}

	filter, err := constructUpdateFilter(updateUser, storedUser.Salt)
	if err != nil {
		return model.User{}, err
	}

	log.Info().Msgf("updating user %s with filter: %s", userID, filter)

	upsert := false
	returnDocument := options.After
	opt := options.FindOneAndUpdateOptions{
		Upsert:         &upsert,
		ReturnDocument: &returnDocument,
	}

	result := c.usersCollection.FindOneAndUpdate(ctx, bson.M{c.dbConfig.UserIdName: userID}, filter, &opt)

	updatedUser := DBUser{}
	err = result.Decode(&updatedUser)
	if err != nil {
		return model.User{}, err
	}

	return toDomainUser(updatedUser), nil
}

func (c *Connection) getUserInternal(ctx context.Context, userID string) (DBUser, error) {
	returnedUser := DBUser{}
	err := c.usersCollection.FindOne(ctx, bson.M{c.dbConfig.UserIdName: userID}).Decode(&returnedUser)
	if err != nil {

		return DBUser{}, err
	}
	return returnedUser, nil
}

func (c *Connection) GetUser(ctx context.Context, id string) (model.User, error) {
	user, err := c.getUserInternal(ctx, id)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.User{}, errUserNotFound
		}
		return model.User{}, err
	}
	return toDomainUser(user), nil
}

func (c *Connection) getUsedLimit(limit int64) int64 {
	if limit == 0 {
		return c.dbConfig.ListUserDefaultLimit
	}
	if limit > c.dbConfig.ListUserMaxLimit {
		return c.dbConfig.ListUserMaxLimit
	}
	return limit
}

func (c *Connection) ListUsers(ctx context.Context, request listusers.Request) (listusers.Response, error) {

	limit := int64(0)
	rawCursor := ""
	if request.Paging != nil {
		limit = request.Paging.Limit
		rawCursor = request.Paging.Cursor
	}

	filter := constructListUsersFilter(request.Filtering, request.Sorting, rawCursor)
	sortBy := constructListUsersSortBy(request.Sorting)

	usedLimit := c.getUsedLimit(limit)

	findOptions := options.Find()
	findOptions.SetSort(sortBy)
	findOptions.SetLimit(usedLimit)

	log.Info().Msgf("looking for users using filter: %v", filter)

	findResult, err := c.usersCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return listusers.Response{}, err
	}
	defer findResult.Close(ctx)

	var dbUsers []DBUser
	for findResult.Next(ctx) {
		userResult := DBUser{}
		err = findResult.Decode(&userResult)
		if err != nil {
			log.Err(err).Msgf("failed to decode user result! might indicate bad entries in database: %v", findResult.Current)
		}
		dbUsers = append(dbUsers, userResult)
	}

	returnedListResult := listusers.Response{}
	if len(dbUsers) > 0 {
		returnedListResult.Next.Limit = usedLimit
		for _, user := range dbUsers {
			returnedListResult.Users = append(returnedListResult.Users, toDomainUser(user))
		}
		lastUser := dbUsers[len(dbUsers)-1]
		returnedListResult.Next.Cursor = constructCursor(lastUser, request.Sorting)
	}

	return returnedListResult, nil
}
