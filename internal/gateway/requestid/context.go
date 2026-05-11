package requestid

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	Header      = "X-Request-ID"
	MetadataKey = "x-request-id"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func WithContext(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, requestIDKey, requestID)
}

func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	requestID, _ := ctx.Value(requestIDKey).(string)
	return requestID
}

func WithOutgoingMetadata(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	requestID := strings.TrimSpace(FromContext(ctx))
	if requestID == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, MetadataKey, requestID)
}
