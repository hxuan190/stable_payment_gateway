package blockchain

import (
	"context"

	"github.com/hxuan190/stable_payment_gateway/internal/ports"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
)

// EventBasedListenerAdapter wraps a BlockchainListener and publishes events instead of using callbacks
// This implements the event-driven architecture pattern for decoupling
type EventBasedListenerAdapter struct {
	// Underlying blockchain listener
	listener ports.BlockchainListener

	// Event bus for publishing events
	eventBus events.Publisher
}

// NewEventBasedListenerAdapter creates a new event-based listener adapter
func NewEventBasedListenerAdapter(listener ports.BlockchainListener, eventBus events.Publisher) *EventBasedListenerAdapter {
	adapter := &EventBasedListenerAdapter{
		listener: listener,
		eventBus: eventBus,
	}

	// Set the confirmation handler to publish events
	listener.SetConfirmationHandler(adapter.handlePaymentConfirmation)

	return adapter
}

// handlePaymentConfirmation is called when a payment is confirmed
// It converts the confirmation into an event and publishes it to the event bus
func (a *EventBasedListenerAdapter) handlePaymentConfirmation(ctx context.Context, confirmation ports.PaymentConfirmation) error {
	// Convert blockchain type to event blockchain type
	var blockchain events.BlockchainType
	switch confirmation.BlockchainType {
	case ports.BlockchainTypeSolana:
		blockchain = events.BlockchainTypeSolana
	case ports.BlockchainTypeBSC:
		blockchain = events.BlockchainTypeBSC
	case ports.BlockchainTypeTRON:
		blockchain = events.BlockchainTypeTRON
	default:
		blockchain = events.BlockchainTypeSolana
	}

	// Create payment confirmed event
	event := events.NewPaymentConfirmedEvent(
		confirmation.PaymentID,
		confirmation.TxHash,
		confirmation.Amount,
		confirmation.TokenSymbol,
		blockchain,
		confirmation.Sender,
		confirmation.Recipient,
		confirmation.BlockNumber,
		confirmation.Timestamp,
	)

	// Publish event to the event bus
	if err := a.eventBus.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

// Delegate all BlockchainListener interface methods to the underlying listener

func (a *EventBasedListenerAdapter) Start(ctx context.Context) error {
	return a.listener.Start(ctx)
}

func (a *EventBasedListenerAdapter) Stop(ctx context.Context) error {
	return a.listener.Stop(ctx)
}

func (a *EventBasedListenerAdapter) IsRunning() bool {
	return a.listener.IsRunning()
}

func (a *EventBasedListenerAdapter) GetBlockchainType() ports.BlockchainType {
	return a.listener.GetBlockchainType()
}

func (a *EventBasedListenerAdapter) GetWalletAddress() string {
	return a.listener.GetWalletAddress()
}

func (a *EventBasedListenerAdapter) SetConfirmationHandler(handler ports.PaymentConfirmationHandler) {
	// Override to prevent direct handler setting
	// The adapter always uses its own handler that publishes events
}

func (a *EventBasedListenerAdapter) GetSupportedTokens() []string {
	return a.listener.GetSupportedTokens()
}

func (a *EventBasedListenerAdapter) GetListenerHealth() ports.ListenerHealth {
	return a.listener.GetListenerHealth()
}
