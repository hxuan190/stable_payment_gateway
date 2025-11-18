package events

import (
	"context"
	"time"
)

// Event represents a domain event that can be published and consumed
type Event interface {
	// Name returns the event name (e.g., "payment.confirmed")
	Name() string
	// OccurredAt returns when the event occurred
	OccurredAt() time.Time
}

// BaseEvent provides common event fields
type BaseEvent struct {
	EventName string    `json:"event_name"`
	Timestamp time.Time `json:"timestamp"`
}

func (e BaseEvent) Name() string {
	return e.EventName
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// NewBaseEvent creates a new base event with current timestamp
func NewBaseEvent(name string) BaseEvent {
	return BaseEvent{
		EventName: name,
		Timestamp: time.Now(),
	}
}

// Handler is a function that handles an event
type Handler func(ctx context.Context, event Event) error

// Publisher can publish events
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// Subscriber can subscribe to events
type Subscriber interface {
	Subscribe(eventName string, handler Handler)
}

// EventBus combines publishing and subscribing
type EventBus interface {
	Publisher
	Subscriber
	// Shutdown gracefully shuts down the event bus
	Shutdown(ctx context.Context) error
}
