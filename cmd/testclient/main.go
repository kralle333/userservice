package main

import (
	"context"
	"flag"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
	proto "userservice/proto/grpc"

	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("addr", "localhost:9091", "the address to connect to")
)

func main() {
	flag.Parse()

	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	c := proto.NewUserServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	addedUserResponse, err := c.AddUser(ctx, &proto.AddUserRequest{
		User: &proto.AddUserRequestUser{
			FirstName: "John",
			LastName:  "Last",
			Nickname:  "Johnnie",
			Email:     "hey@yo.com",
			Password:  "xsecurex",
			Country:   "DK",
		},
	})
	if err != nil {
		log.Fatalf("could not add: %v", err)
	}

	listUsersResponse, err := c.ListUsers(ctx, &proto.ListUsersRequest{})
	if err != nil {
		log.Fatalf("could not list: %v", err)
	}
	for _, user := range listUsersResponse.Users {
		log.Printf("user: %v\n", user)
	}

	_, err = c.RemoveUser(ctx, &proto.RemoveUserRequest{
		UserID: addedUserResponse.User.Id,
	})
	if err != nil {
		log.Fatalf("could not remove user: %v", err)
	}
}
