package middleware

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCAuthInterceptor извлекает пользователя из metadata-заголовка authorization.
// Ожидаемый формат:
// authorization: Bearer userID|signature
// Если заголовок отсутствует или невалиден, пользователь считается
// неаутентифицированным, и userID в контекст не записывается
func GRPCAuthInterceptor(secretKey string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		var userID string

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("authorization"); len(values) > 0 {
				token := strings.TrimSpace(values[0])

				const bearerPrefix = "bearer "
				if len(token) > len(bearerPrefix) &&
					strings.EqualFold(token[:len(bearerPrefix)], bearerPrefix) {
					token = strings.TrimSpace(token[len(bearerPrefix):])
				}

				if token != "" {
					id, ok := ValidateSignedUserID(secretKey, token)
					if !ok {
						return nil, status.Error(
							codes.Unauthenticated,
							"invalid authorization metadata",
						)
					}

					userID = id
				}
			}
		}

		if userID != "" {
			signedToken := SignUserID(secretKey, userID)
			_ = grpc.SetHeader(
				ctx,
				metadata.Pairs("authorization", "Bearer "+signedToken),
			)
		}

		ctx = context.WithValue(ctx, UserIDKey, userID)

		return handler(ctx, req)
	}
}

// GRPCLogging логирует unary gRPC-запросы
func GRPCLogging(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		logger.Info("gRPC request",
			zap.String("method", info.FullMethod),
			zap.String("code", status.Code(err).String()),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)

		return resp, err
	}
}
