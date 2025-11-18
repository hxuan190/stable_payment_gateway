package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
)

var (
	// ErrInvalidAmount is returned when payment amount is invalid
	ErrInvalidAmount = errors.New("invalid payment amount")
	// ErrPaymentExpired is returned when payment has expired
	ErrPaymentExpired = errors.New("payment has expired")
	// ErrPaymentAlreadyCompleted is returned when payment is already completed
	ErrPaymentAlreadyCompleted = errors.New("payment is already completed")
	// ErrPaymentNotFound is returned when payment is not found
	ErrPaymentNotFound = repository.ErrPaymentNotFound
	// ErrInvalidPaymentState is returned when payment is in invalid state for operation
	ErrInvalidPaymentState = errors.New("payment is in invalid state for this operation")
	// ErrAmountMismatch is returned when actual amount doesn't match expected amount
	ErrAmountMismatch = errors.New("payment amount mismatch")
	// ErrInvalidChain is returned when chain is not supported
	ErrInvalidChain = errors.New("invalid or unsupported blockchain chain")
)

const (
	// DefaultPaymentExpiryMinutes is the default expiration time for payments
	DefaultPaymentExpiryMinutes = 30
	// DefaultFeePercentage is the default fee percentage (1%)
	DefaultFeePercentage = 0.01
	// PaymentReferenceLength is the length of the payment reference
	PaymentReferenceLength = 16
)

// PaymentRepository defines the interface for payment data access
type PaymentRepository interface {
	Create(payment *model.Payment) error
	GetByID(id string) (*model.Payment, error)
	GetByTxHash(txHash string) (*model.Payment, error)
	GetByPaymentReference(reference string) (*model.Payment, error)
	UpdateStatus(id string, status model.PaymentStatus) error
	Update(payment *model.Payment) error
	ListByMerchant(merchantID string, limit, offset int) ([]*model.Payment, error)
	GetExpiredPayments() ([]*model.Payment, error)
}

// MerchantRepository defines the interface for merchant data access
type MerchantRepository interface {
	GetByID(id string) (*model.Merchant, error)
}

// ExchangeRateProvider defines the interface for getting exchange rates
type ExchangeRateProvider interface {
	GetUSDTToVND(ctx context.Context) (decimal.Decimal, error)
	GetUSDCToVND(ctx context.Context) (decimal.Decimal, error)
}

// PaymentService handles payment business logic
type PaymentService struct {
	paymentRepo         PaymentRepository
	merchantRepo        MerchantRepository
	exchangeRateService ExchangeRateProvider
	redisClient         *redis.Client // For publishing real-time events
	logger              *logrus.Logger
	defaultChain        model.Chain
	defaultCurrency     string
	walletAddress       string
	feePercentage       decimal.Decimal
	expiryMinutes       int
}

// PaymentServiceConfig contains configuration for PaymentService
type PaymentServiceConfig struct {
	DefaultChain    model.Chain
	DefaultCurrency string
	WalletAddress   string
	FeePercentage   float64
	ExpiryMinutes   int
	RedisClient     *redis.Client // Optional: for real-time events
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	paymentRepo PaymentRepository,
	merchantRepo MerchantRepository,
	exchangeRateService ExchangeRateProvider,
	config PaymentServiceConfig,
	logger *logrus.Logger,
) *PaymentService {
	feePercentage := decimal.NewFromFloat(DefaultFeePercentage)
	if config.FeePercentage > 0 && config.FeePercentage < 1 {
		feePercentage = decimal.NewFromFloat(config.FeePercentage)
	}

	expiryMinutes := DefaultPaymentExpiryMinutes
	if config.ExpiryMinutes > 0 {
		expiryMinutes = config.ExpiryMinutes
	}

	defaultChain := model.ChainSolana
	if config.DefaultChain != "" {
		defaultChain = config.DefaultChain
	}

	defaultCurrency := "USDT"
	if config.DefaultCurrency != "" {
		defaultCurrency = config.DefaultCurrency
	}

	return &PaymentService{
		paymentRepo:         paymentRepo,
		merchantRepo:        merchantRepo,
		exchangeRateService: exchangeRateService,
		redisClient:         config.RedisClient,
		logger:              logger,
		defaultChain:        defaultChain,
		defaultCurrency:     defaultCurrency,
		walletAddress:       config.WalletAddress,
		feePercentage:       feePercentage,
		expiryMinutes:       expiryMinutes,
	}
}

// CreatePaymentRequest contains parameters for creating a payment
type CreatePaymentRequest struct {
	MerchantID  string
	AmountVND   decimal.Decimal
	Currency    string // USDT, USDC, BUSD
	Chain       model.Chain // solana, bsc
	OrderID     string
	Description string
	CallbackURL string
}

