package internal

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"time"
	"userservice/internal/config"
	appdb "userservice/internal/db"
	"userservice/internal/kafkaoutbox"
	"userservice/internal/user"
	grpcproto "userservice/proto/grpc"

	confkafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type App struct {
	repo               *appdb.MongoDBRepo
	usersComponent     user.Component
	kafkaOutboxService kafkaoutbox.OutboxService
	server             *server
	config             config.AppConfig
}

func NewApp(config config.AppConfig) *App {
	ctx := context.Background()

	mongoDBConn, err := constructDBConnection(ctx, config.Database)
	if err != nil {
		log.Panic().Err(err).Msg("failed constructing Database connection")
	}

	dbRepo := appdb.NewDBRepo(mongoDBConn, config.Database)

	outboxCollection := mongoDBConn.Database(config.Database.DatabaseName).Collection(config.Database.KafkaOutboxCollectionName)
	kafkaOutboxRepo := appdb.NewKafkaMessageOutboxDBRepo(outboxCollection, dbRepo, config.Kafka.Outbox)

	kafkaService, err := constructKafkaOutbox(kafkaOutboxRepo, config.Kafka.Outbox)
	if err != nil {
		log.Panic().Err(err).Msg("failed constructing kafka service")
	}

	usersComponent := user.NewUserComponent(dbRepo, dbRepo, dbRepo, config.Kafka)
	userServer := newServer(usersComponent)

	return &App{
		repo:               dbRepo,
		usersComponent:     usersComponent,
		kafkaOutboxService: kafkaService,
		server:             userServer,
		config:             config,
	}
}

func (a *App) Run() error {
	ctx := context.Background()
	defer a.cleanUp(ctx)

	portString := fmt.Sprintf(":%d", a.config.Server.ListeningPort)
	lis, err := net.Listen("tcp", portString)
	if err != nil {
		log.Panic().Msgf("failed to listen on port %s", portString)
	}

	go a.kafkaOutboxService.Run()

	// health
	s := grpc.NewServer()
	healthServer := newHealthChecker(a.repo, a.config.HealthChecker)
	healthgrpc.RegisterHealthServer(s, healthServer)
	go healthServer.ServeOrdinaryHealthEndpoint()
	go healthServer.Run()

	// server
	grpcproto.RegisterUserServiceServer(s, a.server)
	log.Info().Msgf("Server listening at %v", lis.Addr())
	err = s.Serve(lis)

	if err != nil {
		log.Panic().Msgf("failed to serve")
	}

	return nil
}

func (a *App) cleanUp(ctx context.Context) {
	if a.repo != nil {
		a.repo.CleanUp(ctx)
	}
	if a.kafkaOutboxService != nil {
		a.kafkaOutboxService.CleanUp()
	}
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

func constructKafkaOutbox(kafkaOutboxRepo appdb.KafkaMessageOutboxDBRepo, config config.OutboxConfig) (kafkaoutbox.OutboxService, error) {
	p, err := confkafka.NewProducer(&confkafka.ConfigMap{
		"bootstrap.servers": "0.0.0.0:29092",
	})
	if err != nil {
		return nil, err
	}

	return kafkaoutbox.NewKafkaOutbox(p, kafkaOutboxRepo, config), nil
}
