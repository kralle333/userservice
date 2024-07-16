package functionaltest

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"userservice/internal/db"
)

func getAllUserDBEntries(ctx context.Context, coll *mongo.Collection) []db.User {
	findOptions := options.Find()
	var results []db.User

	cur, err := coll.Find(ctx, bson.D{{}}, findOptions)
	if err != nil {
		panic(err)
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var user db.User
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

func getAllKafkaDBEntries(ctx context.Context, coll *mongo.Collection) []db.KafkaOutboxMessage {
	findOptions := options.Find()
	var results []db.KafkaOutboxMessage

	cur, err := coll.Find(ctx, bson.D{{}}, findOptions)
	if err != nil {
		panic(err)
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var msg db.KafkaOutboxMessage
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
