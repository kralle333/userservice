package kafkaoutbox

import (
	"context"
	"errors"
	"fmt"
	confkafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
	"userservice/internal/config"
	"userservice/internal/db"
	"userservice/internal/kafkamessage"
)

type outbox struct {
	producer        *confkafka.Producer
	outboxLock      sync.RWMutex
	kafkaOutboxRepo db.KafkaMessageOutboxDBRepo
	shutdownChannel chan struct{}
	hasBeenShutDown bool
	config          config.OutboxConfig
}

type OutboxService interface {
	CleanUp()
	Run()
}

func NewKafkaOutbox(producer *confkafka.Producer, repo db.KafkaMessageOutboxDBRepo, config config.OutboxConfig) OutboxService {
	return &outbox{
		producer:        producer,
		outboxLock:      sync.RWMutex{},
		kafkaOutboxRepo: repo,
		shutdownChannel: make(chan struct{}),
		hasBeenShutDown: true,
		config:          config,
	}
}

func (s *outbox) CleanUp() {
	if s.hasBeenShutDown {
		return
	}

	s.producer.Close()
	s.shutdownChannel <- struct{}{}
	close(s.shutdownChannel)
	s.hasBeenShutDown = true
}

func (s *outbox) Run() {
	ctx := context.Background()
	ticker := time.NewTicker(time.Duration(s.config.SleepIntervalSeconds) * time.Second)

	log.Info().Msgf("KafkaOutbox: starting")
outerLoop:
	for {
		select {
		case <-s.shutdownChannel:
			break outerLoop
		case <-ticker.C:
			message, err := s.getOnePendingKafkaMessage(ctx)
			if errors.Is(err, db.ErrNoPendingMessage) {
				log.Info().Msgf("KafkaOutbox: found no pending messages, going back to sleep")
				break
			}
			err = s.produce(ctx, message)
			if err != nil {
				log.Error().Err(err).Msgf("KafkaOutbox: failed to produce message with id %s, going back to sleep", message.ID)
			}
		}
	}
	log.Info().Msgf("KafkaOutbox: stopped main loop")
}

func toConfluentKafkaMessage(m kafkamessage.InternalMessage) *confkafka.Message {
	return &confkafka.Message{
		TopicPartition: confkafka.TopicPartition{
			Topic: &m.TopicID,
		},
		Value: m.Value,
		Key:   m.Key,
	}
}

func (s *outbox) getOnePendingKafkaMessage(ctx context.Context) (kafkamessage.InternalMessage, error) {
	s.outboxLock.RLock()
	defer s.outboxLock.RUnlock()
	message, err := s.kafkaOutboxRepo.GetPendingMessage(ctx)
	if err != nil {
		return kafkamessage.InternalMessage{}, err
	}
	return message, nil
}

func (s *outbox) retryKafkaMessage(ctx context.Context, id string) {
	s.outboxLock.RLock()
	defer s.outboxLock.RUnlock()
	s.kafkaOutboxRepo.RetryMessage(ctx, id)
}

func (s *outbox) markKafkaMessageSent(ctx context.Context, id string) {
	s.outboxLock.RLock()
	defer s.outboxLock.RUnlock()
	s.kafkaOutboxRepo.MarkMessageSent(ctx, id)
}

func (s *outbox) produce(ctx context.Context, msg kafkamessage.InternalMessage) error {

	successChan := make(chan confkafka.Event)
	err := s.producer.Produce(toConfluentKafkaMessage(msg), successChan)
	if err != nil {
		return fmt.Errorf("failed to produce message %v", err.Error())
	}
	go s.listenForKafkaMessageProduced(ctx, successChan, msg)

	return nil
}

func (s *outbox) listenForKafkaMessageProduced(ctx context.Context, successChannel chan confkafka.Event, msg kafkamessage.InternalMessage) {
	maxWaitingTime := 300 // 5 minutes, then we assume we failed
	tickerTimeSeconds := 5
	timer := time.NewTicker(time.Duration(tickerTimeSeconds) * time.Second)
	tickCount := 0
	allowedTicks := maxWaitingTime / tickerTimeSeconds

	for {
		select {
		case <-timer.C:
			tickCount = tickCount + 1
			if tickCount > allowedTicks {
				errMessage := fmt.Sprintf("waited for %d seconds, unable to verify that message was successfully sent, message: %v", maxWaitingTime, msg)
				log.Error().Msg(errMessage)
				s.retryKafkaMessage(ctx, msg.ID)
				return
			}
		case <-successChannel:
			log.Info().Msgf("KafkaOutbox: message sent with id %s", msg.ID)
			s.markKafkaMessageSent(ctx, msg.ID)
			return
		}
	}

}
