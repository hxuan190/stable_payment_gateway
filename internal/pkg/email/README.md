# Email Package

This package provides email sending functionality for the Stablecoin Payment Gateway, with SendGrid integration and templated emails.

## Features

- ✅ **SendGrid Integration** - Production-ready email delivery via SendGrid API
- ✅ **Email Templates** - Pre-built, professional HTML and text email templates
- ✅ **Retry Logic** - Automatic retry with exponential backoff (3 attempts: 1s, 5s, 15s)
- ✅ **Template Rendering** - Dynamic content with `{{variable}}` placeholders
- ✅ **Development Mode** - Gracefully falls back to logging when SendGrid is not configured

## Quick Start

### 1. Configuration

Set the SendGrid API key in your environment:

```bash
SENDGRID_API_KEY=your_sendgrid_api_key_here
EMAIL_FROM=noreply@yourdomain.com
```

### 2. Initialize SendGrid Provider

```go
import "github.com/hxuan190/stable_payment_gateway/internal/pkg/email"

provider := email.NewSendGridProvider(email.SendGridConfig{
    APIKey:    os.Getenv("SENDGRID_API_KEY"),
    FromEmail: os.Getenv("EMAIL_FROM"),
    FromName:  "Stablecoin Payment Gateway",
    Logger:    logger,
})
```

### 3. Send an Email

```go
// Get template
template := email.GetTemplate(email.TemplatePaymentConfirmation)

// Prepare data
data := map[string]interface{}{
    "payment_id": "pay_123456",
    "amount":     "100.00",
    "currency":   "USDT",
    "timestamp":  email.FormatTimestamp(time.Now()),
}

// Render email
renderedEmail, err := email.RenderTemplate(template, data)
if err != nil {
    log.Fatal(err)
}

// Set recipient and send
renderedEmail.To = "merchant@example.com"
err = provider.Send(context.Background(), renderedEmail)
if err != nil {
    log.Fatal(err)
}
```

## Available Templates

### Payment Confirmation
```go
email.TemplatePaymentConfirmation
```
**Required Fields**: `payment_id`, `amount`, `currency`, `timestamp`

**Use Case**: Notify merchant when crypto payment is confirmed

---

### Payout Approved
```go
email.TemplatePayoutApproved
```
**Required Fields**: `payout_id`, `amount`, `timestamp`

**Use Case**: Notify merchant when payout request is approved

---

### Payout Completed
```go
email.TemplatePayoutCompleted
```
**Required Fields**: `payout_id`, `amount`, `reference_number`, `timestamp`

**Use Case**: Notify merchant when bank transfer is completed

---

### Daily Settlement Report
```go
email.TemplateDailySettlement
```
**Required Fields**: `date`, `total_payments`, `total_payouts`, `total_volume`, `revenue`

**Use Case**: Send daily operations summary to ops team

---

### KYC Approved
```go
email.TemplateKYCApproved
```
**Required Fields**: `merchant_id`, `api_key`

**Use Case**: Welcome new merchant with API credentials

---

### KYC Rejected
```go
email.TemplateKYCRejected
```
**Required Fields**: `merchant_id`, `reason`

**Use Case**: Notify merchant of KYC issues

---

### Treasury Alert
```go
email.TemplateTreasuryAlert
```
**Required Fields**: `alert_type`, `wallet_address`, `balance`, `threshold`, `recommended_action`

**Use Case**: Alert ops team when wallet balance exceeds/falls below threshold

---

### Compliance Alert
```go
email.TemplateComplianceAlert
```
**Required Fields**: `alert_type`, `merchant_id`, `details`, `required_action`

**Use Case**: Alert ops team of compliance issues (sanctions, AML, etc.)

---

## Template Customization

Templates are defined in `templates.go`. Each template has:
- **Subject** - with `{{variable}}` placeholders
- **HTMLBody** - styled HTML version
- **TextBody** - plain text fallback
- **RequiredData** - list of required data fields

### Example Custom Template

```go
const myCustomHTML = `
<!DOCTYPE html>
<html>
<body>
    <h2>Hello {{merchant_name}}!</h2>
    <p>Your balance is: {{balance}} VND</p>
</body>
</html>
`

const myCustomText = `
Hello {{merchant_name}}!
Your balance is: {{balance}} VND
`

// Add to GetTemplate() switch statement
case TemplateMyCustom:
    return &EmailTemplate{
        Subject:     "Balance Update for {{merchant_name}}",
        HTMLBody:    myCustomHTML,
        TextBody:    myCustomText,
        RequireData: []string{"merchant_name", "balance"},
    }
```

