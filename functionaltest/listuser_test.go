package functionaltest

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sort"
	"testing"
	"time"
	"userservice/internal/infrastructure/mongodb"
	"userservice/internal/util/crypto"
	proto "userservice/proto/grpc"
)

func TestListUsers(t *testing.T) {
	ctx := context.Background()
	defer testerApp.clearDB()

	names := []string{"Jacob", "Oliver", "Diane", "Rose", "Kristian"}

	startTime := time.Now().UTC()
	createdAtTime := startTime

	userCount := 5
	for i := range userCount {

		testUser := mongodb.DBUser{
			ID:        uuid.New().String(),
			FirstName: names[i],
			LastName:  "Petersen",
			Nickname:  fmt.Sprintf("test user %d", i),
			Password:  "Alalal",
			Email:     "hello@example.com",
			Country:   "SWE",
			Salt:      crypto.GenerateSalt(),
			CreatedAt: createdAtTime,
			UpdatedAt: createdAtTime,
		}

		coll := testerApp.dbConn.Database("functional").Collection("user")
		_, err := coll.InsertOne(ctx, testUser)
		require.NoError(t, err)
		createdAtTime.Add(24 * time.Hour)
	}

	type tester struct {
		option      proto.Ordering
		requireFunc func(t require.TestingT, e1 interface{}, e2 interface{}, msgAndArgs ...interface{})
	}

	// test ordering of first name
	resp, err := testerApp.grpcClient.ListUsers(ctx, &proto.ListUsersRequest{})
	require.NoError(t, err)
	require.Equal(t, userCount, len(resp.Users))

	tests := []tester{
		{
			option:      proto.Ordering_ORDERING_UNSPECIFIED,
			requireFunc: require.Less,
		},
		{
			option:      proto.Ordering_ORDERING_ASCENDING,
			requireFunc: require.Less,
		},
		{
			option:      proto.Ordering_ORDERING_DESCENDING,
			requireFunc: require.Greater,
		},
	}

	for _, test := range tests {
		// sorting on first name
		resp, err = testerApp.grpcClient.ListUsers(ctx, &proto.ListUsersRequest{
			Sorting: &proto.SortInfo{
				By:    proto.UserField_USER_FIELD_FIRST_NAME,
				Order: &test.option,
			},
		})
		require.NoError(t, err)
		prevName := ""
		for _, user := range resp.Users {
			if prevName != "" {
				test.requireFunc(t, prevName, user.FirstName)
			}
			prevName = user.FirstName
		}
	}

	// filtering on first name
	resp, err = testerApp.grpcClient.ListUsers(ctx, &proto.ListUsersRequest{
		Filtering: &proto.FilterInfo{
			Left:     proto.UserField_USER_FIELD_FIRST_NAME,
			Comparer: proto.Comparer_COMPARER_GREATER_THAN_EQUAL,
			Right:    "Kristian",
		},
		Sorting: &proto.SortInfo{
			By: proto.UserField_USER_FIELD_FIRST_NAME,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 3, len(resp.Users))
	require.Equal(t, "Kristian", resp.Users[0].FirstName)
	require.Equal(t, "Oliver", resp.Users[1].FirstName)
	require.Equal(t, "Rose", resp.Users[2].FirstName)

	// paging one by one
	sortedNames := names[:]
	sort.Strings(sortedNames)

	pagingInfo := &proto.PageInfo{
		Limit:  1,
		Cursor: "",
	}
	for i := range userCount {
		resp, err = testerApp.grpcClient.ListUsers(ctx, &proto.ListUsersRequest{
			Sorting: &proto.SortInfo{
				By: proto.UserField_USER_FIELD_FIRST_NAME,
			},
			Paging: pagingInfo,
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Users))
		require.Equal(t, sortedNames[i], resp.Users[0].FirstName)
		fmt.Printf("first name: %s\n", resp.Users[0].FirstName)
		pagingInfo = resp.Next
	}

	// paging where we are running out of entries
	pagingInfo = &proto.PageInfo{
		Limit:  3,
		Cursor: "",
	}

	// first page
	resp, err = testerApp.grpcClient.ListUsers(ctx, &proto.ListUsersRequest{
		Sorting: &proto.SortInfo{
			By: proto.UserField_USER_FIELD_FIRST_NAME,
		},
		Paging: pagingInfo,
	})
	require.NoError(t, err)
	require.Equal(t, 3, len(resp.Users))
	require.Equal(t, sortedNames[0], resp.Users[0].FirstName)
	require.Equal(t, sortedNames[1], resp.Users[1].FirstName)
	require.Equal(t, sortedNames[2], resp.Users[2].FirstName)
	pagingInfo = resp.Next

	// second page
	resp, err = testerApp.grpcClient.ListUsers(ctx, &proto.ListUsersRequest{
		Sorting: &proto.SortInfo{
			By: proto.UserField_USER_FIELD_FIRST_NAME,
		},
		Paging: pagingInfo,
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.Users))
	require.Equal(t, sortedNames[3], resp.Users[0].FirstName)
	require.Equal(t, sortedNames[4], resp.Users[1].FirstName)
	pagingInfo = resp.Next
}
