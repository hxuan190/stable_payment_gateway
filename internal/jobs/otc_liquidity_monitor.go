package jobs

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	otcDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/otc/domain"
)

const (
	// OTCLiquidityMonitorJobName is the name of the OTC liquidity monitoring job
	OTCLiquidityMonitorJobName = "otc_liquidity_monitor"
	// OTCLiquidityMonitorJobSchedule is the cron schedule (every 15 minutes)
	OTCLiquidityMonitorJobSchedule = "*/15 * * * *"
)

// OTCLiquidityMonitorJob handles monitoring OTC partner VND liquidity pools
type OTCLiquidityMonitorJob struct {
	db            *sql.DB
	alertService  AlertService
	logger        *logrus.Logger
	minBalanceVND decimal.Decimal // Minimum safe balance (default 50M VND)
	criticalVND   decimal.Decimal // Critical threshold (default 10M VND)
}

// AlertService defines the interface for sending alerts
type AlertService interface {
	SendTelegramAlert(ctx context.Context, message string) error
	SendSlackAlert(ctx context.Context, message string) error
	SendEmailAlert(ctx context.Context, to, subject, body string) error
}

// NewOTCLiquidityMonitorJob creates a new OTC liquidity monitoring job
func NewOTCLiquidityMonitorJob(
	db *sql.DB,
	alertService AlertService,
	logger *logrus.Logger,
) *OTCLiquidityMonitorJob {
	return &OTCLiquidityMonitorJob{
		db:            db,
		alertService:  alertService,
		logger:        logger,
		minBalanceVND: decimal.NewFromInt(50000000), // 50M VND
		criticalVND:   decimal.NewFromInt(10000000), // 10M VND
	}
}

// Run executes the OTC liquidity monitoring job
func (j *OTCLiquidityMonitorJob) Run(ctx context.Context) error {
	j.logger.Info("Starting OTC liquidity monitoring job")

	// Get all active OTC partners
	partners, err := j.getActivePartners(ctx)
	if err != nil {
		j.logger.WithError(err).Error("Failed to get active OTC partners")
		return fmt.Errorf("failed to get active OTC partners: %w", err)
	}

	if len(partners) == 0 {
		j.logger.Warn("No active OTC partners found")
		return nil
	}

	j.logger.WithField("count", len(partners)).Info("Checking OTC partner balances")

	alertsSent := 0
	for _, partner := range partners {
		// Update last balance check time
		if err := j.updateLastBalanceCheck(ctx, partner.ID); err != nil {
			j.logger.WithError(err).WithField("partner_id", partner.ID).Warn("Failed to update last balance check")
		}

		// Check if balance is low or critical
		if partner.RequiresImmediateAlert() {
			if err := j.sendAlert(ctx, partner); err != nil {
				j.logger.WithError(err).WithField("partner_id", partner.ID).Error("Failed to send alert")
				continue
			}
			alertsSent++
		}
	}

	j.logger.WithFields(logrus.Fields{
		"total_partners": len(partners),
		"alerts_sent":    alertsSent,
	}).Info("OTC liquidity monitoring job completed")

	return nil
}

// getActivePartners retrieves all active OTC partners
func (j *OTCLiquidityMonitorJob) getActivePartners(ctx context.Context) ([]*otcDomain.OTCPartnerLiquidity, error) {
	query := `
		SELECT id, partner_name, partner_code,
		       current_balance_vnd, available_balance_vnd, reserved_balance_vnd,
		       low_balance_threshold_vnd, critical_threshold_vnd,
		       status, is_balance_low, is_balance_critical,
		       last_alert_sent_at, alert_count_today,
		       contact_email, contact_phone,
		       created_at, updated_at, last_balance_check_at
		FROM otc_partner_liquidity
		WHERE status = 'active'
		ORDER BY current_balance_vnd ASC
	`

	rows, err := j.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query OTC partners: %w", err)
	}
	defer rows.Close()

	partners := make([]*otcDomain.OTCPartnerLiquidity, 0)
	for rows.Next() {
		partner := &otcDomain.OTCPartnerLiquidity{}
		err := rows.Scan(
			&partner.ID,
			&partner.PartnerName,
			&partner.PartnerCode,
			&partner.CurrentBalanceVND,
			&partner.AvailableBalanceVND,
			&partner.ReservedBalanceVND,
			&partner.LowBalanceThresholdVND,
			&partner.CriticalThresholdVND,
			&partner.Status,
			&partner.IsBalanceLow,
			&partner.IsBalanceCritical,
			&partner.LastAlertSentAt,
			&partner.AlertCountToday,
			&partner.ContactEmail,
			&partner.ContactPhone,
			&partner.CreatedAt,
			&partner.UpdatedAt,
			&partner.LastBalanceCheckAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan OTC partner: %w", err)
		}
		partners = append(partners, partner)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating OTC partners: %w", err)
	}

	return partners, nil
}

