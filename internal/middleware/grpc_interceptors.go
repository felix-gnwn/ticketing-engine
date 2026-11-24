package middleware

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCTraceInterceptor extracts X-Trace-ID from incoming gRPC metadata or generates a new UUID, injecting it into context using TraceIDKey
func GRPCTraceInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var traceID string

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			values := md.Get("x-trace-id")
			if len(values) > 0 && values[0] != "" {
				traceID = values[0]
			}
		}

		if traceID == "" {
			traceID = uuid.New().String()
		}

		ctx = context.WithValue(ctx, TraceIDKey, traceID)
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-trace-id", traceID))

		return handler(ctx, req)
	}
}

// GRPCLoggingInterceptor logs all incoming gRPC RPC calls with duration and status code
func GRPCLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		statusCode := status.Code(err)

		if err != nil {
			slog.ErrorContext(ctx, "gRPC request failed",
				slog.String("grpc_method", info.FullMethod),
				slog.String("status_code", statusCode.String()),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.String("error", err.Error()),
			)
		} else {
			slog.InfoContext(ctx, "gRPC request completed successfully",
				slog.String("grpc_method", info.FullMethod),
				slog.String("status_code", statusCode.String()),
				slog.Int64("duration_ms", duration.Milliseconds()),
			)
		}

		return resp, err
	}
}

// GRPCRecoveryInterceptor safely catches panics during RPC handler execution
func GRPCRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "gRPC server recovered from panic",
					slog.String("grpc_method", info.FullMethod),
					slog.Any("panic", r),
					slog.String("stack_trace", string(debug.Stack())),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}
