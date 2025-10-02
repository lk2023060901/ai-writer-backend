package logger

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type mockServerInfo struct {
	fullMethod string
}

func (m *mockServerInfo) FullMethod() string {
	return m.fullMethod
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockServerStream) RecvMsg(msg interface{}) error {
	return nil
}

func TestUnaryServerInterceptor(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	interceptor := UnaryServerInterceptor(logger)

	tests := []struct {
		name        string
		ctx         context.Context
		req         interface{}
		handler     grpc.UnaryHandler
		wantErr     bool
		wantCode    codes.Code
	}{
		{
			name: "success",
			ctx:  context.Background(),
			req:  map[string]string{"key": "value"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return map[string]string{"result": "ok"}, nil
			},
			wantErr:  false,
			wantCode: codes.OK,
		},
		{
			name: "with request ID in metadata",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{
				"x-request-id": []string{"test-request-id"},
			}),
			req: map[string]string{"key": "value"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				// Verify request ID is in context
				requestID := GetRequestID(ctx)
				if requestID != "test-request-id" {
					t.Errorf("expected request ID 'test-request-id', got '%s'", requestID)
				}
				return map[string]string{"result": "ok"}, nil
			},
			wantErr:  false,
			wantCode: codes.OK,
		},
		{
			name: "internal error",
			ctx:  context.Background(),
			req:  map[string]string{"key": "value"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, status.Error(codes.Internal, "internal error")
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
		{
			name: "not found error",
			ctx:  context.Background(),
			req:  map[string]string{"key": "value"},
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, status.Error(codes.NotFound, "not found")
			},
			wantErr:  true,
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/Method",
			}

			resp, err := interceptor(tt.ctx, tt.req, info, tt.handler)

			if (err != nil) != tt.wantErr {
				t.Errorf("interceptor() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				st, _ := status.FromError(err)
				if st.Code() != tt.wantCode {
					t.Errorf("interceptor() code = %v, want %v", st.Code(), tt.wantCode)
				}
			}

			if !tt.wantErr && resp == nil {
				t.Error("interceptor() returned nil response")
			}
		})
	}
}

func TestUnaryServerInterceptorWithConfig(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	opts := GRPCInterceptorOptions{
		SkipMethods: []string{"/test.Service/SkipMethod"},
		LogPayload:  true,
		LogMetadata: true,
	}

	interceptor := UnaryServerInterceptorWithConfig(logger, opts)

	// Test skip method
	t.Run("skip method", func(t *testing.T) {
		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "response", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/SkipMethod",
		}

		_, err := interceptor(context.Background(), "request", info, handler)
		if err != nil {
			t.Errorf("interceptor() error = %v", err)
		}
		if !called {
			t.Error("handler was not called for skipped method")
		}
	})

	// Test with payload logging
	t.Run("log payload", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return map[string]string{"result": "ok"}, nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Method",
		}

		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
			"user-agent": []string{"test-agent"},
		})

		_, err := interceptor(ctx, map[string]string{"key": "value"}, info, handler)
		if err != nil {
			t.Errorf("interceptor() error = %v", err)
		}
	})
}

func TestStreamServerInterceptor(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	interceptor := StreamServerInterceptor(logger)

	tests := []struct {
		name     string
		ctx      context.Context
		handler  grpc.StreamHandler
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name: "success",
			ctx:  context.Background(),
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			},
			wantErr:  false,
			wantCode: codes.OK,
		},
		{
			name: "with error",
			ctx:  context.Background(),
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return status.Error(codes.Internal, "stream error")
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := &mockServerStream{ctx: tt.ctx}
			info := &grpc.StreamServerInfo{
				FullMethod:     "/test.Service/StreamMethod",
				IsClientStream: true,
				IsServerStream: true,
			}

			err := interceptor(nil, stream, info, tt.handler)

			if (err != nil) != tt.wantErr {
				t.Errorf("interceptor() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				st, _ := status.FromError(err)
				if st.Code() != tt.wantCode {
					t.Errorf("interceptor() code = %v, want %v", st.Code(), tt.wantCode)
				}
			}
		})
	}
}

