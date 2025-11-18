package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// InMemoryEventBus implements EventBus using in-memory channels
// Suitable for modular monolith deployment
type InMemoryEventBus struct {
	handlers map[string][]Handler
	mu       sync.RWMutex
	logger   *logrus.Logger
	wg       sync.WaitGroup
	stopped  bool
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus(logger *logrus.Logger) *InMemoryEventBus {
	if logger == nil {
		logger = logrus.New()
	}

	return &InMemoryEventBus{
		handlers: make(map[string][]Handler),
		logger:   logger,
	}
}

// Publish publishes an event to all registered handlers
// Handlers are executed asynchronously in separate goroutines
func (b *InMemoryEventBus) Publish(ctx context.Context, event Event) error {
	b.mu.RLock()
	if b.stopped {
		b.mu.RUnlock()
		return fmt.Errorf("event bus is stopped")
	}

	handlers, exists := b.handlers[event.Name()]
	b.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		b.logger.WithFields(logrus.Fields{
			"event": event.Name(),
		}).Debug("No handlers registered for event")
		return nil
	}

	b.logger.WithFields(logrus.Fields{
		"event":         event.Name(),
		"handler_count": len(handlers),
	}).Info("Publishing event")

	// Execute all handlers asynchronously
	for i, handler := range handlers {
		b.wg.Add(1)
		go func(h Handler, index int) {
			defer b.wg.Done()

			if err := h(ctx, event); err != nil {
				b.logger.WithFields(logrus.Fields{
					"event":   event.Name(),
					"handler": index,
					"error":   err.Error(),
				}).Error("Event handler failed")
			}
		}(handler, i)
	}

	return nil
}

// Subscribe registers a handler for a specific event
func (b *InMemoryEventBus) Subscribe(eventName string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.stopped {
		b.logger.Warn("Cannot subscribe to stopped event bus")
		return
	}

	b.handlers[eventName] = append(b.handlers[eventName], handler)

	b.logger.WithFields(logrus.Fields{
		"event":         eventName,
		"handler_count": len(b.handlers[eventName]),
	}).Info("Subscribed handler to event")
}

// Shutdown gracefully shuts down the event bus
// Waits for all in-flight event handlers to complete
func (b *InMemoryEventBus) Shutdown(ctx context.Context) error {
	b.mu.Lock()
	b.stopped = true
	b.mu.Unlock()

	b.logger.Info("Shutting down event bus, waiting for handlers to complete")

	// Wait for all handlers with context timeout
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		b.logger.Info("Event bus shutdown complete")
		return nil
	case <-ctx.Done():
		b.logger.Warn("Event bus shutdown timed out")
		return ctx.Err()
	}
}

// HandlerCount returns the number of handlers registered for an event
func (b *InMemoryEventBus) HandlerCount(eventName string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[eventName])
}
