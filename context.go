package observer

import "context"

type contextKey string

const (
	uuidContextKey = contextKey("UUID")
)

// ContextWithUUID creates a new context with a uuid.
func ContextWithUUID(ctx context.Context, uuid string) context.Context {
	return context.WithValue(ctx, uuidContextKey, uuid)
}

// UUIDFromContext retrieves a uuid from a context.
func UUIDFromContext(ctx context.Context) (string, bool) {
	uuid, ok := ctx.Value(uuidContextKey).(string)
	return uuid, ok
}
