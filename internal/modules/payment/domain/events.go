package domain

import (
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/shopspring/decimal"
)

// PaymentCreatedEvent is published when a new payment is created
type PaymentCreatedEvent struct {
	events.BaseEvent
	PaymentID    string          `json:"payment_id"`
	MerchantID   string          `json:"merchant_id"`
	AmountVND    decimal.Decimal `json:"amount_vnd"`
	AmountCrypto decimal.Decimal `json:"amount_crypto"`
	Chain        string          `json:"chain"`
	Token        string          `json:"token"`
	ExchangeRate decimal.Decimal `json:"exchange_rate"`
}

// NewPaymentCreatedEvent creates a new payment created event
func NewPaymentCreatedEvent(paymentID, merchantID string, amountVND, amountCrypto, exchangeRate decimal.Decimal, chain, token string) *PaymentCreatedEvent {
	return &PaymentCreatedEvent{
		BaseEvent:    events.NewBaseEvent("payment.created"),
		PaymentID:    paymentID,
		MerchantID:   merchantID,
		AmountVND:    amountVND,
		AmountCrypto: amountCrypto,
		Chain:        chain,
		Token:        token,
		ExchangeRate: exchangeRate,
	}
}

// PaymentConfirmedEvent is published when a payment is confirmed
type PaymentConfirmedEvent struct {
	events.BaseEvent
	PaymentID    string          `json:"payment_id"`
	MerchantID   string          `json:"merchant_id"`
	AmountVND    decimal.Decimal `json:"amount_vnd"`
	AmountCrypto decimal.Decimal `json:"amount_crypto"`
	Chain        string          `json:"chain"`
	Token        string          `json:"token"`
	TxHash       string          `json:"tx_hash"`
	ConfirmedAt  time.Time       `json:"confirmed_at"`
}

// NewPaymentConfirmedEvent creates a new payment confirmed event
func NewPaymentConfirmedEvent(paymentID, merchantID string, amountVND, amountCrypto decimal.Decimal, chain, token, txHash string, confirmedAt time.Time) *PaymentConfirmedEvent {
	return &PaymentConfirmedEvent{
		BaseEvent:    events.NewBaseEvent("payment.confirmed"),
		PaymentID:    paymentID,
		MerchantID:   merchantID,
		AmountVND:    amountVND,
		AmountCrypto: amountCrypto,
		Chain:        chain,
		Token:        token,
		TxHash:       txHash,
		ConfirmedAt:  confirmedAt,
	}
}

// PaymentExpiredEvent is published when a payment expires
type PaymentExpiredEvent struct {
	events.BaseEvent
	PaymentID  string    `json:"payment_id"`
	MerchantID string    `json:"merchant_id"`
	ExpiredAt  time.Time `json:"expired_at"`
}

// NewPaymentExpiredEvent creates a new payment expired event
func NewPaymentExpiredEvent(paymentID, merchantID string, expiredAt time.Time) *PaymentExpiredEvent {
	return &PaymentExpiredEvent{
		BaseEvent:  events.NewBaseEvent("payment.expired"),
		PaymentID:  paymentID,
		MerchantID: merchantID,
		ExpiredAt:  expiredAt,
	}
}

// PaymentFailedEvent is published when a payment fails
type PaymentFailedEvent struct {
	events.BaseEvent
	PaymentID  string    `json:"payment_id"`
	MerchantID string    `json:"merchant_id"`
	Reason     string    `json:"reason"`
	FailedAt   time.Time `json:"failed_at"`
}

// NewPaymentFailedEvent creates a new payment failed event
func NewPaymentFailedEvent(paymentID, merchantID, reason string, failedAt time.Time) *PaymentFailedEvent {
	return &PaymentFailedEvent{
		BaseEvent:  events.NewBaseEvent("payment.failed"),
		PaymentID:  paymentID,
		MerchantID: merchantID,
		Reason:     reason,
		FailedAt:   failedAt,
	}
}
