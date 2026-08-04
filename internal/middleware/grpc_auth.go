package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCAuthInterceptor извлекает пользователя из metadata-заголовка authorization.
// Ожидаемый формат:
// authorization: Bearer userID|signature
// Если заголовок отсутствует, interceptor создаёт нового пользователя и
// возвращает подписанный токен в response metadata:
// authorization: Bearer userID|signature
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
					if id, ok := ValidateSignedUserID(secretKey, token); ok {
						userID = id
					} else {
						return nil, status.Error(
							codes.Unauthenticated,
							"invalid authorization metadata",
						)
					}
				}
			}
		}

		if userID == "" {
			userID = uuid.New().String()
		}

		signedToken := SignUserID(secretKey, userID)

		_ = grpc.SetHeader(
			ctx,
			metadata.Pairs("authorization", "Bearer "+signedToken),
		)

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
