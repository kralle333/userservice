package functionaltest

import (
	"context"
	"flag"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
	"time"
	"userservice/internal/config"
	proto "userservice/proto/grpc"
)

var (
	serverAddr = flag.String("addr", "localhost:9091", "the address to connect to")
	configPath = flag.String("config", "../config/debug.json", "Path to config file")
)

type TesterApp struct {
	grpcClient proto.UserServiceClient
	// for verifying correct stage
	consumer              *kafkaConsumerWrapper
	dbConn                *mongo.Client
	db                    *mongo.Database
	kafkaOutboxCollection *mongo.Collection
	userCollection        *mongo.Collection
	serverConfig          config.AppConfig
}

var testerApp = &TesterApp{}

func TestMain(m *testing.M) {
	flag.Parse()

	log.Info().Msg("setting up tests")

	readConfig, err := config.ReadConfig(*configPath)
	if err != nil {
		panic(err)
	}
	readConfig.Database.DatabaseName = "functional" // make sure that server is run with the same database name
	testerApp.serverConfig = readConfig

	addDBConnection()
	addGRPCClient()
	addKafkaConsumer()
	go testerApp.consumer.listen()
	defer testerApp.consumer.shutdown()

	waitingDelay := 5 * time.Second
	log.Info().Msgf("waiting %s for kafka to settle", waitingDelay)
	time.Sleep(5 * time.Second)

	log.Info().Msg("starting tests")
	m.Run()
}

func addDBConnection() {
	ctx := context.Background()

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(testerApp.serverConfig.Database.ConnectionString).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(timeoutCtx, opts)
	if err != nil {
		panic(err)
	}
	err = client.Ping(timeoutCtx, nil)
	if err != nil {
		panic(err)
	}

	testerApp.dbConn = client
	testerApp.db = client.Database(testerApp.serverConfig.Database.DatabaseName)
	testerApp.userCollection = testerApp.db.Collection(testerApp.serverConfig.Database.UserCollectionName)
	testerApp.kafkaOutboxCollection = testerApp.db.Collection(testerApp.serverConfig.Database.KafkaOutboxCollectionName)
	testerApp.clearDB()
}

func addGRPCClient() {
	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	testerApp.grpcClient = proto.NewUserServiceClient(conn)
}

func addKafkaConsumer() {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "0.0.0.0:29092",
		"group.id":          uuid.New().String(), // make sure to not reuse group id, to prevent getting old messages
		"auto.offset.reset": "latest",            // only interested in new messages
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Created Consumer %v\n", c)

	kc := testerApp.serverConfig.Kafka

	err = c.SubscribeTopics([]string{kc.UserAddedTopicName, kc.UserRemovedTopicName}, nil)
	if err != nil {
		panic(err)
	}

	subscriptions, err := c.Subscription()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Subscribed to topics %v\n", subscriptions)

	if c.IsClosed() {
		panic("is down")
	}

	testerApp.consumer = newKafkaConsumerWrapper(c)
}

func (t *TesterApp) clearDB() {
	// quick and dirty way to make sure we have a clean DB
	err := t.db.Drop(context.Background())
	if err != nil {
		panic(err)
	}

}
