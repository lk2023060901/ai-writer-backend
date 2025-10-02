package logger

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCInterceptorOptions configures the gRPC interceptor
type GRPCInterceptorOptions struct {
	// SkipMethods is a list of methods to skip logging (e.g., "/grpc.health.v1.Health/Check")
	SkipMethods []string
	// LogPayload enables logging request and response payload
	LogPayload bool
	// LogMetadata enables logging gRPC metadata
	LogMetadata bool
}

// UnaryServerInterceptor returns a new unary server interceptor for logging
func UnaryServerInterceptor(logger *Logger) grpc.UnaryServerInterceptor {
	return UnaryServerInterceptorWithConfig(logger, GRPCInterceptorOptions{})
}

// UnaryServerInterceptorWithConfig returns a new unary server interceptor with custom config
func UnaryServerInterceptorWithConfig(logger *Logger, opts GRPCInterceptorOptions) grpc.UnaryServerInterceptor {
	// Build skip methods map for fast lookup
	skipMethods := make(map[string]bool)
	for _, method := range opts.SkipMethods {
		skipMethods[method] = true
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if method should be skipped
		if skipMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract or generate request ID
		requestID := extractRequestID(ctx)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to context
		ctx = WithRequestID(ctx, requestID)

		// Add request ID to outgoing metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

		// Start timer
		start := time.Now()

		// Build initial log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.String("service", path.Dir(info.FullMethod)[1:]),
			zap.String("rpc", path.Base(info.FullMethod)),
		}

		// Add metadata if enabled
		if opts.LogMetadata {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				fields = append(fields, zap.Any("metadata", md))
			}
		}

		// Add payload if enabled
		if opts.LogPayload {
			fields = append(fields, zap.Any("request", req))
		}

		// Call the handler
		resp, err := handler(ctx, req)

		// Calculate latency
		latency := time.Since(start)
		fields = append(fields, zap.Duration("latency", latency))

		// Get status code
		st, _ := status.FromError(err)
		fields = append(fields, zap.String("code", st.Code().String()))

		// Add response payload if enabled and no error
		if opts.LogPayload && err == nil && resp != nil {
			fields = append(fields, zap.Any("response", resp))
		}

		// Add error if present
		if err != nil {
			fields = append(fields, zap.Error(err))
			fields = append(fields, zap.String("message", st.Message()))
		}

		// Log based on status code
		switch st.Code() {
		case codes.OK:
			logger.Info("gRPC call", fields...)
		case codes.Canceled, codes.DeadlineExceeded, codes.NotFound:
			logger.Warn("gRPC call", fields...)
		default:
			logger.Error("gRPC call", fields...)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a new stream server interceptor for logging
func StreamServerInterceptor(logger *Logger) grpc.StreamServerInterceptor {
	return StreamServerInterceptorWithConfig(logger, GRPCInterceptorOptions{})
}

// StreamServerInterceptorWithConfig returns a new stream server interceptor with custom config
func StreamServerInterceptorWithConfig(logger *Logger, opts GRPCInterceptorOptions) grpc.StreamServerInterceptor {
	// Build skip methods map for fast lookup
	skipMethods := make(map[string]bool)
	for _, method := range opts.SkipMethods {
		skipMethods[method] = true
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if method should be skipped
		if skipMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		// Extract or generate request ID
		ctx := ss.Context()
		requestID := extractRequestID(ctx)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to context
		ctx = WithRequestID(ctx, requestID)

		// Wrap the stream with our context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Start timer
		start := time.Now()

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.String("service", path.Dir(info.FullMethod)[1:]),
			zap.String("rpc", path.Base(info.FullMethod)),
			zap.Bool("is_client_stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
		}

		// Add metadata if enabled
		if opts.LogMetadata {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				fields = append(fields, zap.Any("metadata", md))
			}
		}

		// Call the handler
		err := handler(srv, wrappedStream)

		// Calculate latency
		latency := time.Since(start)
		fields = append(fields, zap.Duration("latency", latency))

		// Get status code
		st, _ := status.FromError(err)
		fields = append(fields, zap.String("code", st.Code().String()))

		// Add error if present
		if err != nil {
			fields = append(fields, zap.Error(err))
			fields = append(fields, zap.String("message", st.Message()))
		}

		// Log based on status code
		switch st.Code() {
		case codes.OK:
			logger.Info("gRPC stream", fields...)
		case codes.Canceled, codes.DeadlineExceeded:
			logger.Warn("gRPC stream", fields...)
		default:
			logger.Error("gRPC stream", fields...)
		}

		return err
	}
}

