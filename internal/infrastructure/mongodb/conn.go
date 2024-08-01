package mongodb

import (
	"context"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"userservice/internal/config"
)

type Connection struct {
	client           *mongo.Client
	usersCollection  *mongo.Collection
	outboxCollection *mongo.Collection
	dbConfig         config.DatabaseConfig
	kafkaConfig      config.KafkaConfig
}

func NewMongoDBConnection(client *mongo.Client, dbConfig config.DatabaseConfig, kafkaConfig config.KafkaConfig) *Connection {

	appDB := client.Database(dbConfig.DatabaseName)

	return &Connection{
		client:           client,
		usersCollection:  appDB.Collection(dbConfig.UserCollectionName),
		outboxCollection: appDB.Collection(dbConfig.KafkaOutboxCollectionName),
		dbConfig:         dbConfig,
		kafkaConfig:      kafkaConfig,
	}
}

func (c *Connection) CleanUp(ctx context.Context) {

	if c.client == nil {
		return
	}
	err := c.client.Disconnect(ctx)
	if err != nil {
		log.Warn().Err(err).Msgf("error when disconnecting MongoDBConnection client")
	}
	c.client = nil
}
