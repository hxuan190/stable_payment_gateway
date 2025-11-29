package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/repository"
	notificationService "github.com/hxuan190/stable_payment_gateway/internal/modules/notification/service"
	paymentDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/sirupsen/logrus"
)

// PaymentRepositoryInterface defines the interface for payment operations needed by compliance
type PaymentRepositoryInterface interface {
	GetByID(id string) (*paymentDomain.Payment, error)
	Update(payment *paymentDomain.Payment) error
	UpdateStatus(id string, status paymentDomain.PaymentStatus) error
}

// ComplianceAlertService handles business logic for compliance alerts
type ComplianceAlertService struct {
	repo                *repository.ComplianceAlertRepository
	paymentRepo         PaymentRepositoryInterface
	notificationService *notificationService.NotificationService
	logger              *logrus.Logger
	opsTeamEmails       []string // Email addresses for compliance team
}

// ComplianceAlertServiceConfig holds configuration for compliance alert service
type ComplianceAlertServiceConfig struct {
	ComplianceAlertRepo *repository.ComplianceAlertRepository
	PaymentRepo         PaymentRepositoryInterface
	NotificationService *notificationService.NotificationService
	Logger              *logrus.Logger
	OpsTeamEmails       []string
}

// NewComplianceAlertService creates a new compliance alert service
func NewComplianceAlertService(config ComplianceAlertServiceConfig) *ComplianceAlertService {
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	if config.OpsTeamEmails == nil {
		config.OpsTeamEmails = []string{}
	}

	return &ComplianceAlertService{
		repo:                config.ComplianceAlertRepo,
		paymentRepo:         config.PaymentRepo,
		notificationService: config.NotificationService,
		logger:              config.Logger,
		opsTeamEmails:       config.OpsTeamEmails,
	}
}

// CreateSanctionedAddressAlert creates an alert for a sanctioned address detection
func (s *ComplianceAlertService) CreateSanctionedAddressAlert(
	ctx context.Context,
	paymentID uuid.UUID,
	fromAddress string,
	blockchain string,
	riskScore int,
	riskFlags []string,
	evidence map[string]interface{},
) (*domain.ComplianceAlert, error) {
	// Get payment details
	payment, err := s.paymentRepo.GetByID(paymentID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Create the alert
	alert := domain.NewComplianceAlert(
		domain.AlertTypeSanctionedAddress,
		domain.SeverityCritical,
		fmt.Sprintf("Sanctioned address detected: %s on %s blockchain. Payment ID: %s", fromAddress, blockchain, paymentID),
	)

	// Set alert details
	alert.PaymentID = &paymentID
	merchantID, err := uuid.Parse(payment.MerchantID)
	if err == nil {
		alert.MerchantID = &merchantID
	}
	alert.FromAddress = &fromAddress
	alert.Blockchain = &blockchain
	alert.RiskScore = &riskScore
	alert.RiskFlags = riskFlags
	alert.TransactionHash = &payment.TxHash.String

	// Add evidence
	if evidence != nil {
		alert.Evidence = evidence
	}

	// Set recommended action
	recommendedAction := "IMMEDIATE: Freeze payment, contact compliance team, do not process payout"
	alert.RecommendedAction = &recommendedAction

	// Save the alert
	if err := s.repo.Create(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create compliance alert: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"alert_id":     alert.ID,
		"payment_id":   paymentID,
		"from_address": fromAddress,
		"blockchain":   blockchain,
		"risk_score":   riskScore,
	}).Warn("Sanctioned address alert created")

	// Flag the payment for review
	if err := s.FlagPaymentForReview(ctx, alert, paymentID); err != nil {
		s.logger.WithError(err).Error("Failed to flag payment for review")
		// Don't fail the alert creation if payment flagging fails
	}

	// Send alert to compliance team
	if err := s.SendAlertNotification(ctx, alert); err != nil {
		s.logger.WithError(err).Error("Failed to send alert notification")
		// Don't fail the alert creation if notification fails
	}

	return alert, nil
}

