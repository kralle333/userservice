package config

import (
	"encoding/json"
	"github.com/kelseyhightower/envconfig"
	"os"
)

type AppConfig struct {
	Server        ServerConfig        `json:"server"`
	Database      DatabaseConfig      `json:"database"`
	Kafka         KafkaConfig         `json:"kafka"`
	HealthChecker HealthCheckerConfig `json:"healthChecker"`
}
type ServerConfig struct {
	ListeningPort int `json:"listeningPort"`
}
type DatabaseConfig struct {
	ConnectionString          string `json:"connectionString"`
	DatabaseName              string `envconfig:"DATABASE_NAME" json:"databaseName"`
	UserCollectionName        string `json:"userCollectionName"`
	KafkaOutboxCollectionName string `json:"kafkaOutboxCollectionName"`
	InitialRetryDelaySeconds  int64  `json:"initialRetryDelaySeconds"`
	UserIdName                string `json:"userIdName"`
	ListUserDefaultLimit      int64  `json:"listUserDefaultLimit"`
	ListUserMaxLimit          int64  `json:"listUserMaxLimit"`
}

type KafkaTopicsConfig struct {
	UserAddedTopicName   string `json:"userAddedTopicName"`
	UserRemovedTopicName string `json:"userRemovedTopicName"`
}

type KafkaConfig struct {
	Topics KafkaTopicsConfig `json:"topics"`
	Outbox OutboxConfig      `json:"outbox"`
}

type OutboxConfig struct {
	SleepIntervalSeconds int64 `json:"producerSleepIntervalSeconds"`
	BaseRetryTimeSeconds int64 `json:"baseRetryTimeSeconds"`
	MaxRetryTimeSeconds  int64 `json:"maxRetryTimeSeconds"`
}

type HealthCheckerConfig struct {
	HealthTopicName                  string `json:"healthTopicName"`
	BootstrapServer                  string `json:"bootstrapServer"`
	TickerIntervalSeconds            int64  `json:"tickerIntervalSeconds"`
	HealthCheckTickerIntervalSeconds int64  `json:"healthCheckTickerIntervalSeconds"`
	OrdinaryHealthCheckListeningPort int    `json:"ordinaryHealthCheckListeningPort"`
}

func ReadConfig(path string) (AppConfig, error) {

	f, err := os.Open(path)
	if err != nil {
		return AppConfig{}, err
	}
	defer f.Close()

	var cfg AppConfig
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return AppConfig{}, err
	}

	err = envconfig.Process("", &cfg)
	if err != nil {
		return AppConfig{}, err
	}

	return cfg, nil
}
