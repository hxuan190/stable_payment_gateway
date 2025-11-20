package email

import (
	"fmt"
	"strings"
	"time"
)

// EmailTemplate represents an email template
type EmailTemplate struct {
	Subject     string
	HTMLBody    string
	TextBody    string
	RequireData []string // Required template data fields
}

// TemplateType represents the type of email template
type TemplateType string

const (
	TemplatePaymentConfirmation TemplateType = "payment_confirmation"
	TemplatePayoutApproved      TemplateType = "payout_approved"
	TemplatePayoutCompleted     TemplateType = "payout_completed"
	TemplateDailySettlement     TemplateType = "daily_settlement"
	TemplateKYCApproved         TemplateType = "kyc_approved"
	TemplateKYCRejected         TemplateType = "kyc_rejected"
	TemplateTreasuryAlert       TemplateType = "treasury_alert"
	TemplateComplianceAlert     TemplateType = "compliance_alert"
)

// GetTemplate returns an email template by type
func GetTemplate(templateType TemplateType) *EmailTemplate {
	switch templateType {
	case TemplatePaymentConfirmation:
		return &EmailTemplate{
			Subject:     "Payment Confirmed - {{payment_id}}",
			HTMLBody:    paymentConfirmationHTML,
			TextBody:    paymentConfirmationText,
			RequireData: []string{"payment_id", "amount", "currency", "timestamp"},
		}
	case TemplatePayoutApproved:
		return &EmailTemplate{
			Subject:     "Payout Approved - {{payout_id}}",
			HTMLBody:    payoutApprovedHTML,
			TextBody:    payoutApprovedText,
			RequireData: []string{"payout_id", "amount", "timestamp"},
		}
	case TemplatePayoutCompleted:
		return &EmailTemplate{
			Subject:     "Payout Completed - {{payout_id}}",
			HTMLBody:    payoutCompletedHTML,
			TextBody:    payoutCompletedText,
			RequireData: []string{"payout_id", "amount", "reference_number", "timestamp"},
		}
	case TemplateDailySettlement:
		return &EmailTemplate{
			Subject:     "Daily Settlement Report - {{date}}",
			HTMLBody:    dailySettlementHTML,
			TextBody:    dailySettlementText,
			RequireData: []string{"date", "total_payments", "total_payouts", "total_volume"},
		}
	case TemplateKYCApproved:
		return &EmailTemplate{
			Subject:     "KYC Verification Approved - Welcome!",
			HTMLBody:    kycApprovedHTML,
			TextBody:    kycApprovedText,
			RequireData: []string{"merchant_id", "api_key"},
		}
	case TemplateKYCRejected:
		return &EmailTemplate{
			Subject:     "KYC Verification Update",
			HTMLBody:    kycRejectedHTML,
			TextBody:    kycRejectedText,
			RequireData: []string{"merchant_id", "reason"},
		}
	case TemplateTreasuryAlert:
		return &EmailTemplate{
			Subject:     "Treasury Alert - {{alert_type}}",
			HTMLBody:    treasuryAlertHTML,
			TextBody:    treasuryAlertText,
			RequireData: []string{"alert_type", "wallet_address", "balance", "threshold"},
		}
	case TemplateComplianceAlert:
		return &EmailTemplate{
			Subject:     "Compliance Alert - Action Required",
			HTMLBody:    complianceAlertHTML,
			TextBody:    complianceAlertText,
			RequireData: []string{"alert_type", "merchant_id", "details"},
		}
	default:
		return nil
	}
}

// RenderTemplate renders a template with provided data
func RenderTemplate(template *EmailTemplate, data map[string]interface{}) (*Email, error) {
	// Validate required data
	for _, field := range template.RequireData {
		if _, ok := data[field]; !ok {
			return nil, fmt.Errorf("missing required template data: %s", field)
		}
	}

	// Render subject
	subject := renderString(template.Subject, data)

	// Render HTML body
	htmlBody := renderString(template.HTMLBody, data)

	// Render text body
	textBody := renderString(template.TextBody, data)

	return &Email{
		Subject:      subject,
		HTMLBody:     htmlBody,
		TextBody:     textBody,
		TemplateData: data,
	}, nil
}

