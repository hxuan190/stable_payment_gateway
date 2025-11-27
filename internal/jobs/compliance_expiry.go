package jobs

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"

	paymentdomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
)

const (
	// ComplianceExpiryJobName is the name of the compliance expiry job
	ComplianceExpiryJobName = "expire_compliance_pending_payments"
	// ComplianceExpiryJobSchedule is the cron schedule (every 5 minutes)
	ComplianceExpiryJobSchedule = "*/5 * * * *"
)

// ComplianceExpiryJob handles expiring payments stuck in pending_compliance status
type ComplianceExpiryJob struct {
	paymentRepo paymentdomain.PaymentRepository
	logger      *logrus.Logger
}

// NewComplianceExpiryJob creates a new compliance expiry job
func NewComplianceExpiryJob(
	paymentRepo paymentdomain.PaymentRepository,
	logger *logrus.Logger,
) *ComplianceExpiryJob {
	return &ComplianceExpiryJob{
		paymentRepo: paymentRepo,
		logger:      logger,
	}
}

// Run executes the compliance expiry job
func (j *ComplianceExpiryJob) Run(ctx context.Context) error {
	j.logger.Info("Starting compliance expiry job")

	// Get payments that are pending_compliance and past their deadline
	expiredPayments, err := j.paymentRepo.GetComplianceExpiredPayments()
	if err != nil {
		j.logger.WithError(err).Error("Failed to get compliance expired payments")
		return fmt.Errorf("failed to get compliance expired payments: %w", err)
	}

	if len(expiredPayments) == 0 {
		j.logger.Info("No compliance expired payments found")
		return nil
	}

	j.logger.WithField("count", len(expiredPayments)).Info("Found compliance expired payments")

	successCount := 0
	errorCount := 0

	for _, payment := range expiredPayments {
		err := j.expirePayment(ctx, payment)
		if err != nil {
			j.logger.WithFields(logrus.Fields{
				"payment_id": payment.ID,
				"error":      err.Error(),
			}).Error("Failed to expire payment")
			errorCount++
			continue
		}
		successCount++
	}

	j.logger.WithFields(logrus.Fields{
		"total":   len(expiredPayments),
		"success": successCount,
		"errors":  errorCount,
	}).Info("Compliance expiry job completed")

	if errorCount > 0 {
		return fmt.Errorf("compliance expiry job completed with %d errors", errorCount)
	}

	return nil
}

// expirePayment marks a single payment as failed due to compliance timeout
func (j *ComplianceExpiryJob) expirePayment(ctx context.Context, payment *paymentdomain.Payment) error {
	j.logger.WithFields(logrus.Fields{
		"payment_id":          payment.ID,
		"merchant_id":         payment.MerchantID,
		"amount_vnd":          payment.AmountVND.String(),
		"compliance_deadline": payment.ComplianceDeadline.Time,
	}).Info("Expiring payment due to compliance timeout")

	// Update payment status to failed with reason
	payment.Status = paymentdomain.PaymentStatusFailed
	payment.FailureReason = sql.NullString{
		Valid:  true,
		String: "Compliance data (FATF Travel Rule) not submitted within 24 hours deadline",
	}

	// Update payment in database
	if err := j.paymentRepo.Update(payment); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// TODO: Send notification to merchant about failed payment
	// TODO: Trigger webhook notification
	// TODO: Log audit event

	j.logger.WithField("payment_id", payment.ID).Info("Payment expired successfully due to compliance timeout")

	return nil
}

// GetName returns the job name
func (j *ComplianceExpiryJob) GetName() string {
	return ComplianceExpiryJobName
}

// GetSchedule returns the cron schedule
func (j *ComplianceExpiryJob) GetSchedule() string {
	return ComplianceExpiryJobSchedule
}
