package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math"
	"math/rand"
	"time"
	"userservice/internal/infrastructure/messaging"
	"userservice/internal/infrastructure/outbox"
	timeutil "userservice/internal/util/time"
)

type Message struct {
	MsgID     string    `bson:"msg_id"` // have an id that is easily accessed without needing to deserialize data
	Data      []byte    `bson:"data"`
	NextRetry time.Time `bson:"next_retry"`
	SentAt    time.Time `bson:"sent_at"`
	Retries   int64     `bson:"retries"`
	State     string    `bson:"state"`
}

type MessageState string

func (k MessageState) String() string {
	return string(k)
}

const (
	StateWaiting    MessageState = "waiting"
	StateProcessing MessageState = "processing"
	StateFinished   MessageState = "finished"
)

func calculateNextAttempt(baseTimeSeconds int64, maxTimeSeconds int64, retryCount int) time.Time {

	maxTime := time.Second * time.Duration(maxTimeSeconds)
	// exponential backoff based on retryCount
	backoffDuration := time.Duration(float64(baseTimeSeconds) * math.Pow(2, float64(retryCount)))
	if backoffDuration > maxTime {
		backoffDuration = maxTime
	}

	// add jitter to prevent thundering herd problem
	jitter := time.Duration(rand.Int63n(int64(backoffDuration)))

	return time.Now().Add(jitter)
}
func (c *Connection) setState(ctx context.Context, msgID string, state MessageState) (Message, error) {

	upsert := false
	returnDocument := options.After
	opts := options.FindOneAndUpdateOptions{
		Upsert:         &upsert,
		ReturnDocument: &returnDocument,
	}

	setFilter := bson.D{{"$set", bson.D{{"state", state.String()}}}}

	var updatedMessage Message
	err := c.outboxCollection.FindOneAndUpdate(ctx, bson.D{{"msg_id", msgID}}, setFilter, &opts).Decode(&updatedMessage)
	if err != nil {
		return Message{}, err
	}
	return updatedMessage, nil
}

func (c *Connection) incrementNumberOfRetries(ctx context.Context, msgID string) (Message, error) {

	// first find the document
	// then update the document with the right values (retries,state,next_retry)

	findFilter := bson.D{{"msg_id", msgID}}
	message := c.outboxCollection.FindOne(ctx, findFilter)
	var decoded Message
	err := message.Decode(&decoded)
	if err != nil {
		return Message{}, err
	}
	if decoded.State != StateProcessing.String() {
		return Message{}, err
	}
	attempts := decoded.Retries + 1
	a := bson.D{}
	a = append(a, bson.E{Key: "retries", Value: attempts})
	a = append(a, bson.E{Key: "next_retry", Value: calculateNextAttempt(c.kafkaConfig.Outbox.BaseRetryTimeSeconds, c.kafkaConfig.Outbox.MaxRetryTimeSeconds, int(attempts))})

	upsert := false
	returnDocument := options.After
	opts := options.FindOneAndUpdateOptions{
		Upsert:         &upsert,
		ReturnDocument: &returnDocument,
	}

	var updatedMessage Message
	err = c.outboxCollection.FindOneAndUpdate(ctx, findFilter, a, &opts).Decode(&updatedMessage)
	if err != nil {
		return Message{}, err
	}
	return updatedMessage, nil
}

func (c *Connection) GetPendingMessage(ctx context.Context) (messaging.KafkaInternalMessage, error) {

	var pendingMessage Message
	err := c.executeInTransaction(ctx, func(sessionContext mongo.SessionContext) error {

		// find message with correct state:
		findFilter := bson.D{{"$and", bson.A{
			bson.D{{"state", StateWaiting}},
			bson.D{{"next_retry", bson.D{{"$lt", time.Now().UTC()}}}},
		}}}

		setFilter := bson.D{{
			"$set", bson.D{{"state", StateProcessing}}},
		}

		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		var processingMessage Message
		err := c.outboxCollection.FindOneAndUpdate(sessionContext, findFilter, setFilter, opts).Decode(&processingMessage)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return outbox.ErrNoPendingMessage
			}
			return err
		}
		// then we set the retry fields and next_attempt_at
		_, err = c.incrementNumberOfRetries(sessionContext, processingMessage.MsgID)
		pendingMessage, err = c.setState(sessionContext, processingMessage.MsgID, StateProcessing)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return messaging.KafkaInternalMessage{}, err
	}

	var toSend messaging.KafkaInternalMessage
	err = json.Unmarshal(pendingMessage.Data, &toSend)
	if err != nil {
		return messaging.KafkaInternalMessage{}, err
	}
	return toSend, nil
}
func (c *Connection) RetryMessage(ctx context.Context, id string) {
	err := c.executeInTransaction(ctx, func(sessionContext mongo.SessionContext) error {
		_, err := c.incrementNumberOfRetries(sessionContext, id)
		if err != nil {
			return err
		}
		_, err = c.setState(sessionContext, id, StateWaiting)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Warn().Err(err).Msgf("failed to retry message %s", id)
	}
}
func (c *Connection) MarkMessageSent(ctx context.Context, id string) {
	a := bson.D{}
	a = append(a, bson.E{Key: "sent_at", Value: timeutil.DBNow()})
	a = append(a, bson.E{Key: "state", Value: StateFinished})
	setFilter := bson.D{{"$set", a}}

	update := c.outboxCollection.FindOneAndUpdate(ctx, bson.D{{"msg_id", id}}, setFilter)
	if update.Err() != nil {
		log.Error().Err(update.Err()).Msgf("failed to mark message %s as sent", id)
	}
}