// CreateHighRiskAlert creates an alert for high-risk transactions
func (s *ComplianceAlertService) CreateHighRiskAlert(
	ctx context.Context,
	paymentID uuid.UUID,
	fromAddress string,
	blockchain string,
	riskScore int,
	riskFlags []string,
	evidence map[string]interface{},
) (*domain.ComplianceAlert, error) {
	// Get payment details
	payment, err := s.paymentRepo.GetByID(paymentID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Determine severity based on risk score
	severity := domain.SeverityMedium
	if riskScore >= 80 {
		severity = domain.SeverityCritical
	} else if riskScore >= 60 {
		severity = domain.SeverityHigh
	}

	// Create the alert
	alert := domain.NewComplianceAlert(
		domain.AlertTypeHighRiskAddress,
		severity,
		fmt.Sprintf("High-risk address detected: %s on %s blockchain. Risk score: %d. Payment ID: %s", fromAddress, blockchain, riskScore, paymentID),
	)

	// Set alert details
	alert.PaymentID = &paymentID
	merchantID, err := uuid.Parse(payment.MerchantID)
	if err == nil {
		alert.MerchantID = &merchantID
	}
	alert.FromAddress = &fromAddress
	alert.Blockchain = &blockchain
	alert.RiskScore = &riskScore
	alert.RiskFlags = riskFlags
	alert.TransactionHash = &payment.TxHash.String

	// Add evidence
	if evidence != nil {
		alert.Evidence = evidence
	}

	// Set recommended action based on severity
	var recommendedAction string
	if severity == domain.SeverityCritical {
		recommendedAction = "URGENT: Review immediately, may need to freeze payment"
	} else if severity == domain.SeverityHigh {
		recommendedAction = "Review within 24 hours, verify merchant and transaction legitimacy"
	} else {
		recommendedAction = "Review when possible, monitor for patterns"
	}
	alert.RecommendedAction = &recommendedAction

	// Save the alert
	if err := s.repo.Create(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create compliance alert: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"alert_id":     alert.ID,
		"payment_id":   paymentID,
		"from_address": fromAddress,
		"blockchain":   blockchain,
		"risk_score":   riskScore,
		"severity":     severity,
	}).Warn("High-risk address alert created")

	// For critical severity, flag the payment
	if severity == domain.SeverityCritical {
		if err := s.FlagPaymentForReview(ctx, alert, paymentID); err != nil {
			s.logger.WithError(err).Error("Failed to flag payment for review")
		}
	}

	// Send alert to compliance team
	if err := s.SendAlertNotification(ctx, alert); err != nil {
		s.logger.WithError(err).Error("Failed to send alert notification")
	}

	return alert, nil
}

// FlagPaymentForReview flags a payment for manual review and prevents automatic processing
func (s *ComplianceAlertService) FlagPaymentForReview(ctx context.Context, alert *domain.ComplianceAlert, paymentID uuid.UUID) error {
	// Get the payment
	payment, err := s.paymentRepo.GetByID(paymentID.String())
	if err != nil {
		return fmt.Errorf("failed to get payment: %w", err)
	}

	// Update payment status to require manual review
	// We'll add a "flagged" status or use the metadata field
	if payment.Metadata == nil {
		payment.Metadata = make(database.JSONBMap)
	}
	payment.Metadata["compliance_flagged"] = true
	payment.Metadata["compliance_alert_id"] = alert.ID.String()
	payment.Metadata["compliance_reason"] = alert.Details

	// Update the payment
	if err := s.paymentRepo.Update(payment); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Record action in alert
	alert.AddAction(domain.ActionPaymentFlagged)
	if err := s.repo.Update(ctx, alert); err != nil {
		s.logger.WithError(err).Error("Failed to update alert with action")
	}

	s.logger.WithFields(logrus.Fields{
		"payment_id": paymentID,
		"alert_id":   alert.ID,
	}).Info("Payment flagged for manual review")

	return nil
}

