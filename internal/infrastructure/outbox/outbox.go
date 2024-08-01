package outbox

import (
	"context"
	"errors"
	"fmt"
	confkafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
	"userservice/internal/config"
	"userservice/internal/infrastructure/messaging"
)

type outbox struct {
	producer        *confkafka.Producer
	outboxLock      sync.RWMutex
	kafkaOutboxRepo MessageOutboxRepo
	shutdownChannel chan struct{}
	hasBeenShutDown bool
	config          config.OutboxConfig
}

type Outbox interface {
	CleanUp()
	Run()
}

func NewKafkaOutbox(producer *confkafka.Producer, repo MessageOutboxRepo, config config.OutboxConfig) Outbox {
	return &outbox{
		producer:        producer,
		outboxLock:      sync.RWMutex{},
		kafkaOutboxRepo: repo,
		shutdownChannel: make(chan struct{}),
		hasBeenShutDown: true,
		config:          config,
	}
}

func (o *outbox) CleanUp() {
	if o.hasBeenShutDown {
		return
	}

	o.producer.Close()
	o.shutdownChannel <- struct{}{}
	close(o.shutdownChannel)
	o.hasBeenShutDown = true
}

func (o *outbox) Run() {
	ctx := context.Background()
	ticker := time.NewTicker(time.Duration(o.config.SleepIntervalSeconds) * time.Second)

	log.Info().Msgf("Outbox: starting")
outerLoop:
	for {
		select {
		case <-o.shutdownChannel:
			break outerLoop
		case <-ticker.C:
			message, err := o.getOnePendingKafkaMessage(ctx)
			if errors.Is(err, ErrNoPendingMessage) {
				log.Info().Msgf("Outbox: found no pending messages, going back to sleep")
				break
			}
			err = o.produce(ctx, message)
			if err != nil {
				log.Error().Err(err).Msgf("Outbox: failed to produce message with id %o, going back to sleep", message.ID)
			}
		}
	}
	log.Info().Msgf("Outbox: stopped main loop")
}

func toConfluentKafkaMessage(m messaging.KafkaInternalMessage) *confkafka.Message {
	return &confkafka.Message{
		TopicPartition: confkafka.TopicPartition{
			Topic: &m.TopicID,
		},
		Value: m.Value,
		Key:   m.Key,
	}
}

func (o *outbox) getOnePendingKafkaMessage(ctx context.Context) (messaging.KafkaInternalMessage, error) {
	o.outboxLock.RLock()
	defer o.outboxLock.RUnlock()
	message, err := o.kafkaOutboxRepo.GetPendingMessage(ctx)
	if err != nil {
		return messaging.KafkaInternalMessage{}, err
	}
	return message, nil
}

func (o *outbox) retryKafkaMessage(ctx context.Context, id string) {
	o.outboxLock.RLock()
	defer o.outboxLock.RUnlock()
	o.kafkaOutboxRepo.RetryMessage(ctx, id)
}

func (o *outbox) markKafkaMessageSent(ctx context.Context, id string) {
	o.outboxLock.RLock()
	defer o.outboxLock.RUnlock()
	o.kafkaOutboxRepo.MarkMessageSent(ctx, id)
}

func (o *outbox) produce(ctx context.Context, msg messaging.KafkaInternalMessage) error {

	successChan := make(chan confkafka.Event)
	err := o.producer.Produce(toConfluentKafkaMessage(msg), successChan)
	if err != nil {
		return fmt.Errorf("failed to produce message %v", err.Error())
	}
	go o.listenForKafkaMessageProduced(ctx, successChan, msg)

	return nil
}

func (o *outbox) listenForKafkaMessageProduced(ctx context.Context, successChannel chan confkafka.Event, msg messaging.KafkaInternalMessage) {
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
				o.retryKafkaMessage(ctx, msg.ID)
				return
			}
		case <-successChannel:
			log.Info().Msgf("Outbox: message sent with id %o", msg.ID)
			o.markKafkaMessageSent(ctx, msg.ID)
			return
		}
	}

}
