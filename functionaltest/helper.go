package functionaltest

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"userservice/internal/infrastructure/mongodb"
)

func getAllUserDBEntries(ctx context.Context, coll *mongo.Collection) []mongodb.DBUser {
	findOptions := options.Find()
	var results []mongodb.DBUser

	cur, err := coll.Find(ctx, bson.D{{}}, findOptions)
	if err != nil {
		panic(err)
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var user mongodb.DBUser
		err = cur.Decode(&user)
		if err != nil {
			panic(err)
		}
		results = append(results, user)
	}

	err = cur.Err()
	if err != nil {
		panic(err)
	}

	return results
}

func getAllKafkaDBEntries(ctx context.Context, coll *mongo.Collection) []mongodb.Message {
	findOptions := options.Find()
	var results []mongodb.Message

	cur, err := coll.Find(ctx, bson.D{{}}, findOptions)
	if err != nil {
		panic(err)
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var msg mongodb.Message
		err = cur.Decode(&msg)
		if err != nil {
			panic(err)
		}
		results = append(results, msg)
	}

	err = cur.Err()
	if err != nil {
		panic(err)
	}

	return results
}
