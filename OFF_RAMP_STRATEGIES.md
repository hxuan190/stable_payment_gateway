# Advanced Off-ramp Strategies

**Project**: Stablecoin Payment Gateway - Withdrawal Module
**Last Updated**: 2025-11-19
**Status**: Design Phase (PRD v2.2)

---

## 🎯 Overview

Flexible **VND withdrawal strategies** allowing merchants to withdraw funds on their own terms.

### Three Modes

1. **On-Demand**: Manual withdrawal requests
2. **Scheduled**: Auto-withdraw on fixed schedule (e.g., every Friday 4PM)
3. **Threshold-based**: Auto-trigger when balance exceeds threshold

---

## 🔄 Mode 1: On-Demand Withdrawal

### User Flow

```
1. Merchant logs into dashboard
2. Clicks "Withdraw VND" button
3. Enters:
   - Amount (VND)
   - Bank account details
4. System checks balance sufficient
5. Create payout request (status: pending)
6. Ops team reviews manually (MVP)
7. Ops approves → Execute bank transfer
8. Mark payout completed
9. Notify merchant
```

### API Endpoint

```typescript
POST /api/v1/merchant/payout/request
Authorization: Bearer {merchant_jwt}

Request Body:
{
  "amount_vnd": 50000000, // 50M VND
  "bank_name": "Vietcombank",
  "bank_account_number": "1234567890",
  "bank_account_name": "CONG TY ABC"
}

Response (201 Created):
{
  "payout_id": "payout_7c9e6679",
  "amount_vnd": 50000000,
  "fee_vnd": 50000,
  "status": "pending",
  "estimated_completion_time": "24-48 hours"
}
```

### Database Record

```sql
INSERT INTO payouts (
  merchant_id,
  amount_vnd,
  fee_vnd,
  bank_name,
  bank_account_number,
  bank_account_name,
  status
) VALUES (
  'merchant_uuid',
  50000000,
  50000,
  'Vietcombank',
  '1234567890',
  'CONG TY ABC',
  'pending'
);
```

---

## 📅 Mode 2: Scheduled Withdrawal

### Configuration

Merchant sets:
- **Frequency**: Weekly or Monthly
- **Day**: Friday (for weekly), 1st (for monthly)
- **Time**: 16:00 Vietnam Time
- **Auto-withdraw percentage**: 80% of balance

### Database Schema

```sql
CREATE TABLE payout_schedules (
    id UUID PRIMARY KEY,
    merchant_id UUID UNIQUE REFERENCES merchants(id),

    -- Scheduled withdrawal
    scheduled_enabled BOOLEAN DEFAULT FALSE,
    scheduled_frequency VARCHAR(20), -- 'weekly', 'monthly'
    scheduled_day_of_week INTEGER, -- 0=Sunday, 6=Saturday
    scheduled_day_of_month INTEGER, -- 1-31
    scheduled_time TIME, -- e.g., '16:00:00'
    scheduled_withdraw_percentage INTEGER DEFAULT 80,

    last_scheduled_run_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Configuration API

```typescript
POST /api/v1/merchant/payout/schedule
Authorization: Bearer {merchant_jwt}

Request Body:
{
  "enabled": true,
  "frequency": "weekly",
  "day_of_week": 5, // Friday
  "time": "16:00",
  "withdraw_percentage": 80
}

Response (200 OK):
{
  "message": "Scheduled payout configured",
  "schedule": {
    "enabled": true,
    "frequency": "weekly",
    "next_run": "2025-11-22T09:00:00Z" // Friday 16:00 VN = 09:00 UTC
  }
}
```

### Scheduler Worker

```typescript
class PayoutScheduler {
  async runScheduledPayouts() {
    const now = new Date()
    const dayOfWeek = now.getDay()
    const dayOfMonth = now.getDate()
    const hour = now.getHours()
    const minute = now.getMinutes()

    // Find merchants with schedule matching current time
    const schedules = await db.payout_schedules.findMany({
      where: {
        scheduled_enabled: true,
        OR: [
          {
            scheduled_frequency: 'weekly',
            scheduled_day_of_week: dayOfWeek
          },
          {
            scheduled_frequency: 'monthly',
            scheduled_day_of_month: dayOfMonth
          }
        ]
      },
      include: { merchant: true }
    })

    for (const schedule of schedules) {
      // Check if time matches (within 1-minute window)
      const [schedHour, schedMin] = schedule.scheduled_time.split(':')
      if (parseInt(schedHour) !== hour || parseInt(schedMin) !== minute) {
        continue
      }

      // Check if already ran today
      if (this.ranToday(schedule.last_scheduled_run_at)) {
        continue
      }

      // Get merchant balance
      const balance = await this.getMerchantBalance(schedule.merchant_id)
      const withdrawAmount = balance.available_vnd * (schedule.scheduled_withdraw_percentage / 100)

      if (withdrawAmount < 1000000) { // Min 1M VND
        console.log(`Skip: balance too low (${withdrawAmount} VND)`)
        continue
      }

      // Create payout request
      await db.payouts.create({
        data: {
          merchant_id: schedule.merchant_id,
          amount_vnd: withdrawAmount,
          fee_vnd: 50000,
          bank_name: schedule.merchant.bank_name,
          bank_account_number: schedule.merchant.bank_account_number,
          bank_account_name: schedule.merchant.bank_account_name,
          status: 'requested',
          trigger_type: 'scheduled'
        }
      })

      // Update last run time
      await db.payout_schedules.update({
        where: { id: schedule.id },
        data: { last_scheduled_run_at: now }
      })

      // Notify merchant
      await this.notifyMerchant(schedule.merchant_id, {
        type: 'scheduled_payout_created',
        amount: withdrawAmount
      })
    }
  }
}

