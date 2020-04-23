package observer

import "context"

type contextKey string

const (
	uuidContextKey     = contextKey("UUID")
	observerContextKey = contextKey("Observer")
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

// ContextWithObserver returns a new context that holds a reference to an observer.
func ContextWithObserver(ctx context.Context, observer *Observer) context.Context {
	return context.WithValue(ctx, observerContextKey, observer)
}

// FromContext returns an observer set on a context.
// If no observer found on the context, the singleton observer will be returned!
func FromContext(ctx context.Context) *Observer {
	val := ctx.Value(observerContextKey)
	if observer, ok := val.(*Observer); ok {
		return observer
	}

	// Return the singleton observer as the default
	return singleton
}
