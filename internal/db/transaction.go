package db

import (
	"context"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionExecutor interface {
	ExecuteInTransaction(ctx context.Context, inner func(mongo.SessionContext) error) error
}

func (m *MongoDBRepo) ExecuteInTransaction(ctx context.Context, inner func(mongo.SessionContext) error) error {
	session, err := m.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	err = mongo.WithSession(ctx, session, func(sessionContext mongo.SessionContext) error {
		log.Info().Msgf("TransactionExecutor: starting transaction")
		if err := session.StartTransaction(); err != nil {
			return err
		}

		log.Info().Msgf("TransactionExecutor: executing inner function")
		if err := inner(sessionContext); err != nil {
			if abortErr := session.AbortTransaction(sessionContext); abortErr != nil {
				return abortErr
			}
			return err
		}

		log.Info().Msgf("TransactionExecutor: commiting transaction")
		if err := session.CommitTransaction(sessionContext); err != nil {
			return err
		}

		return nil
	})

	return err
}
