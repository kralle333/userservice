package functionaltest

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
	"time"
)

type kafkaConsumerWrapper struct {
	consumer         *kafka.Consumer
	messageChannel   chan kafka.Message
	messageValidator func(message *kafka.Message)
	shutdownChannel  chan struct{}
	running          bool
}

func newKafkaConsumerWrapper(consumer *kafka.Consumer) *kafkaConsumerWrapper {
	return &kafkaConsumerWrapper{consumer: consumer, messageChannel: nil}
}

func (k *kafkaConsumerWrapper) setMessageValidator(validator func(val *kafka.Message)) {
	k.messageValidator = validator
}

func (k *kafkaConsumerWrapper) shutdown() {
	k.running = false
}

func (k *kafkaConsumerWrapper) listen() {
	c := k.consumer
	k.running = true

	for k.running {
		msg, err := c.ReadMessage(-1)
		if err == nil {
			log.Info().Msgf("KafkaConsumer: Received message: %s", string(msg.Value))
			k.tryCommunicateKafkaMessage(msg)
		} else if err.(kafka.Error).Code() != kafka.ErrTimedOut {
			log.Info().Err(err).Msgf("KafkaConsumer: error: %v", msg)
		}
	}

	log.Info().Msg("KafkaConsumer: shutting down")
}

func (k *kafkaConsumerWrapper) setCommunicationChannel(channel chan kafka.Message) {
	k.messageChannel = channel
}

func (k *kafkaConsumerWrapper) clearCommunicationChannel() {
	k.messageChannel = nil
}

func (k *kafkaConsumerWrapper) tryCommunicateKafkaMessage(msg *kafka.Message) {
	channel := k.messageChannel

	if channel == nil {
		// not ready to communicate yet, most likely old messages we are not interested in testing
		return
	}

	for {
		timeOut := time.NewTimer(1 * time.Second)
		select {
		case channel <- *msg:
			return
		case <-timeOut.C:
			log.Fatal().Msgf("Attempted to communicate message consumed, but no one is there to consume!")
			return
		}
	}
}
