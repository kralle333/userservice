package user

import (
	"FACEITBackendTest/internal/config"
	"FACEITBackendTest/internal/db"
	"FACEITBackendTest/internal/mock"
	"FACEITBackendTest/internal/util/crypto"
	timeutil "FACEITBackendTest/internal/util/time"
	"FACEITBackendTest/proto/grpc"
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func getSuccessfulUserRequest() *grpc.AddUserRequestUser {

	return &grpc.AddUserRequestUser{
		FirstName: "John",
		LastName:  "Smith",
		Nickname:  "Security Expert",
		Email:     "john@example.com",
		Password:  "superSecurePassword",
		Country:   "GB",
	}
}

func getMocks() (*mock.UserRepoMock, db.TransactionExecutor, db.KafkaMessageSender) {
	return mock.NewUserRepoMock(), mock.NewTransactionExecutorMock(), mock.NewKafkaFailStoreMock()
}

var mockConfig = config.AppConfig{}

// / ADDING USERS
// ///////////////

// TODO: Figure out how to write tests with mocked transaction
func TestSuccessAddUser(t *testing.T) {
	t.Skip("missing implementation")
	mockUserRepo, mockExecutor, mockKafkaMessageSender := getMocks()
	c := NewUserComponent(mockUserRepo, mockExecutor, mockKafkaMessageSender, mockConfig.Kafka)

	addedUser, err := c.AddUser(context.Background(), getSuccessfulUserRequest())
	require.NoError(t, err)

	parsedUUID, err := uuid.Parse(addedUser.Id)
	require.NoError(t, err)

	user, err := mockUserRepo.GetUser(context.Background(), parsedUUID.String())
	require.NoError(t, err)
	require.Equal(t, addedUser.FirstName, user.FirstName)

	require.Equal(t, len(mockUserRepo.Users), 1)

}

func TestFailAddUserWithBadCountryCode(t *testing.T) {
	mockUserRepo, mockExecutor, mockKafkaMessageSender := getMocks()
	c := NewUserComponent(mockUserRepo, mockExecutor, mockKafkaMessageSender, mockConfig.Kafka)

	u := getSuccessfulUserRequest()
	u.Country = "DENMARK"
	_, err := c.AddUser(context.Background(), u)
	require.Error(t, err)
}
func TestFailAddUserWithBadEmail(t *testing.T) {
	mockUserRepo, mockExecutor, mockKafkaMessageSender := getMocks()
	c := NewUserComponent(mockUserRepo, mockExecutor, mockKafkaMessageSender, mockConfig.Kafka)

	u := getSuccessfulUserRequest()
	u.Email = "foo@bar"
	_, err := c.AddUser(context.Background(), u)
	require.Error(t, err)
}
func TestFailEmptyField(t *testing.T) {
	mockUserRepo, mockExecutor, mockKafkaMessageSender := getMocks()
	c := NewUserComponent(mockUserRepo, mockExecutor, mockKafkaMessageSender, mockConfig.Kafka)

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

// TODO: Figure out how to write tests with mocked transaction
func TestSuccessRemoveUser(t *testing.T) {
	t.Skip("missing implementation")
	mockUserRepo, mockExecutor, mockKafkaMessageSender := getMocks()
	c := NewUserComponent(mockUserRepo, mockExecutor, mockKafkaMessageSender, config.KafkaConfig{})

	userToRemove, err := c.AddUser(context.Background(), getSuccessfulUserRequest())
	require.NoError(t, err)

	// insert other users to make it a bit more interesting
	otherUsersToAdd := 3
	for range otherUsersToAdd {
		_, err = c.AddUser(context.Background(), getSuccessfulUserRequest())
		require.NoError(t, err)
	}

	parsedUUID, err := uuid.Parse(userToRemove.Id)
	require.NoError(t, err)

	removedUser, err := c.RemoveUser(context.Background(), parsedUUID.String())
	require.NoError(t, err)
	require.Equal(t, len(mockUserRepo.Users), otherUsersToAdd)
	require.Equal(t, userToRemove.Id, removedUser.Id)

	// TODO: check outbox
}

// TODO: Figure out how to write tests with mocked transaction
func TestFailNoUserWithID(t *testing.T) {
	t.Skip("missing implementation")
	mockUserRepo, mockExecutor, mockKafkaMessageSender := getMocks()
	c := NewUserComponent(mockUserRepo, mockExecutor, mockKafkaMessageSender, mockConfig.Kafka)

	addedUser, err := c.AddUser(context.Background(), getSuccessfulUserRequest())
	require.NoError(t, err)

	// very unlikely to clash, but doesn't hurt to be aware of the case
	invalidID := uuid.New()
	for invalidID.String() == addedUser.Id {
		invalidID = uuid.New()
	}

	user, err := c.RemoveUser(context.Background(), invalidID.String())
	require.Error(t, err)
	require.Nil(t, user)
}

// / LISTING USERS
// ///////////////

func createDummyDBUser() db.User {
	now := timeutil.DBNow()
	return db.User{
		ID:        uuid.New().String(),
		FirstName: "John",
		LastName:  "Tester",
		Nickname:  "TheTester",
		Password:  "1234!",
		Email:     "test@email.com",
		Country:   "DK",
		Salt:      crypto.GenerateSalt(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestSuccessListUsers(t *testing.T) {
	mockUserRepo, mockExecutor, mockKafkaMessageSender := getMocks()
	mockUserRepo.SetMockListResult([]db.User{
		createDummyDBUser(),
		createDummyDBUser(),
		createDummyDBUser(),
	})
	c := NewUserComponent(mockUserRepo, mockExecutor, mockKafkaMessageSender, mockConfig.Kafka)

	_, err := c.ListUsers(context.Background(), &grpc.ListUsersRequest{
		Paging: &grpc.PageInfo{
			Cursor: "",
			Limit:  25,
		},
	})
	require.NoError(t, err)
}
