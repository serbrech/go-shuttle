package message

import (
	"context"
	"time"

	servicebus "github.com/Azure/azure-service-bus-go"
	"github.com/Azure/go-shuttle/tracing"
	"github.com/pkg/errors"
)

// RetryLater waits for the given duration before abandoning the lock.
// Consider examining/increasing your LockDuration/MaxDeliveryCount before using.
// Undefined behavior if RetryLater is passed a duration that puts the message handling past the lock duration (which has a max of 5 minutes)
func RetryLater(retryAfter time.Duration) Handler {
	return &retryLaterHandler{
		retryAfter: retryAfter,
	}
}

type retryLaterHandler struct {
	retryAfter time.Duration
}

func (r *retryLaterHandler) Do(ctx context.Context, orig Handler, message *servicebus.Message) Handler {
	go func() {
		ctx, span := tracing.StartSpanFromMessageAndContext(ctx, "go-shuttle.retryLater.Do", message)
		defer span.End()

		select {
		// TODO this can go past lock duration pretty easily
		// Ideally we'd also timeout at dequeue time + lockdurtaion - 1 second but don't have access to dequeue time
		// Maybe we can use context.WithTimeout when we recieve the message?
		case <-ctx.Done():
			span.Logger().Error(errors.Wrap(ctx.Err(), "Retry context expired"))
		case <-time.After(r.retryAfter):
			Abandon().Do(ctx, orig, message)
		}
	}()
	return done() //if we stick with abandon for retry we don't need to pass and return handlers
}
