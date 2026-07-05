package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		if err != nil {
			st, _ := status.FromError(err)
			logger.Error("gRPC request failed",
				slog.String("method", info.FullMethod),
				slog.String("code", st.Code().String()),
				slog.String("message", st.Message()),
				slog.Duration("duration", duration),
				slog.Any("error", err),
			)
		} else {
			logger.Info("gRPC request success",
				slog.String("method", info.FullMethod),
				slog.Duration("duration", duration),
			)
		}

		return resp, err
	}
}
