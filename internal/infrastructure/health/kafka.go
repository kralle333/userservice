package health

import (
	"context"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"time"
)

type kafkaHealthCheckable struct {
	producer *kafka.Producer
}

func NewKafkaHealthCheckable(producer *kafka.Producer) Checkable {
	return &kafkaHealthCheckable{producer: producer}
}

func (k *kafkaHealthCheckable) GetName() string {
	return "Kafka"
}

func (k *kafkaHealthCheckable) RunHealthCheck(checkContext context.Context, reportChannel chan Report, checkTimeInterval time.Duration) {
	checkTicker := time.NewTicker(checkTimeInterval)
	go func() {

		for {
			select {
			case <-checkContext.Done():
			case <-checkTicker.C:
				err := k.checkHealth()
				var report Report
				if err != nil {
					log.Error().Err(err).Msg("Kafka Healthcheck Fail")
					report = NewHealthReport(k, StatusNotServing)
				} else {
					report = NewHealthReport(k, StatusServing)
				}
				reportChannel <- report
			}
		}
	}()
}

func (k *kafkaHealthCheckable) checkHealth() error {
	// produce a test message to check if we can communicate with kafka

	topic := "Health"
	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte("user service health check"),
	}

	deliveryChan := make(chan kafka.Event)
	defer close(deliveryChan)

	err := k.producer.Produce(message, deliveryChan)
	if err != nil {
		return errors.Wrap(err, "Failed to produce message")
	}

	// Wait for the delivery report
	select {
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return errors.Wrap(m.TopicPartition.Error, "Failed to deliver message")
		}
		return nil

	case <-time.After(10 * time.Second):
		return errors.New("Failed to deliver message")
	}
}