// sendAlert sends alerts to all configured channels
func (j *OTCLiquidityMonitorJob) sendAlert(ctx context.Context, partner *otcDomain.OTCPartnerLiquidity) error {
	j.logger.WithFields(logrus.Fields{
		"partner_id":   partner.ID,
		"partner_name": partner.PartnerName,
		"balance":      partner.CurrentBalanceVND.String(),
		"is_critical":  partner.IsCriticalBalance(),
	}).Warn("Sending OTC liquidity alert")

	// Determine alert level and type
	alertLevel := "warning"
	alertType := "low_balance"
	if partner.IsCriticalBalance() {
		alertLevel = "critical"
		alertType = "critical_balance"
	}

	// Generate alert message
	message := partner.GetAlertMessage()
	message += fmt.Sprintf("\nAvailable: %s VND | Reserved: %s VND",
		partner.AvailableBalanceVND.String(),
		partner.ReservedBalanceVND.String(),
	)

	// Add contact info if available
	if partner.ContactEmail.Valid {
		message += fmt.Sprintf("\nContact: %s", partner.ContactEmail.String)
	}

	// Send alerts to all channels
	sentToTelegram := false
	sentToSlack := false
	sentToEmail := false

	// Send to Telegram
	if j.alertService != nil {
		if err := j.alertService.SendTelegramAlert(ctx, message); err != nil {
			j.logger.WithError(err).Warn("Failed to send Telegram alert")
		} else {
			sentToTelegram = true
		}

		// Send to Slack
		if err := j.alertService.SendSlackAlert(ctx, message); err != nil {
			j.logger.WithError(err).Warn("Failed to send Slack alert")
		} else {
			sentToSlack = true
		}

		// Send email to contact person
		if partner.ContactEmail.Valid {
			subject := fmt.Sprintf("[URGENT] OTC Liquidity Alert: %s", partner.PartnerName)
			if err := j.alertService.SendEmailAlert(ctx, partner.ContactEmail.String, subject, message); err != nil {
				j.logger.WithError(err).Warn("Failed to send email alert")
			} else {
				sentToEmail = true
			}
		}
	}

	// Log the alert in database
	if err := j.logAlert(ctx, partner, alertType, alertLevel, message, sentToTelegram, sentToSlack, sentToEmail); err != nil {
		j.logger.WithError(err).Error("Failed to log alert")
		return fmt.Errorf("failed to log alert: %w", err)
	}

	// Update last alert sent time and increment alert count
	if err := j.updateAlertMetadata(ctx, partner.ID); err != nil {
		j.logger.WithError(err).Warn("Failed to update alert metadata")
	}

	return nil
}

// logAlert records an alert in the database
func (j *OTCLiquidityMonitorJob) logAlert(
	ctx context.Context,
	partner *otcDomain.OTCPartnerLiquidity,
	alertType, alertLevel, message string,
	sentToTelegram, sentToSlack, sentToEmail bool,
) error {
	query := `
		INSERT INTO otc_liquidity_alerts (
			partner_id, alert_type, alert_level, message,
			current_balance_vnd, threshold_vnd,
			sent_to_telegram, sent_to_slack, sent_to_email,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`

	threshold := partner.LowBalanceThresholdVND
	if partner.IsCriticalBalance() {
		threshold = partner.CriticalThresholdVND
	}

	_, err := j.db.ExecContext(ctx, query,
		partner.ID,
		alertType,
		alertLevel,
		message,
		partner.CurrentBalanceVND,
		threshold,
		sentToTelegram,
		sentToSlack,
		sentToEmail,
	)

	if err != nil {
		return fmt.Errorf("failed to insert alert log: %w", err)
	}

	return nil
}

// updateLastBalanceCheck updates the last balance check timestamp
func (j *OTCLiquidityMonitorJob) updateLastBalanceCheck(ctx context.Context, partnerID string) error {
	query := `
		UPDATE otc_partner_liquidity
		SET last_balance_check_at = NOW()
		WHERE id = $1
	`

	_, err := j.db.ExecContext(ctx, query, partnerID)
	return err
}

// updateAlertMetadata updates last alert sent time and increment alert count
func (j *OTCLiquidityMonitorJob) updateAlertMetadata(ctx context.Context, partnerID string) error {
	query := `
		UPDATE otc_partner_liquidity
		SET last_alert_sent_at = NOW(),
		    alert_count_today = alert_count_today + 1
		WHERE id = $1
	`

	_, err := j.db.ExecContext(ctx, query, partnerID)
	return err
}

// GetName returns the job name
func (j *OTCLiquidityMonitorJob) GetName() string {
	return OTCLiquidityMonitorJobName
}

// GetSchedule returns the cron schedule
func (j *OTCLiquidityMonitorJob) GetSchedule() string {
	return OTCLiquidityMonitorJobSchedule
}
