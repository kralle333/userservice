package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"userservice/internal/config"
	"userservice/internal/db"
	"userservice/internal/kafkamessage"
	"userservice/internal/util/converter"
	"userservice/internal/util/mongodb"
	"userservice/internal/util/validation"
	"userservice/proto/grpc"
	"userservice/proto/kafkaschema"
)

var (
	ErrRequestedUserIDIsNotUUID = errors.New("requested user id is not uuid")
)

type component struct {
	conn                db.UserDBRepo
	kafkaMessageOutbox  db.KafkaMessageSender
	transactionExecutor db.TransactionExecutor
	kafkaConfig         config.KafkaConfig
}

type Component interface {
	AddUser(ctx context.Context, requestUser *grpc.AddUserRequestUser) (*grpc.ResponseUser, error)
	RemoveUser(ctx context.Context, userID string) (*grpc.ResponseUser, error)
	ModifyUser(ctx context.Context, userID string, user *grpc.ModifyUserRequestUser) (*grpc.ResponseUser, error)
	ListUsers(ctx context.Context, request *grpc.ListUsersRequest) (*grpc.ListUsersResponse, error)
}

func NewUserComponent(conn db.UserDBRepo, executor db.TransactionExecutor, kafkaMessageSender db.KafkaMessageSender, kafkaConfig config.KafkaConfig) Component {
	return &component{
		conn:                conn,
		transactionExecutor: executor,
		kafkaMessageOutbox:  kafkaMessageSender,
		kafkaConfig:         kafkaConfig,
	}
}

func validateAddUserRequest(user *grpc.AddUserRequestUser) error {
	if user.FirstName == "" {
		return errors.New("first name is required")
	}
	if user.LastName == "" {
		return errors.New("last name is required")
	}
	if user.Nickname == "" {
		return errors.New("nickname is required")
	}
	if user.Email == "" {
		return errors.New("email is required")
	}

	if user.Password == "" {
		return errors.New("password is required")
	}
	if user.Country == "" {
		return errors.New("country is required")
	}

	if !validation.IsEmailValid(user.Email) {
		return errors.New("email is invalid")
	}

	return validation.ValidateCountryCode(user.Country)
}

func isValidUUID(idString string) bool {
	_, err := uuid.Parse(idString)
	return err == nil
}

func (c *component) putKafkaMessageInOutbox(ctx context.Context, messageToSend kafkamessage.InternalMessage) error {

	log.Info().Msgf("UserComponent: putting kafka message in outbox %s", messageToSend.TopicID)
	err := c.kafkaMessageOutbox.PutMessageInOutbox(ctx, messageToSend)
	if err != nil {
		return err
	}
	return nil
}

func createKafkaMessage(topic string, key string, data interface{}) (kafkamessage.InternalMessage, error) {

	kafkaValue, err := json.Marshal(data)
	if err != nil {
		return kafkamessage.InternalMessage{}, err
	}
	// use id as key to ensure correct ordering of user events
	return *kafkamessage.NewInternalMessage(
		topic,
		[]byte(key),
		kafkaValue,
	), nil
}

func (c *component) AddUser(ctx context.Context, requestUser *grpc.AddUserRequestUser) (*grpc.ResponseUser, error) {

	log.Info().Msgf("UserComponent: adding user")
	err := validateAddUserRequest(requestUser)
	if err != nil {
		return nil, err
	}

	var newUser db.User

	// first add the user to the database and then add an outbox message for kafka
	err = c.transactionExecutor.ExecuteInTransaction(ctx, func(sessionContext mongo.SessionContext) error {
		newUser = converter.CreateUserDBEntry(requestUser)
		err = c.conn.AddUser(sessionContext, newUser)
		if err != nil {
			return err
		}

		messageToSend, err := createKafkaMessage(c.kafkaConfig.UserAddedTopicName, newUser.ID, converter.FromRepoUserToKafkaAddedUserMessage(newUser))
		err = c.putKafkaMessageInOutbox(sessionContext, messageToSend)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return converter.FromRepoUserToResponseUser(newUser), nil
}

func (c *component) RemoveUser(ctx context.Context, userID string) (*grpc.ResponseUser, error) {
	log.Info().Msgf("UserComponent: removing user %s", userID)

	if !isValidUUID(userID) {
		return nil, ErrRequestedUserIDIsNotUUID
	}

	var err error
	var removedUser db.User

	// first remove the user from the database and then add an outbox message for kafka
	err = c.transactionExecutor.ExecuteInTransaction(ctx, func(sessionContext mongo.SessionContext) error {
		removedUser, err = c.conn.RemoveUser(sessionContext, userID)
		if err != nil {
			return err
		}

		messageToSend, err := createKafkaMessage(c.kafkaConfig.UserRemovedTopicName, userID, &kafkaschema.UserRemovedMessage{
			Id: userID,
		})

		err = c.putKafkaMessageInOutbox(sessionContext, messageToSend)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return converter.FromRepoUserToResponseUser(removedUser), nil
}

func (c *component) ModifyUser(ctx context.Context, userID string, user *grpc.ModifyUserRequestUser) (*grpc.ResponseUser, error) {

	// validate that the requested ID is a valid UUID
	if !isValidUUID(userID) {
		return nil, ErrRequestedUserIDIsNotUUID
	}

	storedUser, err := c.conn.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, errors.New(fmt.Sprintf("user with id %s not found", userID))
		}
		return nil, err
	}

	filter, err := mongodb.ConstructUpdateFilter(user, storedUser.Salt)
	if err != nil {
		return nil, err
	}

	modifiedUser, err := c.conn.UpdateUser(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	return converter.FromRepoUserToResponseUser(modifiedUser), nil
}

func (c *component) ListUsers(ctx context.Context, request *grpc.ListUsersRequest) (*grpc.ListUsersResponse, error) {

	limit := int64(0)
	rawCursor := ""
	if request.Paging != nil {
		limit = request.Paging.Limit
		rawCursor = request.Paging.Cursor
	}

	filter := mongodb.ConstructListUsersFilter(request.Filtering, request.Sorting, rawCursor)
	sortBy := mongodb.ConstructListUsersSortBy(request.Sorting)

	usersResult, err := c.conn.ListUsers(ctx, limit, filter, sortBy)
	if err != nil {
		return nil, err
	}

	respUsers := make([]*grpc.ResponseUser, len(usersResult.Users))
	for i, user := range usersResult.Users {
		respUsers[i] = converter.FromRepoUserToResponseUser(user)
	}

	nextPage := grpc.PageInfo{}
	if len(usersResult.Users) > 0 {
		lastUser := usersResult.Users[len(usersResult.Users)-1]
		if request.Sorting != nil && request.Sorting.By != grpc.UserField_USER_FIELD_UNSPECIFIED {
			nextPage.Cursor = mongodb.ConstructCursor(lastUser, request.Sorting.By)
		} else {
			nextPage.Cursor = fmt.Sprintf(":%s", lastUser)
		}

		nextPage.Limit = limit
	}

	return &grpc.ListUsersResponse{Users: respUsers, Next: &nextPage}, nil
}
