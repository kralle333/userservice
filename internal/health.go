package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net/http"
	"sync"
	"time"
	"userservice/internal/config"
	"userservice/internal/db"
)

type ServiceName string

const (
	ServiceNameMongoDB ServiceName = "MongoDB"
	ServiceNameKafka   ServiceName = "Kafka"
)

var allServices = []ServiceName{ServiceNameMongoDB, ServiceNameKafka}

func parseServiceName(name string) (ServiceName, error) {
	for _, service := range allServices {
		if string(service) == name {
			return service, nil
		}
	}
	return "", errors.New("invalid service name")
}

type healthChecker struct {
	grpc_health_v1.UnimplementedHealthServer
	mongoDBRepo   *db.MongoDBRepo
	config        config.HealthCheckerConfig
	statusMapLock sync.Mutex
	statusMap     map[ServiceName]grpc_health_v1.HealthCheckResponse_ServingStatus
	startTime     time.Time
}

func newHealthChecker(mongoDBRepo *db.MongoDBRepo, config config.HealthCheckerConfig) *healthChecker {
	status := make(map[ServiceName]grpc_health_v1.HealthCheckResponse_ServingStatus)

	for _, service := range allServices {
		status[service] = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	return &healthChecker{
		mongoDBRepo:   mongoDBRepo,
		config:        config,
		statusMapLock: sync.Mutex{},
		statusMap:     status,
		startTime:     time.Now().UTC(),
	}
}

func (h *healthChecker) setStatus(service ServiceName, serving grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.statusMapLock.Lock()
	defer h.statusMapLock.Unlock()
	h.statusMap[service] = serving
}

func (h *healthChecker) getStatus(serviceName ServiceName) grpc_health_v1.HealthCheckResponse_ServingStatus {
	h.statusMapLock.Lock()
	defer h.statusMapLock.Unlock()
	_, ok := h.statusMap[serviceName]
	if !ok {
		log.Warn().Msgf("serviceName %s is not found in status map! check constructor setting of map - setting value in map", serviceName)
		h.statusMap[serviceName] = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}
	return h.statusMap[serviceName]
}

func (h *healthChecker) updateStatusMongoDB(ctx context.Context) {
	err := h.mongoDBRepo.Ping(ctx, 5*time.Second)
	if err != nil {
		h.setStatus(ServiceNameMongoDB, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	} else {
		h.setStatus(ServiceNameMongoDB, grpc_health_v1.HealthCheckResponse_SERVING)
	}

}

func (h *healthChecker) updateStatusKafka() {
	h.setStatus(ServiceNameKafka, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers":        h.config.BootstrapServer,
		"allow.auto.create.topics": true,
	}
	producer, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		log.Error().Err(err).Msg("HealthChecker: failed to create producer")
		return
	}
	defer producer.Close()

	// produce a test message to check if we can communicate with kafka
	topic := h.config.HealthTopicName
	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte("user service health check"),
	}

	deliveryChan := make(chan kafka.Event)
	defer close(deliveryChan)

	err = producer.Produce(message, deliveryChan)
	if err != nil {
		log.Error().Err(err).Msg("HealthChecker: failed to produce message")
	}

	// Wait for the delivery report
	select {
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			log.Error().Err(m.TopicPartition.Error).Msgf("HealthChecker: delivery failed")
			return
		}
		h.setStatus(ServiceNameKafka, grpc_health_v1.HealthCheckResponse_SERVING)
	case <-time.After(10 * time.Second):
		log.Warn().Msgf("HealthChecker: failed to deliver health check kafka message: timed out")
	}

}

func (h *healthChecker) Run() {
	ctx := context.Background()

	sleepInterval := time.Duration(h.config.TickerIntervalSeconds) * time.Second

	for {
		log.Info().Msg("HealthChecker: checking status")
		h.updateStatusMongoDB(ctx)
		h.updateStatusKafka()
		time.Sleep(sleepInterval)
	}
}

func (h *healthChecker) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	h.statusMapLock.Lock()
	defer h.statusMapLock.Unlock()

	name, err := parseServiceName(req.Service)
	if err != nil {
		return nil, err
	}
	resp := &grpc_health_v1.HealthCheckResponse{}
	resp.Status = h.getStatus(name)
	return resp, nil
}

func (h *healthChecker) Watch(req *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {
	name, err := parseServiceName(req.Service)
	if err != nil {
		return err
	}

	for {
		time.Sleep(5 * time.Second)
		err := server.Send(&grpc_health_v1.HealthCheckResponse{Status: h.getStatus(name)})
		if err != nil {
			return err
		}
	}
}

func (h *healthChecker) ServeOrdinaryHealthEndpoint() {
	type OrdinaryHealthCheckResponse struct {
		Uptime string   `json:"uptime"`
		Status []string `json:"status"`
	}
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {

		h.statusMapLock.Lock()
		defer h.statusMapLock.Unlock()

		resp := OrdinaryHealthCheckResponse{
			Status: []string{},
			Uptime: fmt.Sprintf("up for %s", time.Since(h.startTime)),
		}
		for s, status := range h.statusMap {
			resp.Status = append(resp.Status, fmt.Sprintf("%s: %s", s, status.String()))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	portString := fmt.Sprintf(":%d", h.config.OrdinaryHealthCheckListeningPort)
	log.Info().Msgf("Serving healtcheck at %s/healthz", portString)
	err := http.ListenAndServe(portString, nil)
	if err != nil {
		log.Panic().Msgf("failed to listen on port %s", portString)
	}
}
