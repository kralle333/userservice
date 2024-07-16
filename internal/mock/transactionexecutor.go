package mock

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionExecutorMock struct {
}

func (t TransactionExecutorMock) ExecuteInTransaction(ctx context.Context, inner func(sessionContext mongo.SessionContext) error) error {
	return nil
}

func NewTransactionExecutorMock() *TransactionExecutorMock {
	return &TransactionExecutorMock{}
}