// CreatePayment creates a new payment request
func (s *PaymentService) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*model.Payment, error) {
	s.logger.WithFields(logrus.Fields{
		"merchant_id": req.MerchantID,
		"amount_vnd":  req.AmountVND,
		"currency":    req.Currency,
		"chain":       req.Chain,
	}).Info("Creating payment")

	// Validate merchant exists and is approved
	merchant, err := s.merchantRepo.GetByID(req.MerchantID)
	if err != nil {
		if errors.Is(err, repository.ErrMerchantNotFound) {
			return nil, ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	if merchant.KYCStatus != model.KYCStatusApproved {
		s.logger.WithField("merchant_id", req.MerchantID).Warn("Merchant not approved")
		return nil, ErrMerchantNotApproved
	}

	// Validate amount
	if req.AmountVND.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidAmount
	}

	// Set defaults if not provided
	currency := req.Currency
	if currency == "" {
		currency = s.defaultCurrency
	}

	chain := req.Chain
	if chain == "" {
		chain = s.defaultChain
	}

	// Validate chain and currency combination
	if err := s.validateChainCurrency(chain, currency); err != nil {
		return nil, err
	}

	// Get current exchange rate
	exchangeRate, err := s.getExchangeRate(ctx, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	// Calculate crypto amount (VND / exchange rate)
	amountCrypto := req.AmountVND.Div(exchangeRate).Round(6) // Round to 6 decimals for USDT/USDC

	// Generate payment ID and reference
	paymentID := uuid.New().String()
	paymentReference, err := s.generatePaymentReference()
	if err != nil {
		return nil, fmt.Errorf("failed to generate payment reference: %w", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(s.expiryMinutes) * time.Minute)

	// Create payment object
	payment := &model.Payment{
		ID:                paymentID,
		MerchantID:        req.MerchantID,
		AmountVND:         req.AmountVND,
		AmountCrypto:      amountCrypto,
		Currency:          currency,
		Chain:             chain,
		ExchangeRate:      exchangeRate,
		Status:            model.PaymentStatusCreated,
		PaymentReference:  paymentReference,
		DestinationWallet: s.walletAddress,
		ExpiresAt:         expiresAt,
		FeePercentage:     s.feePercentage,
	}

	// Set optional fields
	if req.OrderID != "" {
		payment.OrderID = sql.NullString{String: req.OrderID, Valid: true}
	}
	if req.Description != "" {
		payment.Description = sql.NullString{String: req.Description, Valid: true}
	}
	if req.CallbackURL != "" {
		payment.CallbackURL = sql.NullString{String: req.CallbackURL, Valid: true}
	}

	// Calculate fee and net amount
	payment.CalculateFee()

	// Save to database
	if err := s.paymentRepo.Create(payment); err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"payment_id":       payment.ID,
		"amount_vnd":       payment.AmountVND,
		"amount_crypto":    payment.AmountCrypto,
		"exchange_rate":    payment.ExchangeRate,
		"payment_reference": payment.PaymentReference,
	}).Info("Payment created successfully")

	return payment, nil
}

// GetPaymentStatus retrieves the status of a payment
func (s *PaymentService) GetPaymentStatus(ctx context.Context, paymentID string) (*model.Payment, error) {
	payment, err := s.paymentRepo.GetByID(paymentID)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return payment, nil
}

// GetPaymentByReference retrieves a payment by its reference
func (s *PaymentService) GetPaymentByReference(ctx context.Context, reference string) (*model.Payment, error) {
	payment, err := s.paymentRepo.GetByPaymentReference(reference)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by reference: %w", err)
	}

	return payment, nil
}

// ValidatePayment checks if a payment is valid and can be paid
func (s *PaymentService) ValidatePayment(ctx context.Context, paymentID string) error {
	payment, err := s.paymentRepo.GetByID(paymentID)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			return ErrPaymentNotFound
		}
		return fmt.Errorf("failed to get payment: %w", err)
	}

	// Check if payment has expired
	if payment.IsExpired() {
		return ErrPaymentExpired
	}

	// Check if payment is already completed
	if payment.Status == model.PaymentStatusCompleted {
		return ErrPaymentAlreadyCompleted
	}

	// Check if payment is in a valid state
	if payment.Status != model.PaymentStatusCreated && payment.Status != model.PaymentStatusPending {
		return ErrInvalidPaymentState
	}

	return nil
}

// ConfirmPaymentRequest contains parameters for confirming a payment
type ConfirmPaymentRequest struct {
	PaymentID     string
	TxHash        string
	FromAddress   string
	ActualAmount  decimal.Decimal
	Confirmations int32
}

