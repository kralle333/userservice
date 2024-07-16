package db

import (
	"context"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
	"userservice/internal/config"
)

type MongoDBRepo struct {
	client          *mongo.Client
	usersCollection *mongo.Collection
	kafkaCollection *mongo.Collection
	config          config.DatabaseConfig
}

func NewDBRepo(client *mongo.Client, config config.DatabaseConfig) *MongoDBRepo {

	appDB := client.Database(config.DatabaseName)

	return &MongoDBRepo{
		client:          client,
		usersCollection: appDB.Collection(config.UserCollectionName),
		kafkaCollection: appDB.Collection(config.KafkaOutboxCollectionName),
		config:          config,
	}
}

func (m *MongoDBRepo) Ping(ctx context.Context, timeoutDuration time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	err := m.client.Ping(ctxWithTimeout, nil)
	if err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepo) CleanUp(ctx context.Context) {

	if m.client == nil {
		return
	}
	err := m.client.Disconnect(ctx)
	if err != nil {
		log.Warn().Err(err).Msgf("error when disconnecting MongoDB client")
	}
	m.client = nil
}
