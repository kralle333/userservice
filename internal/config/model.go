package config

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

type KafkaConfig struct {
	UserAddedTopicName   string       `json:"userAddedTopicName"`
	UserRemovedTopicName string       `json:"userRemovedTopicName"`
	Outbox               OutboxConfig `json:"outbox"`
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
	OrdinaryHealthCheckListeningPort int    `json:"ordinaryHealthCheckListeningPort"`
}
