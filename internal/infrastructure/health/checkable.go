package health

import (
	"context"
	"time"
)

type Status string

const (
	StatusServing    Status = "SERVING"
	StatusNotServing Status = "NOT_SERVING"
	StatusUnknown    Status = "UNKNOWN"
)

type Checkable interface {
	GetName() string
	RunHealthCheck(checkContext context.Context, reportChannel chan Report, checkTimeInterval time.Duration)
}

type Report struct {
	ServiceName string
	Status      Status
}

func NewHealthReport(c Checkable, status Status) Report {
	return Report{c.GetName(), status}
}
