package health

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type mongoDBHealthCheckable struct {
	mongoDBConn *mongo.Client
}

func NewMongoDBHealthCheckable(mongoDBConn *mongo.Client) Checkable {
	return &mongoDBHealthCheckable{mongoDBConn: mongoDBConn}
}

func (m *mongoDBHealthCheckable) Ping(ctx context.Context, timeoutDuration time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	err := m.mongoDBConn.Ping(ctxWithTimeout, nil)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoDBHealthCheckable) GetName() string {
	return "MongoDB"
}

func (m *mongoDBHealthCheckable) RunHealthCheck(checkContext context.Context, reportChannel chan Report, checkTimeInterval time.Duration) {

	go func() {
		ticker := time.NewTicker(checkTimeInterval)
		for {
			select {
			case <-checkContext.Done():
			case <-ticker.C:
				err := m.Ping(checkContext, 5*time.Second)
				if err != nil {
					reportChannel <- NewHealthReport(m, StatusNotServing)
				} else {
					reportChannel <- NewHealthReport(m, StatusServing)
				}
			}
		}
	}()
}
