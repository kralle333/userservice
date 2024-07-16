package db

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
	"userservice/internal/config"
	"userservice/internal/kafkamessage"
	timeutil "userservice/internal/util/time"
)

var ErrNoPendingMessage = errors.New("no pending message found")

type repo struct {
	collection          *mongo.Collection
	transactionExecutor TransactionExecutor
	config              config.OutboxConfig
}
type KafkaMessageOutboxDBRepo interface {
	GetPendingMessage(ctx context.Context) (kafkamessage.InternalMessage, error)
	RetryMessage(ctx context.Context, id string)
	MarkMessageSent(ctx context.Context, id string)
}

func NewKafkaMessageOutboxDBRepo(collection *mongo.Collection, transactionExecutor TransactionExecutor, config config.OutboxConfig) KafkaMessageOutboxDBRepo {
	return &repo{collection: collection, transactionExecutor: transactionExecutor, config: config}
}

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
func (k *repo) setState(ctx context.Context, msgID string, state KafkaOutboxMessageState) (KafkaOutboxMessage, error) {

	upsert := false
	returnDocument := options.After
	opts := options.FindOneAndUpdateOptions{
		Upsert:         &upsert,
		ReturnDocument: &returnDocument,
	}

	setFilter := bson.D{{"$set", bson.D{{"state", state.String()}}}}

	var updatedMessage KafkaOutboxMessage
	err := k.collection.FindOneAndUpdate(ctx, bson.D{{"msg_id", msgID}}, setFilter, &opts).Decode(&updatedMessage)
	if err != nil {
		return KafkaOutboxMessage{}, err
	}
	return updatedMessage, nil
}
func (k *repo) incrementNumberOfRetries(ctx context.Context, msgID string) (KafkaOutboxMessage, error) {

	// first find the document
	// then update the document with the right values (retries,state,next_retry)

	findFilter := bson.D{{"msg_id", msgID}}
	message := k.collection.FindOne(ctx, findFilter)
	var decoded KafkaOutboxMessage
	err := message.Decode(&decoded)
	if err != nil {
		return KafkaOutboxMessage{}, err
	}
	if decoded.State != KafkaOutboxMessageStateProcessing.String() {
		return KafkaOutboxMessage{}, err
	}
	attempts := decoded.Retries + 1
	a := bson.D{}
	a = append(a, bson.E{Key: "retries", Value: attempts})
	a = append(a, bson.E{Key: "next_retry", Value: calculateNextAttempt(k.config.BaseRetryTimeSeconds, k.config.MaxRetryTimeSeconds, int(attempts))})

	upsert := false
	returnDocument := options.After
	opts := options.FindOneAndUpdateOptions{
		Upsert:         &upsert,
		ReturnDocument: &returnDocument,
	}

	var updatedMessage KafkaOutboxMessage
	err = k.collection.FindOneAndUpdate(ctx, findFilter, a, &opts).Decode(&updatedMessage)
	if err != nil {
		return KafkaOutboxMessage{}, err
	}
	return updatedMessage, nil
}

func (k *repo) GetPendingMessage(ctx context.Context) (kafkamessage.InternalMessage, error) {

	var pendingMessage KafkaOutboxMessage
	err := k.transactionExecutor.ExecuteInTransaction(ctx, func(sessionContext mongo.SessionContext) error {

		// find message with correct state:
		findFilter := bson.D{{"$and", bson.A{
			bson.D{{"state", KafkaOutboxMessageStateWaiting}},
			bson.D{{"next_retry", bson.D{{"$lt", time.Now().UTC()}}}},
		}}}

		setFilter := bson.D{{
			"$set", bson.D{{"state", KafkaOutboxMessageStateProcessing}}},
		}

		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		var processingMessage KafkaOutboxMessage
		err := k.collection.FindOneAndUpdate(sessionContext, findFilter, setFilter, opts).Decode(&processingMessage)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return ErrNoPendingMessage
			}
			return err
		}
		// then we set the retry fields and next_attempt_at
		_, err = k.incrementNumberOfRetries(sessionContext, processingMessage.MsgID)
		pendingMessage, err = k.setState(sessionContext, processingMessage.MsgID, KafkaOutboxMessageStateProcessing)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return kafkamessage.InternalMessage{}, err
	}

	var toSend kafkamessage.InternalMessage
	err = json.Unmarshal(pendingMessage.Data, &toSend)
	if err != nil {
		return kafkamessage.InternalMessage{}, err
	}
	return toSend, nil
}
func (k *repo) RetryMessage(ctx context.Context, id string) {
	err := k.transactionExecutor.ExecuteInTransaction(ctx, func(sessionContext mongo.SessionContext) error {
		_, err := k.incrementNumberOfRetries(sessionContext, id)
		if err != nil {
			return err
		}
		_, err = k.setState(sessionContext, id, KafkaOutboxMessageStateWaiting)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Warn().Err(err).Msgf("failed to retry message %s", id)
	}
}
func (k *repo) MarkMessageSent(ctx context.Context, id string) {
	a := bson.D{}
	a = append(a, bson.E{Key: "sent_at", Value: timeutil.DBNow()})
	a = append(a, bson.E{Key: "state", Value: KafkaOutboxMessageStateFinished})
	setFilter := bson.D{{"$set", a}}

	update := k.collection.FindOneAndUpdate(ctx, bson.D{{"msg_id", id}}, setFilter)
	if update.Err() != nil {
		log.Error().Err(update.Err()).Msgf("failed to mark message %s as sent", id)
	}
}