// Cron job: Run every minute
setInterval(() => {
  new PayoutScheduler().runScheduledPayouts()
}, 60 * 1000)
```

---

## 🎯 Mode 3: Threshold-based Withdrawal

### Configuration

Merchant sets:
- **Threshold**: 5,000 USDT (~115M VND)
- **Auto-withdraw percentage**: 90% when triggered

### Database Schema

```sql
ALTER TABLE payout_schedules ADD COLUMN threshold_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE payout_schedules ADD COLUMN threshold_usdt DECIMAL(20, 8);
ALTER TABLE payout_schedules ADD COLUMN threshold_withdraw_percentage INTEGER DEFAULT 90;
ALTER TABLE payout_schedules ADD COLUMN last_threshold_trigger_at TIMESTAMP;
```

### Configuration API

```typescript
POST /api/v1/merchant/payout/threshold
Authorization: Bearer {merchant_jwt}

Request Body:
{
  "enabled": true,
  "threshold_usdt": 5000.00,
  "withdraw_percentage": 90
}

Response (200 OK):
{
  "message": "Threshold payout configured",
  "threshold": {
    "enabled": true,
    "threshold_usdt": 5000.00,
    "withdraw_percentage": 90,
    "current_balance_usdt": 1234.50
  }
}
```

### Monitor Worker

```typescript
class ThresholdMonitor {
  async checkThresholds() {
    const schedules = await db.payout_schedules.findMany({
      where: { threshold_enabled: true },
      include: { merchant: true }
    })

    for (const schedule of schedules) {
      // Get merchant balance (in USDT)
      const balance = await this.getMerchantBalanceUSDT(schedule.merchant_id)

      // Check if threshold exceeded
      if (balance < schedule.threshold_usdt) {
        continue
      }

      // Check if already triggered recently (cooldown: 24 hours)
      if (this.triggeredRecently(schedule.last_threshold_trigger_at, 24)) {
        continue
      }

      // Calculate withdraw amount
      const withdrawUSDT = balance * (schedule.threshold_withdraw_percentage / 100)
      const withdrawVND = withdrawUSDT * this.getExchangeRate('USDT', 'VND')

      // Create payout request
      await db.payouts.create({
        data: {
          merchant_id: schedule.merchant_id,
          amount_vnd: withdrawVND,
          fee_vnd: 50000,
          bank_name: schedule.merchant.bank_name,
          bank_account_number: schedule.merchant.bank_account_number,
          bank_account_name: schedule.merchant.bank_account_name,
          status: 'requested',
          trigger_type: 'threshold'
        }
      })

      // Update last trigger time
      await db.payout_schedules.update({
        where: { id: schedule.id },
        data: { last_threshold_trigger_at: new Date() }
      })

      // Notify merchant
      await this.notifyMerchant(schedule.merchant_id, {
        type: 'threshold_payout_triggered',
        balance: balance,
        threshold: schedule.threshold_usdt,
        amount: withdrawVND
      })
    }
  }
}

// Cron job: Run every hour
setInterval(() => {
  new ThresholdMonitor().checkThresholds()
}, 60 * 60 * 1000)
```

---

## 📊 Admin Approval Workflow

For MVP, all payouts require manual approval.

### Approval UI (Admin Panel)

```typescript
// GET /api/admin/payouts/pending
[
  {
    "payout_id": "payout_7c9e6679",
    "merchant": {
      "id": "merchant_uuid",
      "business_name": "Coffee Shop ABC"
    },
    "amount_vnd": 50000000,
    "fee_vnd": 50000,
    "bank_account_number": "1234567890",
    "trigger_type": "scheduled", // or 'manual', 'threshold'
    "created_at": "2025-11-19T10:00:00Z",
    "merchant_risk_level": "low"
  }
]

// POST /api/admin/payouts/{payout_id}/approve
{
  "approved_by": "admin_user_uuid",
  "notes": "Verified bank account"
}

// POST /api/admin/payouts/{payout_id}/reject
{
  "rejected_by": "admin_user_uuid",
  "reason": "Suspicious activity"
}
```

---

## ✅ Success Criteria

- [x] **Scheduled payouts run on time** (±1 minute accuracy)
- [x] **Threshold triggers < 1 hour** after balance exceeds
- [x] **Payout success rate > 98%** (successful transfers / total requests)
- [x] **Merchant satisfaction > 85%** with withdrawal flexibility

---

**Document Status**: Design Phase
**Owner**: Backend Team
**Last Updated**: 2025-11-19