// UnaryClientInterceptor returns a new unary client interceptor for logging
func UnaryClientInterceptor(logger *Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Extract or generate request ID
		requestID := extractRequestID(ctx)
		if requestID == "" {
			requestID = uuid.New().String()
			ctx = WithRequestID(ctx, requestID)
		}

		// Add request ID to outgoing metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

		// Start timer
		start := time.Now()

		// Call the invoker
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		st, _ := status.FromError(err)

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("target", cc.Target()),
			zap.Duration("latency", latency),
			zap.String("code", st.Code().String()),
		}

		// Add error if present
		if err != nil {
			fields = append(fields, zap.Error(err))
			fields = append(fields, zap.String("message", st.Message()))
		}

		// Log based on status code
		switch st.Code() {
		case codes.OK:
			logger.Info("gRPC client call", fields...)
		case codes.Canceled, codes.DeadlineExceeded, codes.NotFound:
			logger.Warn("gRPC client call", fields...)
		default:
			logger.Error("gRPC client call", fields...)
		}

		return err
	}
}

// StreamClientInterceptor returns a new stream client interceptor for logging
func StreamClientInterceptor(logger *Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Extract or generate request ID
		requestID := extractRequestID(ctx)
		if requestID == "" {
			requestID = uuid.New().String()
			ctx = WithRequestID(ctx, requestID)
		}

		// Add request ID to outgoing metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

		// Start timer
		start := time.Now()

		// Call the streamer
		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		st, _ := status.FromError(err)

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("target", cc.Target()),
			zap.Duration("latency", latency),
			zap.String("code", st.Code().String()),
			zap.Bool("client_streams", desc.ClientStreams),
			zap.Bool("server_streams", desc.ServerStreams),
		}

		// Add error if present
		if err != nil {
			fields = append(fields, zap.Error(err))
			fields = append(fields, zap.String("message", st.Message()))
		}

		// Log based on status code
		switch st.Code() {
		case codes.OK:
			logger.Info("gRPC client stream", fields...)
		case codes.Canceled, codes.DeadlineExceeded:
			logger.Warn("gRPC client stream", fields...)
		default:
			logger.Error("gRPC client stream", fields...)
		}

		return clientStream, err
	}
}

// extractRequestID extracts request ID from gRPC metadata
func extractRequestID(ctx context.Context) string {
	// Try to get from context first
	if requestID := GetRequestID(ctx); requestID != "" {
		return requestID
	}

	// Try to get from incoming metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-request-id"); len(values) > 0 {
			return values[0]
		}
	}

	// Try to get from outgoing metadata
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if values := md.Get("x-request-id"); len(values) > 0 {
			return values[0]
		}
	}

	return ""
}

// wrappedServerStream wraps grpc.ServerStream with custom context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// RecoveryInterceptor returns a unary server interceptor for panic recovery
func RecoveryInterceptor(logger *Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Get request ID from context
				requestID := GetRequestID(ctx)

				// Log the panic
				logger.Error("gRPC panic recovered",
					zap.String("request_id", requestID),
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.Stack("stacktrace"),
				)

				// Return Internal error
				err = status.Errorf(codes.Internal, "internal server error: %v", r)
			}
		}()

		return handler(ctx, req)
	}
}

// RecoveryStreamInterceptor returns a stream server interceptor for panic recovery
func RecoveryStreamInterceptor(logger *Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Get request ID from context
				requestID := GetRequestID(ss.Context())

				// Log the panic
				logger.Error("gRPC stream panic recovered",
					zap.String("request_id", requestID),
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.Stack("stacktrace"),
				)

				// Return Internal error
				err = status.Errorf(codes.Internal, "internal server error: %v", r)
			}
		}()

		return handler(srv, ss)
	}
}

// ChainUnaryServer creates a single interceptor from multiple unary interceptors
func ChainUnaryServer(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		buildChain := func(current grpc.UnaryServerInterceptor, next grpc.UnaryHandler) grpc.UnaryHandler {
			return func(ctx context.Context, req interface{}) (interface{}, error) {
				return current(ctx, req, info, next)
			}
		}

		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}

		return chain(ctx, req)
	}
}

// ChainStreamServer creates a single interceptor from multiple stream interceptors
func ChainStreamServer(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		buildChain := func(current grpc.StreamServerInterceptor, next grpc.StreamHandler) grpc.StreamHandler {
			return func(srv interface{}, ss grpc.ServerStream) error {
				return current(srv, ss, info, next)
			}
		}

		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}

		return chain(srv, ss)
	}
}

// Example usage:
// Create a gRPC server with logging interceptors
func ExampleGRPCServer(logger *Logger) *grpc.Server {
	// Chain multiple interceptors
	unaryInterceptor := ChainUnaryServer(
		RecoveryInterceptor(logger),
		UnaryServerInterceptor(logger),
	)

	streamInterceptor := ChainStreamServer(
		RecoveryStreamInterceptor(logger),
		StreamServerInterceptor(logger),
	)

	// Create server with interceptors
	server := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	return server
}

// Example usage:
// Create a gRPC client with logging interceptors
func ExampleGRPCClient(logger *Logger, target string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(
		target,
		grpc.WithUnaryInterceptor(UnaryClientInterceptor(logger)),
		grpc.WithStreamInterceptor(StreamClientInterceptor(logger)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	return conn, nil
}