func TestRecoveryInterceptor(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	interceptor := RecoveryInterceptor(logger)

	tests := []struct {
		name     string
		handler  grpc.UnaryHandler
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name: "no panic",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "response", nil
			},
			wantErr: false,
		},
		{
			name: "panic recovered",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				panic("test panic")
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/Method",
			}

			resp, err := interceptor(context.Background(), "request", info, tt.handler)

			if (err != nil) != tt.wantErr {
				t.Errorf("interceptor() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				st, _ := status.FromError(err)
				if st.Code() != tt.wantCode {
					t.Errorf("interceptor() code = %v, want %v", st.Code(), tt.wantCode)
				}
			}

			if !tt.wantErr && resp == nil {
				t.Error("interceptor() returned nil response")
			}
		})
	}
}

func TestRecoveryStreamInterceptor(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	interceptor := RecoveryStreamInterceptor(logger)

	tests := []struct {
		name     string
		handler  grpc.StreamHandler
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name: "no panic",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "panic recovered",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				panic("test stream panic")
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := &mockServerStream{ctx: context.Background()}
			info := &grpc.StreamServerInfo{
				FullMethod: "/test.Service/StreamMethod",
			}

			err := interceptor(nil, stream, info, tt.handler)

			if (err != nil) != tt.wantErr {
				t.Errorf("interceptor() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				st, _ := status.FromError(err)
				if st.Code() != tt.wantCode {
					t.Errorf("interceptor() code = %v, want %v", st.Code(), tt.wantCode)
				}
			}
		})
	}
}

func TestExtractRequestID(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "from context",
			ctx:  WithRequestID(context.Background(), "ctx-request-id"),
			want: "ctx-request-id",
		},
		{
			name: "from incoming metadata",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{
				"x-request-id": []string{"incoming-request-id"},
			}),
			want: "incoming-request-id",
		},
		{
			name: "from outgoing metadata",
			ctx: metadata.NewOutgoingContext(context.Background(), metadata.MD{
				"x-request-id": []string{"outgoing-request-id"},
			}),
			want: "outgoing-request-id",
		},
		{
			name: "no request ID",
			ctx:  context.Background(),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRequestID(tt.ctx)
			if got != tt.want {
				t.Errorf("extractRequestID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChainUnaryServer(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create multiple interceptors
	interceptor1 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = context.WithValue(ctx, "key1", "value1")
		return handler(ctx, req)
	}

	interceptor2 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = context.WithValue(ctx, "key2", "value2")
		return handler(ctx, req)
	}

	// Chain interceptors
	chained := ChainUnaryServer(interceptor1, interceptor2)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Verify both values are present
		if ctx.Value("key1") != "value1" {
			t.Error("key1 not found in context")
		}
		if ctx.Value("key2") != "value2" {
			t.Error("key2 not found in context")
		}
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err = chained(context.Background(), "request", info, handler)
	if err != nil {
		t.Errorf("chained interceptor error = %v", err)
	}
}

func TestChainStreamServer(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	callCount := 0

	// Create multiple interceptors
	interceptor1 := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		callCount++
		return handler(srv, ss)
	}

	interceptor2 := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		callCount++
		return handler(srv, ss)
	}

	// Chain interceptors
	chained := ChainStreamServer(interceptor1, interceptor2)

	handler := func(srv interface{}, ss grpc.ServerStream) error {
		callCount++
		return nil
	}

	stream := &mockServerStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	err = chained(nil, stream, info, handler)
	if err != nil {
		t.Errorf("chained interceptor error = %v", err)
	}

	// Verify all interceptors and handler were called
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWrappedServerStream(t *testing.T) {
	originalCtx := context.Background()
	newCtx := context.WithValue(originalCtx, "test-key", "test-value")

	originalStream := &mockServerStream{ctx: originalCtx}
	wrapped := &wrappedServerStream{
		ServerStream: originalStream,
		ctx:          newCtx,
	}

	// Verify the wrapped stream returns the new context
	if wrapped.Context() != newCtx {
		t.Error("wrapped stream did not return the correct context")
	}

	if wrapped.Context().Value("test-key") != "test-value" {
		t.Error("wrapped stream context does not contain expected value")
	}
}

// Benchmark tests
func BenchmarkUnaryServerInterceptor(b *testing.B) {
	logger, _ := New(DefaultConfig())
	defer logger.Sync()

	interceptor := UnaryServerInterceptor(logger)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(context.Background(), "request", info, handler)
	}
}

func BenchmarkUnaryServerInterceptorWithError(b *testing.B) {
	logger, _ := New(DefaultConfig())
	defer logger.Sync()

	interceptor := UnaryServerInterceptor(logger)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, errors.New("test error")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(context.Background(), "request", info, handler)
	}
}
