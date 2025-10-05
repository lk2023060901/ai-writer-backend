package service

import (
	"context"

	pb "github.com/lk2023060901/ai-writer-backend/api/auth/v1"
	"github.com/lk2023060901/ai-writer-backend/internal/auth/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/validator"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// GRPCAuthService gRPC 认证服务实现
type GRPCAuthService struct {
	pb.UnimplementedAuthServiceServer
	authUC *biz.AuthUseCase
	logger *logger.Logger
}

// NewGRPCAuthService 创建 gRPC 认证服务
func NewGRPCAuthService(authUC *biz.AuthUseCase, log *logger.Logger) *GRPCAuthService {
	return &GRPCAuthService{
		authUC: authUC,
		logger: log,
	}
}

// Register 用户注册
func (s *GRPCAuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// 参数验证
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "name, email and password are required")
	}

	if len(req.Password) < 8 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}

	user, err := s.authUC.Register(ctx, req.Name, req.Email, req.Password)
	if err != nil {
		s.logger.Error("failed to register user", zap.Error(err), zap.String("email", req.Email))

		if err == biz.ErrEmailAlreadyExists {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}

		return nil, status.Error(codes.Internal, "failed to register user")
	}

	return &pb.RegisterResponse{
		UserId:  user.ID,
		Email:   user.Email,
		Message: "Registration successful. Please verify your email.",
	}, nil
}

// Login 用户登录
func (s *GRPCAuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	// 从 gRPC peer 获取客户端 IP
	ip := "127.0.0.1" // 默认值
	if p, ok := peer.FromContext(ctx); ok {
		ip = validator.GetIPOrDefault(p.Addr.String(), "127.0.0.1")
	}

	result, err := s.authUC.Login(ctx, req.Email, req.Password, ip, false) // gRPC登录暂不支持rememberMe
	if err != nil {
		s.logger.Warn("login failed",
			zap.Error(err),
			zap.String("email", req.Email),
			zap.String("ip", ip))

		switch err {
		case biz.ErrInvalidCredentials:
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		case biz.ErrAccountLocked:
			return nil, status.Error(codes.PermissionDenied, "account locked due to too many failed attempts")
		default:
			return nil, status.Error(codes.Internal, "login failed")
		}
	}

	resp := &pb.LoginResponse{
		Require_2Fa: result.Require2FA,
	}

	if result.Require2FA {
		// 需要 2FA 验证
		resp.PendingAuthId = result.PendingAuthID
	} else {
		// 不需要 2FA，直接返回 tokens
		if result.Tokens != nil {
			resp.Tokens = &pb.TokenPair{
				AccessToken:  result.Tokens.AccessToken,
				RefreshToken: result.Tokens.RefreshToken,
				ExpiresIn:    int64(result.Tokens.ExpiresIn),
			}
		}
	}

	return resp, nil
}

// Verify2FA 验证双因子认证代码
func (s *GRPCAuthService) Verify2FA(ctx context.Context, req *pb.Verify2FARequest) (*pb.Verify2FAResponse, error) {
	if req.PendingAuthId == "" || req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "pending_auth_id and code are required")
	}

	if len(req.Code) != 6 {
		return nil, status.Error(codes.InvalidArgument, "code must be 6 digits")
	}

	result, err := s.authUC.Verify2FA(ctx, req.PendingAuthId, req.Code)
	if err != nil {
		s.logger.Warn("2FA verification failed",
			zap.Error(err),
			zap.String("pending_auth_id", req.PendingAuthId))

		switch err {
		case biz.ErrPendingAuthNotFound, biz.ErrPendingAuthExpired:
			return nil, status.Error(codes.NotFound, "pending auth not found or expired, please login again")
		case biz.ErrTooManyAttempts:
			return nil, status.Error(codes.ResourceExhausted, "too many verification attempts, please login again")
		case biz.ErrInvalid2FACode:
			return nil, status.Error(codes.Unauthenticated, "invalid 2FA code")
		default:
			return nil, status.Error(codes.Internal, "2FA verification failed")
		}
	}

	if result.Tokens == nil {
		return nil, status.Error(codes.Internal, "failed to generate tokens")
	}

	return &pb.Verify2FAResponse{
		Tokens: &pb.TokenPair{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			ExpiresIn:    int64(result.Tokens.ExpiresIn),
		},
	}, nil
}

