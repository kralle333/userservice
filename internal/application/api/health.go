package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net/http"
	"sync"
	"time"
	"userservice/internal/config"
	"userservice/internal/infrastructure/health"
)

type HealthCheckController struct {
	grpc_health_v1.UnimplementedHealthServer
	config              config.HealthCheckerConfig
	statusMapLock       sync.Mutex
	statusMap           map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
	healthReportChannel chan health.Report
	startTime           time.Time
}

func NewHealthCheckController(config config.HealthCheckerConfig) *HealthCheckController {
	status := make(map[string]grpc_health_v1.HealthCheckResponse_ServingStatus)

	return &HealthCheckController{
		config:              config,
		statusMapLock:       sync.Mutex{},
		healthReportChannel: make(chan health.Report),
		statusMap:           status,
		startTime:           time.Now().UTC(),
	}
}

func convertToGRPCStatus(status health.Status) grpc_health_v1.HealthCheckResponse_ServingStatus {
	switch status {
	case health.StatusServing:
		return grpc_health_v1.HealthCheckResponse_SERVING
	case health.StatusNotServing:
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	case health.StatusUnknown:
		return grpc_health_v1.HealthCheckResponse_UNKNOWN
	}
	return grpc_health_v1.HealthCheckResponse_UNKNOWN
}

func (h *HealthCheckController) RegisterHealthCheckable(ctx context.Context, service health.Checkable) {
	h.setStatus(service.GetName(), grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN)
	service.RunHealthCheck(ctx, h.healthReportChannel, time.Duration(h.config.HealthCheckTickerIntervalSeconds)*time.Second)
}

func (h *HealthCheckController) setStatus(service string, serving grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.statusMapLock.Lock()
	defer h.statusMapLock.Unlock()
	h.statusMap[service] = serving
}

func (h *HealthCheckController) getStatus(serviceName string) grpc_health_v1.HealthCheckResponse_ServingStatus {
	h.statusMapLock.Lock()
	defer h.statusMapLock.Unlock()
	_, ok := h.statusMap[serviceName]
	if !ok {
		log.Warn().Msgf("serviceName %s is not found in status map! check constructor setting of map - setting value in map", serviceName)
		h.statusMap[serviceName] = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}
	return h.statusMap[serviceName]
}

func (h *HealthCheckController) Run(ctx context.Context) {

	sleepInterval := time.Duration(h.config.TickerIntervalSeconds) * time.Second

	for {
		log.Info().Msg("HealthChecker: checking status")
		select {
		case report := <-h.healthReportChannel:
			h.setStatus(report.ServiceName, convertToGRPCStatus(report.Status))
		case <-ctx.Done():
			break
		}
		time.Sleep(sleepInterval)
	}
}

func (h *HealthCheckController) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	h.statusMapLock.Lock()
	defer h.statusMapLock.Unlock()

	resp := &grpc_health_v1.HealthCheckResponse{}
	resp.Status = h.getStatus(req.Service)
	return resp, nil
}

func (h *HealthCheckController) Watch(req *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {

	for {
		time.Sleep(5 * time.Second)
		err := server.Send(&grpc_health_v1.HealthCheckResponse{Status: h.getStatus(req.Service)})
		if err != nil {
			return err
		}
	}
}

func (h *HealthCheckController) ServeOrdinaryHealthEndpoint() {
	type OrdinaryHealthCheckResponse struct {
		Uptime string   `json:"uptime"`
		Status []string `json:"status"`
	}
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {

		h.statusMapLock.Lock()
		defer h.statusMapLock.Unlock()

		resp := OrdinaryHealthCheckResponse{
			Status: []string{},
			Uptime: fmt.Sprintf("up for %s", time.Since(h.startTime)),
		}
		for s, status := range h.statusMap {
			resp.Status = append(resp.Status, fmt.Sprintf("%s: %s", s, status.String()))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	portString := fmt.Sprintf(":%d", h.config.OrdinaryHealthCheckListeningPort)
	log.Info().Msgf("Serving healtcheck at %s/healthz", portString)
	err := http.ListenAndServe(portString, nil)
	if err != nil {
		log.Panic().Msgf("failed to listen on port %s", portString)
	}
}