// renderString replaces {{key}} placeholders with values from data
func renderString(template string, data map[string]interface{}) string {
	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// FormatTimestamp formats a timestamp for email display
func FormatTimestamp(t time.Time) string {
	return t.Format("January 2, 2006 at 3:04 PM MST")
}

// FormatAmount formats an amount for email display
func FormatAmount(amount, currency string) string {
	return fmt.Sprintf("%s %s", amount, currency)
}

// Template HTML/Text content below...

const paymentConfirmationHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Payment Confirmed</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #10b981;">✓ Payment Confirmed</h2>
        <p>Great news! Your payment has been successfully confirmed and processed.</p>

        <div style="background: #f3f4f6; padding: 15px; border-radius: 5px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Payment ID:</strong> {{payment_id}}</p>
            <p style="margin: 5px 0;"><strong>Amount:</strong> {{amount}} {{currency}}</p>
            <p style="margin: 5px 0;"><strong>Time:</strong> {{timestamp}}</p>
        </div>

        <p>The funds will be available in your balance shortly.</p>

        <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">
            This is an automated notification from Stablecoin Payment Gateway.<br>
            If you have any questions, please contact support.
        </p>
    </div>
</body>
</html>
`

const paymentConfirmationText = `
Payment Confirmed
==================

Great news! Your payment has been successfully confirmed and processed.

Payment Details:
- Payment ID: {{payment_id}}
- Amount: {{amount}} {{currency}}
- Time: {{timestamp}}

The funds will be available in your balance shortly.

---
This is an automated notification from Stablecoin Payment Gateway.
If you have any questions, please contact support.
`

const payoutApprovedHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Payout Approved</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #3b82f6;">✓ Payout Approved</h2>
        <p>Your payout request has been approved and is being processed.</p>

        <div style="background: #f3f4f6; padding: 15px; border-radius: 5px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Payout ID:</strong> {{payout_id}}</p>
            <p style="margin: 5px 0;"><strong>Amount:</strong> {{amount}} VND</p>
            <p style="margin: 5px 0;"><strong>Approved:</strong> {{timestamp}}</p>
        </div>

        <p>You will receive another notification once the bank transfer is completed.</p>

        <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">
            This is an automated notification from Stablecoin Payment Gateway.
        </p>
    </div>
</body>
</html>
`

const payoutApprovedText = `
Payout Approved
===============

Your payout request has been approved and is being processed.

Payout Details:
- Payout ID: {{payout_id}}
- Amount: {{amount}} VND
- Approved: {{timestamp}}

You will receive another notification once the bank transfer is completed.

---
This is an automated notification from Stablecoin Payment Gateway.
`

const payoutCompletedHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Payout Completed</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #10b981;">✓ Payout Completed</h2>
        <p>Your payout has been successfully transferred to your bank account.</p>

        <div style="background: #f3f4f6; padding: 15px; border-radius: 5px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Payout ID:</strong> {{payout_id}}</p>
            <p style="margin: 5px 0;"><strong>Amount:</strong> {{amount}} VND</p>
            <p style="margin: 5px 0;"><strong>Reference:</strong> {{reference_number}}</p>
            <p style="margin: 5px 0;"><strong>Completed:</strong> {{timestamp}}</p>
        </div>

        <p>Please allow 1-2 business days for the funds to appear in your account.</p>

        <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">
            This is an automated notification from Stablecoin Payment Gateway.
        </p>
    </div>
</body>
</html>
`

const payoutCompletedText = `
Payout Completed
================

Your payout has been successfully transferred to your bank account.

Payout Details:
- Payout ID: {{payout_id}}
- Amount: {{amount}} VND
- Reference: {{reference_number}}
- Completed: {{timestamp}}

Please allow 1-2 business days for the funds to appear in your account.

---
This is an automated notification from Stablecoin Payment Gateway.
`

const dailySettlementHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Daily Settlement Report</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #3b82f6;">Daily Settlement Report</h2>
        <p>Here's your daily settlement summary for {{date}}.</p>

        <div style="background: #f3f4f6; padding: 15px; border-radius: 5px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Total Payments:</strong> {{total_payments}}</p>
            <p style="margin: 5px 0;"><strong>Total Payouts:</strong> {{total_payouts}}</p>
            <p style="margin: 5px 0;"><strong>Total Volume:</strong> {{total_volume}} VND</p>
            <p style="margin: 5px 0;"><strong>Revenue:</strong> {{revenue}} VND</p>
        </div>

        <p>View detailed analytics in your dashboard.</p>

        <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">
            This is an automated daily report from Stablecoin Payment Gateway.
        </p>
    </div>
</body>
</html>
`

const dailySettlementText = `
Daily Settlement Report
========================

Here's your daily settlement summary for {{date}}.

Summary:
- Total Payments: {{total_payments}}
- Total Payouts: {{total_payouts}}
- Total Volume: {{total_volume}} VND
- Revenue: {{revenue}} VND

View detailed analytics in your dashboard.

---
This is an automated daily report from Stablecoin Payment Gateway.
`

const kycApprovedHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>KYC Approved - Welcome!</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #10b981;">🎉 Welcome to Stablecoin Payment Gateway!</h2>
        <p>Congratulations! Your KYC verification has been approved.</p>

        <div style="background: #f3f4f6; padding: 15px; border-radius: 5px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Merchant ID:</strong> {{merchant_id}}</p>
            <p style="margin: 5px 0;"><strong>API Key:</strong> <code style="background: #e5e7eb; padding: 2px 5px;">{{api_key}}</code></p>
        </div>

        <p><strong>Next Steps:</strong></p>
        <ol>
            <li>Integrate our payment API using your API key</li>
            <li>Test payments in sandbox mode</li>
            <li>Go live and start accepting crypto payments!</li>
        </ol>

        <p style="color: #dc2626; background: #fee2e2; padding: 10px; border-radius: 5px;">
            <strong>Security Note:</strong> Keep your API key secure. Never share it publicly or commit it to version control.
        </p>

        <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">
            Need help getting started? Check our documentation or contact support.
        </p>
    </div>
</body>
</html>
`

const kycApprovedText = `
Welcome to Stablecoin Payment Gateway!
=======================================

Congratulations! Your KYC verification has been approved.

Account Details:
- Merchant ID: {{merchant_id}}
- API Key: {{api_key}}

Next Steps:
1. Integrate our payment API using your API key
2. Test payments in sandbox mode
3. Go live and start accepting crypto payments!

SECURITY NOTE: Keep your API key secure. Never share it publicly or commit it to version control.

---
Need help getting started? Check our documentation or contact support.
`

const kycRejectedHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>KYC Verification Update</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #dc2626;">KYC Verification Update</h2>
        <p>Thank you for submitting your KYC verification. Unfortunately, we need additional information.</p>

        <div style="background: #fee2e2; padding: 15px; border-radius: 5px; margin: 20px 0; border-left: 4px solid #dc2626;">
            <p style="margin: 5px 0;"><strong>Merchant ID:</strong> {{merchant_id}}</p>
            <p style="margin: 5px 0;"><strong>Reason:</strong> {{reason}}</p>
        </div>

        <p><strong>What to do next:</strong></p>
        <ol>
            <li>Review the reason for rejection</li>
            <li>Prepare the required documents</li>
            <li>Resubmit your KYC application</li>
        </ol>

        <p>If you have any questions, please contact our support team.</p>

        <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">
            This is an automated notification from Stablecoin Payment Gateway.
        </p>
    </div>
</body>
</html>
`

const kycRejectedText = `
KYC Verification Update
========================

Thank you for submitting your KYC verification. Unfortunately, we need additional information.

Details:
- Merchant ID: {{merchant_id}}
- Reason: {{reason}}

What to do next:
1. Review the reason for rejection
2. Prepare the required documents
3. Resubmit your KYC application

If you have any questions, please contact our support team.

---
This is an automated notification from Stablecoin Payment Gateway.
`

const treasuryAlertHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Treasury Alert</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #f59e0b;">⚠️ Treasury Alert: {{alert_type}}</h2>
        <p>This is an automated alert from the treasury monitoring system.</p>

        <div style="background: #fef3c7; padding: 15px; border-radius: 5px; margin: 20px 0; border-left: 4px solid #f59e0b;">
            <p style="margin: 5px 0;"><strong>Alert Type:</strong> {{alert_type}}</p>
            <p style="margin: 5px 0;"><strong>Wallet:</strong> <code>{{wallet_address}}</code></p>
            <p style="margin: 5px 0;"><strong>Current Balance:</strong> {{balance}}</p>
            <p style="margin: 5px 0;"><strong>Threshold:</strong> {{threshold}}</p>
            <p style="margin: 5px 0;"><strong>Time:</strong> {{timestamp}}</p>
        </div>

        <p><strong>Recommended Action:</strong> {{recommended_action}}</p>

        <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">
            This is an automated alert from the treasury monitoring system.
        </p>
    </div>
</body>
</html>
`

const treasuryAlertText = `
Treasury Alert: {{alert_type}}
===============================

This is an automated alert from the treasury monitoring system.

Alert Details:
- Alert Type: {{alert_type}}
- Wallet: {{wallet_address}}
- Current Balance: {{balance}}
- Threshold: {{threshold}}
- Time: {{timestamp}}

Recommended Action: {{recommended_action}}

---
This is an automated alert from the treasury monitoring system.
`

const complianceAlertHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Compliance Alert</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #dc2626;">🚨 Compliance Alert - Action Required</h2>
        <p>A compliance issue requires your immediate attention.</p>

        <div style="background: #fee2e2; padding: 15px; border-radius: 5px; margin: 20px 0; border-left: 4px solid #dc2626;">
            <p style="margin: 5px 0;"><strong>Alert Type:</strong> {{alert_type}}</p>
            <p style="margin: 5px 0;"><strong>Merchant ID:</strong> {{merchant_id}}</p>
            <p style="margin: 5px 0;"><strong>Details:</strong> {{details}}</p>
            <p style="margin: 5px 0;"><strong>Time:</strong> {{timestamp}}</p>
        </div>

        <p><strong>Required Action:</strong></p>
        <p>{{required_action}}</p>

        <p style="color: #dc2626; background: #fee2e2; padding: 10px; border-radius: 5px;">
            <strong>Important:</strong> This requires immediate manual review. Please log in to the admin panel.
        </p>

        <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">
            This is an automated compliance alert from Stablecoin Payment Gateway.
        </p>
    </div>
</body>
</html>
`

const complianceAlertText = `
Compliance Alert - Action Required
===================================

A compliance issue requires your immediate attention.

Alert Details:
- Alert Type: {{alert_type}}
- Merchant ID: {{merchant_id}}
- Details: {{details}}
- Time: {{timestamp}}

Required Action:
{{required_action}}

IMPORTANT: This requires immediate manual review. Please log in to the admin panel.

---
This is an automated compliance alert from Stablecoin Payment Gateway.
`