// ConfirmPayment confirms a payment after blockchain transaction is detected
func (s *PaymentService) ConfirmPayment(ctx context.Context, req ConfirmPaymentRequest) (*model.Payment, error) {
	s.logger.WithFields(logrus.Fields{
		"payment_id":    req.PaymentID,
		"tx_hash":       req.TxHash,
		"actual_amount": req.ActualAmount,
	}).Info("Confirming payment")

	// Get payment
	payment, err := s.paymentRepo.GetByID(req.PaymentID)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Validate payment can be confirmed
	if !payment.CanBeConfirmed() {
		s.logger.WithFields(logrus.Fields{
			"payment_id": req.PaymentID,
			"status":     payment.Status,
			"is_expired": payment.IsExpired(),
		}).Warn("Payment cannot be confirmed")

		if payment.IsExpired() {
			return nil, ErrPaymentExpired
		}
		if payment.Status == model.PaymentStatusCompleted {
			return nil, ErrPaymentAlreadyCompleted
		}
		return nil, ErrInvalidPaymentState
	}

	// Verify amount matches expected amount exactly
	// Allow a small tolerance for rounding errors (0.000001 crypto)
	tolerance := decimal.NewFromFloat(0.000001)
	diff := req.ActualAmount.Sub(payment.AmountCrypto).Abs()
	if diff.GreaterThan(tolerance) {
		s.logger.WithFields(logrus.Fields{
			"payment_id":      req.PaymentID,
			"expected_amount": payment.AmountCrypto,
			"actual_amount":   req.ActualAmount,
			"difference":      diff,
		}).Error("Payment amount mismatch")
		return nil, ErrAmountMismatch
	}

	// Update payment with transaction details
	now := time.Now()
	payment.Status = model.PaymentStatusConfirming
	payment.TxHash = sql.NullString{String: req.TxHash, Valid: true}
	payment.TxConfirmations = sql.NullInt32{Int32: req.Confirmations, Valid: true}
	payment.FromAddress = sql.NullString{String: req.FromAddress, Valid: true}
	payment.PaidAt = sql.NullTime{Time: now, Valid: true}

	// If confirmations meet threshold, mark as completed
	// For Solana with finalized commitment, confirmations should be sufficient
	if req.Confirmations >= 1 {
		payment.Status = model.PaymentStatusCompleted
		payment.ConfirmedAt = sql.NullTime{Time: now, Valid: true}
	}

	// Update payment in database
	if err := s.paymentRepo.Update(payment); err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"payment_id": payment.ID,
		"status":     payment.Status,
		"tx_hash":    req.TxHash,
	}).Info("Payment confirmed successfully")

	// Publish real-time event to Redis for WebSocket clients
	eventType := "payment.confirming"
	eventMessage := "Payment is being confirmed"
	if payment.Status == model.PaymentStatusCompleted {
		eventType = "payment.completed"
		eventMessage = "Payment completed successfully"
	}

	s.publishPaymentEvent(ctx, PaymentEvent{
		Type:      eventType,
		PaymentID: payment.ID,
		Status:    string(payment.Status),
		TxHash:    req.TxHash,
		Timestamp: time.Now(),
		Message:   eventMessage,
	})

	// TODO: In a complete implementation, this should:
	// 1. Create ledger entries (handled by ledger service)
	// 2. Update merchant balance (handled by balance service)
	// 3. Trigger webhook notification (handled by notification service)
	// These will be implemented when those services are available

	return payment, nil
}

// ExpirePayment marks a payment as expired
func (s *PaymentService) ExpirePayment(ctx context.Context, paymentID string) error {
	s.logger.WithField("payment_id", paymentID).Info("Expiring payment")

	// Get payment
	payment, err := s.paymentRepo.GetByID(paymentID)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			return ErrPaymentNotFound
		}
		return fmt.Errorf("failed to get payment: %w", err)
	}

	// Only expire payments that are in created or pending status
	if payment.Status != model.PaymentStatusCreated && payment.Status != model.PaymentStatusPending {
		s.logger.WithFields(logrus.Fields{
			"payment_id": paymentID,
			"status":     payment.Status,
		}).Warn("Payment is not in a state that can be expired")
		return ErrInvalidPaymentState
	}

	// Update status to expired
	if err := s.paymentRepo.UpdateStatus(paymentID, model.PaymentStatusExpired); err != nil {
		return fmt.Errorf("failed to expire payment: %w", err)
	}

	s.logger.WithField("payment_id", paymentID).Info("Payment expired successfully")

	// Publish real-time event to Redis for WebSocket clients
	s.publishPaymentEvent(ctx, PaymentEvent{
		Type:      "payment.expired",
		PaymentID: paymentID,
		Status:    string(model.PaymentStatusExpired),
		Timestamp: time.Now(),
		Message:   "Payment has expired",
	})

	return nil
}

