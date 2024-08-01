package user

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"userservice/internal/domain/model"
	"userservice/internal/domain/model/adduser"
	"userservice/internal/domain/model/listusers"
	"userservice/internal/domain/model/updateuser"
	"userservice/internal/util/validation"
)

var (
	ErrRequestedCountryIsNotValid      = errors.New("request country is not valid")
	ErrRequestedEmailIsNotValid        = errors.New("request email is not valid")
	ErrRequestedUserIDIsNotUUID        = errors.New("requested user id is not uuid")
	errUnableToUpdateUserInternalError = errors.New("unable to update user: internal error")
	errUnableToRemoveUserInternalError = errors.New("unable to remove user: internal error")
)

type component struct {
	repo Repo
}

type Component interface {
	AddUser(ctx context.Context, requestUser adduser.Request) (model.User, error)
	RemoveUser(ctx context.Context, userID string) (model.User, error)
	UpdateUser(ctx context.Context, userID string, user updateuser.Request) (model.User, error)
	ListUsers(ctx context.Context, request listusers.Request) (listusers.Response, error)
}

func NewUserComponent(conn Repo) Component {
	return &component{
		repo: conn,
	}
}

func isValidUUID(idString string) bool {
	_, err := uuid.Parse(idString)
	return err == nil
}

func (c *component) AddUser(ctx context.Context, requestUser adduser.Request) (model.User, error) {

	log.Info().Msgf("UserComponent: adding user")

	err := validation.ValidateCountryCode(requestUser.Country)
	if err != nil {
		return model.User{}, ErrRequestedCountryIsNotValid
	}

	if !validation.IsEmailValid(requestUser.Email) {
		return model.User{}, ErrRequestedEmailIsNotValid
	}
	if requestUser.FirstName == "" {
		return model.User{}, errors.New("first name is required")
	}
	if requestUser.LastName == "" {
		return model.User{}, errors.New("last name is required")
	}
	if requestUser.Nickname == "" {
		return model.User{}, errors.New("nickname is required")
	}
	if requestUser.Email == "" {
		return model.User{}, errors.New("email is required")
	}

	if requestUser.Password == "" {
		return model.User{}, errors.New("password is required")
	}
	if requestUser.Country == "" {
		return model.User{}, errors.New("country is required")
	}

	newUser := model.User{
		ID:        uuid.NewString(),
		FirstName: requestUser.FirstName,
		LastName:  requestUser.LastName,
		Nickname:  requestUser.Nickname,
		Password:  requestUser.Password,
		Email:     requestUser.Email,
		Country:   requestUser.Country,
	}

	addUser, err := c.repo.AddUser(ctx, newUser)
	if err != nil {
		return model.User{}, err
	}

	return addUser, nil
}

func (c *component) RemoveUser(ctx context.Context, userID string) (model.User, error) {
	log.Info().Msgf("UserComponent: removing user %s", userID)

	if !isValidUUID(userID) {
		return model.User{}, ErrRequestedUserIDIsNotUUID
	}

	removedUser, err := c.repo.RemoveUser(ctx, userID)
	if err != nil {
		log.Err(err).Msgf("UserComponent: failed to remove user %s", userID)
		return model.User{}, errUnableToRemoveUserInternalError
	}

	return removedUser, nil
}

func (c *component) UpdateUser(ctx context.Context, userID string, user updateuser.Request) (model.User, error) {

	// validate that the requested ID is a valid UUID
	if !isValidUUID(userID) {
		return model.User{}, ErrRequestedUserIDIsNotUUID
	}

	modifiedUser, err := c.repo.UpdateUser(ctx, userID, user)
	if err != nil {
		log.Err(err).Msgf("UserComponent: failed to update user %s", userID)
		return model.User{}, errUnableToUpdateUserInternalError
	}

	return modifiedUser, nil
}

func (c *component) ListUsers(ctx context.Context, request listusers.Request) (listusers.Response, error) {

	users, err := c.repo.ListUsers(ctx, request)
	if err != nil {
		return listusers.Response{}, err
	}

	return users, nil
}
