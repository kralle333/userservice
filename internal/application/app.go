package application

import (
	"context"
	"fmt"
	confkafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"net"
	"time"
	"userservice/internal/application/api"
	"userservice/internal/config"
	"userservice/internal/domain/user"
	"userservice/internal/infrastructure/health"
	appdb "userservice/internal/infrastructure/mongodb"
	"userservice/internal/infrastructure/outbox"
)

type App struct {
	kafkaOutboxService outbox.Outbox
	server             *api.Server
	config             config.AppConfig
}

func NewApp(config config.AppConfig) *App {
	app, err := buildApp(config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to build app")
	}
	return app
}

func buildApp(config config.AppConfig) (*App, error) {
	ctx := context.Background()
	mongoDBConn, err := constructDBConnection(ctx, config.Database)
	if err != nil {
		log.Panic().Err(err).Msg("failed constructing Database connection")
	}

	// Dependencies
	dbRepo := appdb.NewMongoDBConnection(mongoDBConn, config.Database, config.Kafka)
	kafkaProducer, err := createKafkaProducer()
	if err != nil {
		return nil, errors.Wrap(err, "failed creating kafka producer")
	}

	// Outbox
	kafkaOutboxService := outbox.NewKafkaOutbox(kafkaProducer, dbRepo, config.Kafka.Outbox)

	// User
	usersComponent := user.NewUserComponent(dbRepo)
	userController := api.NewUserController(usersComponent)

	// Health Check
	healthCheckController := api.NewHealthCheckController(config.HealthChecker)
	healthCheckController.RegisterHealthCheckable(ctx, health.NewMongoDBHealthCheckable(mongoDBConn))
	healthCheckController.RegisterHealthCheckable(ctx, health.NewKafkaHealthCheckable(kafkaProducer))

	server := api.NewServer(userController, healthCheckController)

	return &App{
		kafkaOutboxService: kafkaOutboxService,
		server:             server,
		config:             config,
	}, nil
}
func (a *App) Run() error {

	portString := fmt.Sprintf(":%d", a.config.Server.ListeningPort)
	lis, err := net.Listen("tcp", portString)
	if err != nil || lis == nil {
		log.Panic().Msgf("failed to listen on port %s", portString)
	}

	// app server
	s := grpc.NewServer()
	a.server.RegisterGRPC(s)
	a.server.Run()

	// kafka outbox
	go a.kafkaOutboxService.Run()

	// serving
	log.Info().Msgf("Server listening at %v", lis.Addr())
	err = s.Serve(lis)

	if err != nil {
		log.Panic().Msgf("failed to serve")
	}

	return nil
}

func constructDBConnection(ctx context.Context, config config.DatabaseConfig) (*mongo.Client, error) {

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Info().Msg("constructing Database connection")

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(config.ConnectionString).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(timeoutCtx, opts)
	if err != nil {
		return nil, err
	}

	err = client.Ping(timeoutCtx, nil)
	if err != nil {
		return nil, err
	}
	log.Info().Msg("connected to Database instance")

	return client, nil
}

func createKafkaProducer() (*confkafka.Producer, error) {
	p, err := confkafka.NewProducer(&confkafka.ConfigMap{
		"bootstrap.servers": "0.0.0.0:29092",
	})
	if err != nil {
		return nil, err
	}

	return p, nil
}