// RefreshToken 刷新访问令牌
func (s *GRPCAuthService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	tokens, err := s.authUC.RefreshAccessToken(ctx, req.RefreshToken)
	if err != nil {
		s.logger.Warn("token refresh failed", zap.Error(err))

		if err == biz.ErrInvalidToken {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired refresh token")
		}

		return nil, status.Error(codes.Internal, "token refresh failed")
	}

	return &pb.RefreshTokenResponse{
		Tokens: &pb.TokenPair{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresIn:    int64(tokens.ExpiresIn),
		},
	}, nil
}

// Enable2FA 启用双因子认证
func (s *GRPCAuthService) Enable2FA(ctx context.Context, req *pb.Enable2FARequest) (*pb.Enable2FAResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	setup, err := s.authUC.Enable2FA(ctx, req.UserId)
	if err != nil {
		s.logger.Error("failed to enable 2FA", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to enable 2FA")
	}

	return &pb.Enable2FAResponse{
		Setup: &pb.TwoFactorSetup{
			Secret:      setup.Secret,
			QrCodeUrl:   "/auth/2fa/qrcode",
			BackupCodes: setup.BackupCodes,
		},
	}, nil
}

// GetQRCode 获取二维码
func (s *GRPCAuthService) GetQRCode(ctx context.Context, req *pb.GetQRCodeRequest) (*pb.GetQRCodeResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	setup, err := s.authUC.Enable2FA(ctx, req.UserId)
	if err != nil {
		s.logger.Error("failed to get QR code", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get QR code")
	}

	return &pb.GetQRCodeResponse{
		QrCodeImage: setup.QRCode,
	}, nil
}

// Confirm2FA 确认启用双因子认证
func (s *GRPCAuthService) Confirm2FA(ctx context.Context, req *pb.Confirm2FARequest) (*pb.Confirm2FAResponse, error) {
	if req.UserId == "" || req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and code are required")
	}

	if len(req.Code) != 6 {
		return nil, status.Error(codes.InvalidArgument, "code must be 6 digits")
	}

	if err := s.authUC.Confirm2FA(ctx, req.UserId, req.Code); err != nil {
		s.logger.Warn("2FA confirmation failed", zap.Error(err), zap.String("user_id", req.UserId))

		if err == biz.ErrInvalid2FACode {
			return nil, status.Error(codes.Unauthenticated, "invalid verification code")
		}

		return nil, status.Error(codes.Internal, "failed to confirm 2FA")
	}

	return &pb.Confirm2FAResponse{
		Success: true,
		Message: "2FA enabled successfully",
	}, nil
}

// Disable2FA 禁用双因子认证
func (s *GRPCAuthService) Disable2FA(ctx context.Context, req *pb.Disable2FARequest) (*pb.Disable2FAResponse, error) {
	if req.UserId == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and password are required")
	}

	// 注意：当前 Disable2FA 接收的是验证码，需要改为验证密码
	// 这里暂时用验证码代替，实际应该先验证密码再禁用
	if err := s.authUC.Disable2FA(ctx, req.UserId, req.Password); err != nil {
		s.logger.Warn("2FA disable failed", zap.Error(err), zap.String("user_id", req.UserId))

		if err == biz.ErrInvalid2FACode {
			return nil, status.Error(codes.Unauthenticated, "invalid password or 2FA code")
		}

		return nil, status.Error(codes.Internal, "failed to disable 2FA")
	}

	return &pb.Disable2FAResponse{
		Success: true,
		Message: "2FA disabled successfully",
	}, nil
}
