package grpcmiddleware

import (
	"context"
	"strings"

	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func RequestIDUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		requestCtx := requestid.WithContext(ctx, incomingRequestID(ctx))
		return next(requestCtx, req)
	}
}

func incomingRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		for _, value := range md.Get(requestid.MetadataKey) {
			requestID := strings.TrimSpace(value)
			if requestID != "" {
				return requestID
			}
		}
	}

	return requestid.Generate()
}
