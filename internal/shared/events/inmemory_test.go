package events

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type TestEvent struct {
	BaseEvent
	Data string
}

func NewTestEvent(data string) *TestEvent {
	return &TestEvent{
		BaseEvent: NewBaseEvent("test.event"),
		Data:      data,
	}
}

func TestInMemoryEventBus_PublishAndSubscribe(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	bus := NewInMemoryEventBus(logger)

	var receivedData string
	var handlerCalled int32

	// Subscribe handler
	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		atomic.AddInt32(&handlerCalled, 1)
		testEvent := event.(*TestEvent)
		receivedData = testEvent.Data
		return nil
	})

	// Publish event
	event := NewTestEvent("hello world")
	err := bus.Publish(context.Background(), event)
	assert.NoError(t, err)

	// Wait for async handler
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&handlerCalled))
	assert.Equal(t, "hello world", receivedData)
}

func TestInMemoryEventBus_MultipleHandlers(t *testing.T) {
	logger := logrus.New()
	bus := NewInMemoryEventBus(logger)

	var callCount int32

	// Subscribe multiple handlers
	for i := 0; i < 3; i++ {
		bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
			atomic.AddInt32(&callCount, 1)
			return nil
		})
	}

	// Publish event
	err := bus.Publish(context.Background(), NewTestEvent("test"))
	assert.NoError(t, err)

	// Wait for async handlers
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

func TestInMemoryEventBus_NoHandlers(t *testing.T) {
	logger := logrus.New()
	bus := NewInMemoryEventBus(logger)

	// Publish event with no handlers - should not error
	err := bus.Publish(context.Background(), NewTestEvent("test"))
	assert.NoError(t, err)
}

func TestInMemoryEventBus_Shutdown(t *testing.T) {
	logger := logrus.New()
	bus := NewInMemoryEventBus(logger)

	var handlerCompleted atomic.Bool

	// Subscribe handler that takes time
	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		time.Sleep(100 * time.Millisecond)
		handlerCompleted.Store(true)
		return nil
	})

	// Publish event
	err := bus.Publish(context.Background(), NewTestEvent("test"))
	assert.NoError(t, err)

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = bus.Shutdown(ctx)
	assert.NoError(t, err)
	assert.True(t, handlerCompleted.Load())
}

func TestInMemoryEventBus_ShutdownTimeout(t *testing.T) {
	logger := logrus.New()
	bus := NewInMemoryEventBus(logger)

	// Subscribe handler that takes longer than shutdown timeout
	bus.Subscribe("test.event", func(ctx context.Context, event Event) error {
		time.Sleep(300 * time.Millisecond)
		return nil
	})

	// Publish event
	err := bus.Publish(context.Background(), NewTestEvent("test"))
	assert.NoError(t, err)

	// Shutdown with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = bus.Shutdown(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestInMemoryEventBus_PublishAfterShutdown(t *testing.T) {
	logger := logrus.New()
	bus := NewInMemoryEventBus(logger)

	// Shutdown bus
	err := bus.Shutdown(context.Background())
	assert.NoError(t, err)

	// Try to publish after shutdown
	err = bus.Publish(context.Background(), NewTestEvent("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stopped")
}