// FailPayment marks a payment as failed
func (s *PaymentService) FailPayment(ctx context.Context, paymentID, reason string) error {
	s.logger.WithFields(logrus.Fields{
		"payment_id": paymentID,
		"reason":     reason,
	}).Info("Failing payment")

	// Get payment
	payment, err := s.paymentRepo.GetByID(paymentID)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			return ErrPaymentNotFound
		}
		return fmt.Errorf("failed to get payment: %w", err)
	}

	// Cannot fail a completed payment
	if payment.Status == model.PaymentStatusCompleted {
		return ErrPaymentAlreadyCompleted
	}

	// Update payment with failure reason and status
	payment.Status = model.PaymentStatusFailed
	payment.FailureReason = sql.NullString{String: reason, Valid: true}

	if err := s.paymentRepo.Update(payment); err != nil {
		return fmt.Errorf("failed to fail payment: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"payment_id": paymentID,
		"reason":     reason,
	}).Info("Payment marked as failed")

	// Publish real-time event to Redis for WebSocket clients
	s.publishPaymentEvent(ctx, PaymentEvent{
		Type:      "payment.failed",
		PaymentID: paymentID,
		Status:    string(model.PaymentStatusFailed),
		Timestamp: time.Now(),
		Message:   reason,
	})

	return nil
}

// ListPaymentsByMerchant retrieves payments for a specific merchant
func (s *PaymentService) ListPaymentsByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*model.Payment, error) {
	payments, err := s.paymentRepo.ListByMerchant(merchantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	return payments, nil
}

// GetExpiredPayments retrieves all expired payments that need to be marked as expired
func (s *PaymentService) GetExpiredPayments(ctx context.Context) ([]*model.Payment, error) {
	payments, err := s.paymentRepo.GetExpiredPayments()
	if err != nil {
		return nil, fmt.Errorf("failed to get expired payments: %w", err)
	}

	return payments, nil
}

// Helper functions

// validateChainCurrency validates if the chain and currency combination is supported
func (s *PaymentService) validateChainCurrency(chain model.Chain, currency string) error {
	validCombinations := map[model.Chain]map[string]bool{
		model.ChainSolana: {
			"USDT": true,
			"USDC": true,
		},
		model.ChainBSC: {
			"USDT": true,
			"BUSD": true,
		},
	}

	if currencies, ok := validCombinations[chain]; ok {
		if currencies[currency] {
			return nil
		}
	}

	return fmt.Errorf("%w: %s on %s", ErrInvalidChain, currency, chain)
}

// getExchangeRate retrieves the exchange rate for a specific currency
func (s *PaymentService) getExchangeRate(ctx context.Context, currency string) (decimal.Decimal, error) {
	switch currency {
	case "USDT":
		return s.exchangeRateService.GetUSDTToVND(ctx)
	case "USDC":
		return s.exchangeRateService.GetUSDCToVND(ctx)
	case "BUSD":
		// BUSD is also USD-pegged, use USDT rate
		return s.exchangeRateService.GetUSDTToVND(ctx)
	default:
		return decimal.Zero, fmt.Errorf("unsupported currency: %s", currency)
	}
}

// generatePaymentReference generates a unique payment reference for use in transaction memos
func (s *PaymentService) generatePaymentReference() (string, error) {
	bytes := make([]byte, PaymentReferenceLength/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// PaymentEvent represents a payment status update event for real-time broadcasting
type PaymentEvent struct {
	Type      string    `json:"type"`      // payment.pending, payment.confirming, payment.completed, payment.expired, payment.failed
	PaymentID string    `json:"payment_id"`
	Status    string    `json:"status"`
	TxHash    string    `json:"tx_hash,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// publishPaymentEvent publishes a payment event to Redis for real-time WebSocket broadcasting
func (s *PaymentService) publishPaymentEvent(ctx context.Context, event PaymentEvent) {
	// Skip if Redis client is not configured
	if s.redisClient == nil {
		return
	}

	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"payment_id": event.PaymentID,
			"event_type": event.Type,
		}).Error("Failed to marshal payment event")
		return
	}

	// Publish to Redis channel
	channelName := fmt.Sprintf("payment_events:%s", event.PaymentID)
	if err := s.redisClient.Publish(ctx, channelName, eventJSON).Err(); err != nil {
		s.logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"payment_id": event.PaymentID,
			"channel":    channelName,
		}).Error("Failed to publish payment event to Redis")
		return
	}

	s.logger.WithFields(logrus.Fields{
		"payment_id": event.PaymentID,
		"event_type": event.Type,
		"channel":    channelName,
	}).Debug("Published payment event to Redis")
}
