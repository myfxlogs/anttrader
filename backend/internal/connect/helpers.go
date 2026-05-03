package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"anttrader/internal/interceptor"
)

func getUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userIDStr := interceptor.GetUserID(ctx)
	if userIDStr == "" {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}
	return uuid.Parse(userIDStr)
}

func requireAuth(ctx context.Context) error {
	userIDStr := interceptor.GetUserID(ctx)
	if userIDStr == "" {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}
	return nil
}