// SendAlertNotification sends email notification to compliance team
func (s *ComplianceAlertService) SendAlertNotification(ctx context.Context, alert *domain.ComplianceAlert) error {
	if s.notificationService == nil {
		s.logger.Warn("Notification service not configured, skipping alert notification")
		return nil
	}

	if len(s.opsTeamEmails) == 0 {
		s.logger.Warn("No ops team emails configured, skipping alert notification")
		return nil
	}

	// Prepare email data
	emailData := map[string]interface{}{
		"alert_id":           alert.ID.String(),
		"alert_type":         alert.AlertType,
		"severity":           alert.Severity,
		"details":            alert.Details,
		"payment_id":         "",
		"from_address":       "",
		"blockchain":         "",
		"risk_score":         "",
		"recommended_action": "",
	}

	if alert.PaymentID != nil {
		emailData["payment_id"] = alert.PaymentID.String()
	}
	if alert.FromAddress != nil {
		emailData["from_address"] = *alert.FromAddress
	}
	if alert.Blockchain != nil {
		emailData["blockchain"] = *alert.Blockchain
	}
	if alert.RiskScore != nil {
		emailData["risk_score"] = fmt.Sprintf("%d", *alert.RiskScore)
	}
	if alert.RecommendedAction != nil {
		emailData["recommended_action"] = *alert.RecommendedAction
	}

	// Send to all ops team members
	merchantID := ""
	if alert.MerchantID != nil {
		merchantID = alert.MerchantID.String()
	}

	requiredAction := ""
	if alert.RecommendedAction != nil {
		requiredAction = *alert.RecommendedAction
	}

	if err := s.notificationService.SendComplianceAlertEmail(
		ctx,
		s.opsTeamEmails,
		string(alert.AlertType),
		merchantID,
		alert.Details,
		requiredAction,
	); err != nil {
		s.logger.WithFields(logrus.Fields{
			"alert_id": alert.ID,
		}).WithError(err).Error("Failed to send compliance alert email")
		return err
	}

	// Mark email as sent
	alert.MarkEmailSent(s.opsTeamEmails)
	if err := s.repo.Update(ctx, alert); err != nil {
		s.logger.WithError(err).Error("Failed to update alert after sending email")
	}

	s.logger.WithFields(logrus.Fields{
		"alert_id":   alert.ID,
		"recipients": len(s.opsTeamEmails),
	}).Info("Compliance alert notifications sent")

	return nil
}

// GetAlert retrieves a compliance alert by ID
func (s *ComplianceAlertService) GetAlert(ctx context.Context, id uuid.UUID) (*domain.ComplianceAlert, error) {
	return s.repo.GetByID(ctx, id)
}

// GetAlertsByPayment retrieves all alerts for a payment
func (s *ComplianceAlertService) GetAlertsByPayment(ctx context.Context, paymentID uuid.UUID) ([]*domain.ComplianceAlert, error) {
	return s.repo.GetByPaymentID(ctx, paymentID)
}

// GetOpenAlerts retrieves all open compliance alerts
func (s *ComplianceAlertService) GetOpenAlerts(ctx context.Context, limit, offset int) ([]*domain.ComplianceAlert, error) {
	return s.repo.GetOpenAlerts(ctx, limit, offset)
}

// ResolveAlert resolves a compliance alert
func (s *ComplianceAlertService) ResolveAlert(ctx context.Context, alertID uuid.UUID, reviewedBy, notes string) error {
	alert, err := s.repo.GetByID(ctx, alertID)
	if err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}

	alert.Resolve(reviewedBy, notes)

	if err := s.repo.Update(ctx, alert); err != nil {
		return fmt.Errorf("failed to update alert: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"alert_id":    alertID,
		"reviewed_by": reviewedBy,
	}).Info("Compliance alert resolved")

	return nil
}

// MarkFalsePositive marks an alert as a false positive
func (s *ComplianceAlertService) MarkFalsePositive(ctx context.Context, alertID uuid.UUID, reviewedBy, notes string) error {
	alert, err := s.repo.GetByID(ctx, alertID)
	if err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}

	alert.MarkFalsePositive(reviewedBy, notes)

	if err := s.repo.Update(ctx, alert); err != nil {
		return fmt.Errorf("failed to update alert: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"alert_id":    alertID,
		"reviewed_by": reviewedBy,
	}).Info("Compliance alert marked as false positive")

	return nil
}
