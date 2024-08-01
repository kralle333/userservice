package user

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"userservice/internal/domain/model"
	"userservice/internal/domain/model/adduser"
	"userservice/internal/domain/model/listusers"
	"userservice/internal/mock"
	timeutil "userservice/internal/util/time"
)

func getSuccessfulUserRequest() adduser.Request {

	return adduser.Request{
		FirstName: "John",
		LastName:  "Smith",
		Nickname:  "Security Expert",
		Email:     "john@example.com",
		Password:  "superSecurePassword",
		Country:   "GB",
	}
}

// / ADDING USERS
// ///////////////

// TODO: Figure out how to write tests with mocked transaction
func TestSuccessAddUser(t *testing.T) {
	mockUserRepo := mock.NewUserRepoMock()
	c := NewUserComponent(mockUserRepo)

	addedUser, err := c.AddUser(context.Background(), getSuccessfulUserRequest())
	require.NoError(t, err)

	parsedUUID, err := uuid.Parse(addedUser.ID)
	require.NoError(t, err)

	user, err := mockUserRepo.GetUser(context.Background(), parsedUUID.String())
	require.NoError(t, err)
	require.Equal(t, addedUser.FirstName, user.FirstName)

	require.Equal(t, len(mockUserRepo.Users), 1)

}

func TestFailAddUserWithBadCountryCode(t *testing.T) {
	mockUserRepo := mock.NewUserRepoMock()
	c := NewUserComponent(mockUserRepo)

	u := getSuccessfulUserRequest()
	u.Country = "DENMARK"
	_, err := c.AddUser(context.Background(), u)
	require.Error(t, err)
}
func TestFailAddUserWithBadEmail(t *testing.T) {
	mockUserRepo := mock.NewUserRepoMock()
	c := NewUserComponent(mockUserRepo)

	u := getSuccessfulUserRequest()
	u.Email = "foo@bar"
	_, err := c.AddUser(context.Background(), u)
	require.Error(t, err)
}
func TestFailEmptyField(t *testing.T) {
	mockUserRepo := mock.NewUserRepoMock()
	c := NewUserComponent(mockUserRepo)

	u := getSuccessfulUserRequest()
	u.Nickname = ""
	_, err := c.AddUser(context.Background(), u)
	require.Error(t, err)

	u = getSuccessfulUserRequest()
	u.FirstName = ""
	_, err = c.AddUser(context.Background(), u)
	require.Error(t, err)

	u = getSuccessfulUserRequest()
	u.LastName = ""
	_, err = c.AddUser(context.Background(), u)
	require.Error(t, err)

	u = getSuccessfulUserRequest()
	u.Password = ""
	_, err = c.AddUser(context.Background(), u)
	require.Error(t, err)

	u = getSuccessfulUserRequest()
	u.Email = ""
	_, err = c.AddUser(context.Background(), u)
	require.Error(t, err)

	u = getSuccessfulUserRequest()
	u.Country = ""
	_, err = c.AddUser(context.Background(), u)
	require.Error(t, err)
}

/// REMOVING USERS
/////////////////

func TestSuccessRemoveUser(t *testing.T) {
	mockUserRepo := mock.NewUserRepoMock()
	c := NewUserComponent(mockUserRepo)

	userToRemove, err := c.AddUser(context.Background(), getSuccessfulUserRequest())
	require.NoError(t, err)

	// insert other users to make it a bit more interesting
	otherUsersToAdd := 3
	for range otherUsersToAdd {
		_, err = c.AddUser(context.Background(), getSuccessfulUserRequest())
		require.NoError(t, err)
	}

	parsedUUID, err := uuid.Parse(userToRemove.ID)
	require.NoError(t, err)

	removedUser, err := c.RemoveUser(context.Background(), parsedUUID.String())
	require.NoError(t, err)
	require.Equal(t, len(mockUserRepo.Users), otherUsersToAdd)
	require.Equal(t, userToRemove.ID, removedUser.ID)

	// TODO: check outbox
}

func TestFailNoUserWithID(t *testing.T) {
	mockUserRepo := mock.NewUserRepoMock()
	c := NewUserComponent(mockUserRepo)

	addedUser, err := c.AddUser(context.Background(), getSuccessfulUserRequest())
	require.NoError(t, err)

	// very unlikely to clash, but doesn't hurt to be aware of the case
	invalidID := uuid.New()
	for invalidID.String() == addedUser.ID {
		invalidID = uuid.New()
	}

	_, err = c.RemoveUser(context.Background(), invalidID.String())
	require.Error(t, err)

}

// / LISTING USERS
// ///////////////

func createDummyDBUser() model.User {
	now := timeutil.DBNow()
	return model.User{
		ID:        uuid.New().String(),
		FirstName: "John",
		LastName:  "Tester",
		Nickname:  "TheTester",
		Password:  "1234!",
		Email:     "test@email.com",
		Country:   "DK",
		CreatedAt: now,
		UpdatedAt: now,
	}

}

func TestSuccessListUsers(t *testing.T) {
	mockUserRepo := mock.NewUserRepoMock()
	mockUserRepo.SetMockListResult([]model.User{
		createDummyDBUser(),
		createDummyDBUser(),
		createDummyDBUser(),
	})
	c := NewUserComponent(mockUserRepo)

	_, err := c.ListUsers(context.Background(), listusers.Request{
		Paging: &listusers.PageInfo{
			Cursor: "",
			Limit:  25,
		},
	})
	require.NoError(t, err)
}
