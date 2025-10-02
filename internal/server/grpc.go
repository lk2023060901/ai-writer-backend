package server

import (
	"fmt"
	"net"

	pb "github.com/lk2023060901/ai-writer-backend/api/auth/v1"
	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer gRPC 服务器
type GRPCServer struct {
	config      *conf.Config
	logger      *logger.Logger
	grpcServer  *grpc.Server
	authService pb.AuthServiceServer
}

// NewGRPCServer 创建 gRPC 服务器
func NewGRPCServer(
	config *conf.Config,
	log *logger.Logger,
	authService pb.AuthServiceServer,
) *GRPCServer {
	// 创建 gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// 可以在这里添加拦截器：日志、认证、限流等
		),
	)

	// 注册服务
	pb.RegisterAuthServiceServer(grpcServer, authService)

	// 启用反射（用于 grpcurl 等工具）
	reflection.Register(grpcServer)

	return &GRPCServer{
		config:      config,
		logger:      log,
		grpcServer:  grpcServer,
		authService: authService,
	}
}

// Start 启动 gRPC 服务器
func (s *GRPCServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.GRPCPort)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("starting gRPC server", zap.String("addr", addr))

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop 停止 gRPC 服务器
func (s *GRPCServer) Stop() {
	s.logger.Info("stopping gRPC server")
	s.grpcServer.GracefulStop()
}