## Retry Logic

SendGrid provider automatically retries failed sends:
- **Attempt 1**: Immediate
- **Attempt 2**: After 1 second
- **Attempt 3**: After 5 seconds (total 6s)
- **Attempt 4**: After 15 seconds (total 21s)

After 3 attempts, the error is returned to the caller.

## Error Handling

```go
err := provider.Send(ctx, renderedEmail)
if err != nil {
    // Email failed after all retry attempts
    // Log error and potentially:
    // - Store in failed_emails table for manual retry
    // - Alert ops team
    // - Fall back to alternative notification channel
}
```

## Development Mode

When `SENDGRID_API_KEY` is not set, emails are logged instead of sent:

```
INFO Email notification (development mode: logging only, not actually sent)
     to="merchant@example.com"
     subject="Payment Confirmed - pay_123456"
     type="payment_confirmation"
```

This allows development and testing without sending actual emails.

## SendGrid Setup

### 1. Create SendGrid Account
Sign up at https://sendgrid.com

### 2. Create API Key
1. Go to Settings > API Keys
2. Create API Key with "Mail Send" permissions
3. Copy the key (you won't see it again!)

### 3. Verify Sender Email
1. Go to Settings > Sender Authentication
2. Verify your "from" email address
3. Or set up domain authentication for better deliverability

### 4. Test Email
```bash
curl -X POST https://api.sendgrid.com/v3/mail/send \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "personalizations": [{
      "to": [{"email": "test@example.com"}]
    }],
    "from": {"email": "noreply@yourdomain.com"},
    "subject": "Test Email",
    "content": [{
      "type": "text/plain",
      "value": "This is a test email"
    }]
  }'
```

## Production Considerations

### Rate Limits
SendGrid free tier:
- **100 emails/day** (Free)
- **40,000 emails/month** for first month (Essentials)
- **100,000 emails/month** (Pro)

Monitor usage and upgrade as needed.

### Bounce Handling
Configure webhook to receive bounce/spam notifications:
1. Go to Settings > Mail Settings > Event Webhook
2. Enable and set endpoint: `https://yourdomain.com/api/v1/webhooks/sendgrid`
3. Handle bounces, spam reports, unsubscribes

### IP Warmup
If sending high volume (>10k/day), use dedicated IP and warm it up gradually to build sender reputation.

### DKIM/SPF
Configure domain authentication for better deliverability:
- Reduces spam filtering
- Shows verified sender in recipient's email client
- Required for high-volume sending

## Integration with Notification Service

The notification service automatically uses email templates:

```go
// Initialize notification service with SendGrid
notificationService := service.NewNotificationService(service.NotificationServiceConfig{
    EmailSender: os.Getenv("EMAIL_FROM"),
    EmailAPIKey: os.Getenv("SENDGRID_API_KEY"),
    Logger:      logger,
})

// Send payment confirmation
err := notificationService.SendPaymentConfirmationEmail(
    ctx,
    "merchant@example.com",
    "pay_123456",
    "100.00",
    "USDT",
)
```

## Monitoring

Track email metrics:
- **Delivery Rate**: % of emails successfully delivered
- **Open Rate**: % of emails opened (requires tracking pixel)
- **Bounce Rate**: % of emails that bounced
- **Spam Rate**: % marked as spam

SendGrid dashboard provides these metrics.

## Troubleshooting

### Emails Not Sending
1. Check `SENDGRID_API_KEY` is set
2. Verify API key has "Mail Send" permissions
3. Confirm sender email is verified in SendGrid
4. Check logs for error messages

### Emails Going to Spam
1. Set up domain authentication (DKIM/SPF)
2. Use consistent "from" address
3. Avoid spam trigger words in subject/body
4. Monitor bounce rate (remove bounced addresses)

### Template Rendering Errors
1. Verify all required data fields are provided
2. Check for typos in variable names
3. Ensure data types match template expectations

## Security

- ✅ **Never commit API keys** - use environment variables
- ✅ **Validate recipient addresses** - prevent email injection
- ✅ **Rate limit** - prevent abuse
- ✅ **Sanitize data** - escape HTML in user-provided content
- ✅ **Monitor usage** - detect anomalies

---

**Last Updated**: 2025-11-20
**Package Version**: 1.0.0
