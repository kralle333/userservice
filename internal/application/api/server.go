package api

import (
	"context"
	"google.golang.org/grpc"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	grpcproto "userservice/proto/grpc"
)

type Server struct {
	userController        *UserController
	healthCheckController *HealthCheckController
}

func NewServer(usersController *UserController, healthCheckController *HealthCheckController) *Server {
	return &Server{
		userController:        usersController,
		healthCheckController: healthCheckController,
	}
}

func (s *Server) RegisterGRPC(grpcServer *grpc.Server) {
	healthgrpc.RegisterHealthServer(grpcServer, s.healthCheckController)
	grpcproto.RegisterUserServiceServer(grpcServer, s.userController)
}

func (s *Server) Run() {
	ctx := context.Background()
	go s.healthCheckController.ServeOrdinaryHealthEndpoint()
	go s.healthCheckController.Run(ctx)

}
