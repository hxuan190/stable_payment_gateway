# PRODUCT REQUIREMENTS DOCUMENT (PRD) v2.2

**Project**: Cổng Thanh toán Stablecoin & Định danh Hợp nhất (Unified Stablecoin Payment & Compliance Gateway)
**Version**: 2.2 (Stablecoin First & Compliance Heavy)
**Focus**: Tuân thủ sâu (Deep Compliance), Lưu trữ Vô hạn, Thông báo Đa kênh
**Last Updated**: 2025-11-19
**Status**: Design Phase

---

## 1. TỔNG QUAN DỰ ÁN (EXECUTIVE SUMMARY)

### 1.1. Định vị Sản phẩm

Hệ thống thanh toán tập trung hoàn toàn vào **Stablecoin (USDT/USDC)** để loại bỏ rủi ro biến động giá. Hệ thống đóng vai trò là "Bên giữ hộ tạm thời" (Custodial Payment Processor), cho phép Merchant tích lũy tiền và rút (Off-ramp) theo yêu cầu hoặc lịch trình.

**Value Proposition**:
> "Merchant tạo QR → User quét → Crypto payment → Giữ trong Treasury → Merchant rút VND khi cần"

### 1.2. Nguyên tắc Cốt lõi

1. **Stablecoin Only**: Chỉ chấp nhận USDT, USDC trên các chain phổ biến (TRON, BSC, Polygon, Solana, Ethereum). Không hỗ trợ token biến động.
2. **Strict Compliance**: Tuân thủ luật pháp sở tại. KYC bắt buộc lần đầu, AML tự xây dựng (Self-built).
3. **Banking-grade Data**: Dữ liệu được lưu trữ vô hạn (Infinite Retention) theo tiêu chuẩn ngân hàng.
4. **Omni-channel Notification**: Thông báo "bủa vây" Merchant qua mọi kênh (Zalo, Telegram, Email, Loa).

### 1.3. Thị trường Mục tiêu

**Primary Market**: Du lịch Đà Nẵng (Hotels, Restaurants, Tourist Services)

**Target Users**:
- **Merchants**: Chủ khách sạn, nhà hàng, dịch vụ du lịch
- **Payers**: Khách du lịch quốc tế có crypto wallet
- **Ops Team**: Quản lý KYC, payout, OTC settlement

**Market Timing**:
- Tether + Đà Nẵng partnership (Nov 2025)
- Resolution 222/2025/QH15 (International Financial Center)
- Early-mover advantage trong regulatory sandbox

---

## 2. LUỒNG NGƯỜI DÙNG & TRẢI NGHIỆM (USER JOURNEYS)

### 2.1. Luồng KYC (Smart Identity Mapping)

**Triết lý**: "KYC một lần, dùng mãi mãi" (One-time KYC, Lifetime Recognition)

#### First-time Payer Flow:
```
1. User scans QR code
2. System detects wallet address → NOT in identity mapping database
3. Show KYC modal:
   - Upload ID/Passport (OCR auto-fill)
   - Face liveness check (anti-spoofing)
   - Confirm personal info
4. System validates KYC documents
5. Create wallet-to-identity mapping in database
6. Cache in Redis: wallet_address → user_id
7. Allow payment to proceed
```

#### Returning Payer Flow:
```
1. User scans QR code (same wallet address)
2. System detects wallet → FOUND in Redis cache
3. Display: "Welcome back, [User Name]!"
4. Skip KYC entirely → Proceed to payment immediately
```

**Benefits**:
- ✅ Frictionless UX for repeat customers
- ✅ Compliance maintained (KYC done once)
- ✅ Fast recognition (Redis cache < 10ms)

### 2.2. Luồng Thanh toán & Thông báo (Payment & Notification)

```
┌─────────────────────────────────────────────────────────────┐
│ 1. PAYMENT INITIATION                                        │
└─────────────────────────────────────────────────────────────┘
    Merchant creates payment → QR code generated
    Payer scans QR → Wallet detected → KYC check (see 2.1)

┌─────────────────────────────────────────────────────────────┐
│ 2. BLOCKCHAIN CONFIRMATION                                   │
└─────────────────────────────────────────────────────────────┘
    Blockchain listener detects transaction
    → AML Engine screens wallet (sanctions, risk)
    → Transaction validated (amount, memo)
    → Wait for finality (Solana: 13s, BSC: 3min)

┌─────────────────────────────────────────────────────────────┐
│ 3. TREASURY UPDATE                                           │
└─────────────────────────────────────────────────────────────┘
    Credit merchant balance (custodial model)
    → Update ledger (double-entry)
    → Record transaction in audit log

┌─────────────────────────────────────────────────────────────┐
│ 4. OMNI-CHANNEL NOTIFICATION (Broadcast Storm!)              │
└─────────────────────────────────────────────────────────────┘
    ┌────────────────┐
    │ At POS Counter │ → Speaker/TTS: "Đã nhận 10 USDT từ khách John Doe"
    └────────────────┘

    ┌────────────────┐
    │ Boss Phone     │ → Telegram Bot: "💰 +10 USDT (230k VND) - Bàn 5"
    └────────────────┘   → Zalo OA: "Giao dịch mới: 10 USDT. Số dư: 1,234 USDT"

    ┌────────────────┐
    │ Payer Email    │ → Email Invoice: "Payment Confirmed - Receipt #12345"
    └────────────────┘

    ┌────────────────┐
    │ Merchant POS   │ → Webhook: POST to callback_url with tx details
    └────────────────┘
```

**Notification Priority**:
- 🔴 **CRITICAL**: Speaker (immediate audible confirmation)
- 🟡 **HIGH**: Telegram/Zalo (real-time push)
- 🟢 **MEDIUM**: Email (confirmation record)
- ⚪ **LOW**: Webhook (for integration)

---

## 3. YÊU CẦU CHỨC NĂNG CHI TIẾT (FUNCTIONAL REQUIREMENTS)

### MODULE 1: IDENTITY MANAGEMENT (Smart Wallet-to-User Mapping)

#### 1.1. Wallet Identity Mapping

**Objective**: Liên kết vĩnh viễn giữa wallet address và thông tin định danh người dùng

**Database Schema**:
```sql
CREATE TABLE wallet_identity_mappings (
    id UUID PRIMARY KEY,
    wallet_address VARCHAR(255) NOT NULL,
    blockchain VARCHAR(20) NOT NULL, -- 'solana', 'bsc', 'tron', etc.
    user_id UUID NOT NULL REFERENCES users(id),

    -- Identity info (encrypted)
    full_name VARCHAR(255) NOT NULL,
    id_type VARCHAR(20), -- 'passport', 'national_id', 'drivers_license'
    id_number VARCHAR(100),
    nationality VARCHAR(10),
    date_of_birth DATE,

    -- KYC verification
    kyc_status VARCHAR(20) DEFAULT 'verified',
    kyc_verified_at TIMESTAMP,
    kyc_provider VARCHAR(50), -- 'sumsub', 'onfido', 'manual'

    -- Metadata
    first_seen_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    payment_count INTEGER DEFAULT 0,
    total_volume_usd DECIMAL(20, 2) DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(wallet_address, blockchain)
);

CREATE INDEX idx_wallet_mapping_address ON wallet_identity_mappings(wallet_address, blockchain);
CREATE INDEX idx_wallet_mapping_user ON wallet_identity_mappings(user_id);
```

**Redis Caching Strategy**:
```
Key: wallet:{blockchain}:{address}
Value: {user_id, full_name, kyc_status, last_used_at}
TTL: 7 days (604800 seconds)

Example:
wallet:solana:8xK7zY9... → {user_id: "uuid", full_name: "John Doe", kyc_status: "verified"}
```

**API Endpoints**:
```typescript
// Check if wallet has KYC
GET /api/v1/wallet/:blockchain/:address/kyc-status
Response: {
  "has_kyc": true,
  "user": {
    "id": "uuid",
    "full_name": "John Doe",
    "kyc_status": "verified"
  },
  "stats": {
    "payment_count": 12,
    "total_volume_usd": 1250.50
  }
}

// Initiate KYC for new wallet
POST /api/v1/wallet/kyc/initiate
Body: {
  "wallet_address": "8xK7zY9...",
  "blockchain": "solana",
  "full_name": "John Doe",
  "id_type": "passport",
  "id_number": "AB1234567"
}

// Upload KYC documents
POST /api/v1/wallet/kyc/upload
Multipart form-data:
  - id_front: File
  - id_back: File
  - selfie: File

// Verify KYC (Face liveness)
POST /api/v1/wallet/kyc/verify
Body: {
  "session_id": "uuid",
  "liveness_video": "base64_encoded_video"
}
```

#### 1.2. Face Liveness Detection

**Purpose**: Chống giả mạo (anti-spoofing), đảm bảo người thật

**Options**:
1. **Sumsub** (recommended for MVP): API-based, $0.50/check
2. **Onfido**: More expensive but higher accuracy
3. **Self-hosted** (Phase 2): Open-source models (FaceNet, ArcFace)

**Flow**:
1. User uploads ID photo
2. System extracts face from ID
3. User records short video (blink, turn head)
4. AI compares video face vs ID face
5. Liveness score > 0.85 → Pass

### MODULE 2: PAYMENT & TREASURY (Custodial Model)

#### 2.1. Stablecoin Processing

**Supported Chains & Tokens**:

| Chain | Tokens | Priority | Reason |
|-------|--------|----------|--------|
| **TRON** | USDT | 🔴 **HIGH** | Cheapest fees (~$1), huge in Asia |
| **Solana** | USDT, USDC | 🔴 **HIGH** | Fast finality (13s), low fees ($0.001) |
| **BSC** | USDT, BUSD | 🟡 **MEDIUM** | Popular in SEA, moderate fees ($0.20) |
| **Polygon** | USDT, USDC | 🟢 **LOW** | Growing, low fees ($0.01) |
| **Ethereum** | USDT, USDC | ⚪ **FUTURE** | Expensive gas ($5-50), for large amounts only |

**Multi-chain Listener Architecture**:
```
┌─────────────────────────────────────────────────────────┐
│              Blockchain Listener Orchestrator           │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐│
│  │  TRON    │  │ Solana   │  │   BSC    │  │ Polygon ││
│  │ Listener │  │ Listener │  │ Listener │  │Listener ││
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬────┘│
│       │             │              │             │     │
│       └─────────────┴──────────────┴─────────────┘     │
│                         ↓                               │
│              Transaction Validator                      │
│              (Verify amount, memo, finality)            │
│                         ↓                               │
│              AML Screening Engine                       │
│                         ↓                               │
│              Treasury Service                           │
│                         ↓                               │
│              Notification Dispatcher                    │
└─────────────────────────────────────────────────────────┘
```

#### 2.2. Custodial Treasury (Giữ tiền thay Merchant)

**Philosophy**: Hệ thống giữ crypto thay merchant, merchant chỉ rút VND khi cần.

**Treasury Architecture**:

```
┌───────────────────────────────────────────────────────┐
│                   HOT WALLETS                          │
│  (Receive payments from payers)                        │
│                                                         │
│  TRON: TYs7zK...  (Balance: 5,000 USDT)               │
│  Solana: 8xK7...  (Balance: 3,200 USDC)               │
│  BSC: 0xAb12...   (Balance: 1,800 USDT)               │
└───────────────────────────┬───────────────────────────┘
                            │
                   Sweeping Worker (every 6 hours)
                   Threshold: > $10,000
                            │
                            ↓
┌───────────────────────────────────────────────────────┐
│                   COLD WALLET                          │
│  (Multi-sig 2-of-3 or MPC)                            │
│                                                         │
│  Balance: $850,000 USDT + USDC                        │
│  Security: Offline signing, Fireblocks/Copper          │
└───────────────────────────────────────────────────────┘
```

**Sweeping Mechanism**:
```typescript
// Sweeping Worker (runs every 6 hours)
class SweepingWorker {
  async sweep() {
    const hotWallets = await getHotWallets()

    for (const wallet of hotWallets) {
      const balance = await wallet.getBalance()
      const threshold = 10000 // $10k USD

      if (balance > threshold) {
        const amountToSweep = balance - 1000 // Keep $1k for gas

        // Transfer to cold wallet
        await wallet.transfer({
          to: COLD_WALLET_ADDRESS,
          amount: amountToSweep,
          requireMultiSig: true
        })

        // Log in audit trail
        await auditLog.create({
          action: 'sweep',
          from: wallet.address,
          to: COLD_WALLET_ADDRESS,
          amount: amountToSweep,
          reason: 'scheduled_sweep'
        })
      }
    }
  }
}
```

**Merchant Balance Tracking**:
```sql
CREATE TABLE merchant_treasury_balances (
    merchant_id UUID PRIMARY KEY,

    -- Crypto balances (custodial)
    usdt_balance DECIMAL(20, 8) DEFAULT 0,
    usdc_balance DECIMAL(20, 8) DEFAULT 0,

    -- VND equivalent (for display)
    total_vnd_value DECIMAL(20, 2) DEFAULT 0,

    -- Available for withdrawal
    available_for_withdrawal_vnd DECIMAL(20, 2) DEFAULT 0,
    pending_withdrawal_vnd DECIMAL(20, 2) DEFAULT 0,

    -- Lifetime stats
    total_received_usdt DECIMAL(20, 8) DEFAULT 0,
    total_received_usdc DECIMAL(20, 8) DEFAULT 0,
    total_withdrawn_vnd DECIMAL(20, 2) DEFAULT 0,

    updated_at TIMESTAMP DEFAULT NOW()
);
```

#### 2.3. Off-ramp Management (Rút tiền VND)

**Withdrawal Modes**:

##### Mode 1: On-Demand (Theo yêu cầu)
```
Merchant clicks "Withdraw VND" button
→ Enter amount VND to withdraw
→ Enter bank account details
→ System checks: balance sufficient?
→ Create payout request (status: pending)
→ Ops team reviews (manual for MVP)
→ Ops team approves → Execute bank transfer
→ Mark payout completed
→ Notify merchant via Zalo/Email
```

##### Mode 2: Scheduled (Lịch trình cố định)
```
Merchant configures:
  - Frequency: Weekly / Monthly
  - Day: Friday
  - Time: 16:00 Vietnam Time
  - Auto-withdraw: 80% of balance

Scheduler Worker (cron job):
  - Runs every day at 16:00
  - Checks merchants with scheduled withdrawal
  - Creates payout requests automatically
  - Sends to ops team for approval
```

##### Mode 3: Threshold-based (Tự động khi đạt ngưỡng)
```
Merchant configures:
  - Threshold: 5,000 USDT (~115M VND)
  - Auto-withdraw: 90% when threshold reached

Balance Monitor Worker:
  - Runs every hour
  - Checks merchant balances
  - If balance > threshold:
    → Create payout request
    → Notify merchant: "Auto-withdrawal triggered"
    → Send to ops for approval
```

**Payout API**:
```typescript
// Create on-demand payout
POST /api/v1/merchant/payout/request
Body: {
  "amount_vnd": 50000000, // 50M VND
  "bank_name": "Vietcombank",
  "bank_account_number": "1234567890",
  "bank_account_name": "CONG TY ABC"
}

// Configure scheduled payout
POST /api/v1/merchant/payout/schedule
Body: {
  "enabled": true,
  "frequency": "weekly", // or "monthly"
  "day_of_week": 5, // Friday (0=Sunday, 6=Saturday)
  "time": "16:00",
  "auto_withdraw_percentage": 80
}

// Configure threshold payout
POST /api/v1/merchant/payout/threshold
Body: {
  "enabled": true,
  "threshold_usdt": 5000,
  "auto_withdraw_percentage": 90
}
```

### MODULE 3: NOTIFICATION CENTER (Omni-channel)

#### 3.1. Plugin Architecture

**Design Philosophy**: Dễ dàng thêm kênh mới mà không ảnh hưởng code cũ

```typescript
// Notification Plugin Interface
interface NotificationPlugin {
  name: string
  priority: number // 1=highest, 5=lowest

  send(message: NotificationMessage): Promise<SendResult>
  validateConfig(config: any): boolean
  getStatus(): PluginStatus
}

// Notification Message
interface NotificationMessage {
  type: 'payment_received' | 'payout_approved' | 'alert'
  recipient: Recipient
  data: {
    amount?: number
    currency?: string
    merchant_name?: string
    transaction_id?: string
    [key: string]: any
  }
  template?: string
}

// Notification Dispatcher
class NotificationDispatcher {
  private plugins: Map<string, NotificationPlugin> = new Map()

  registerPlugin(plugin: NotificationPlugin) {
    this.plugins.set(plugin.name, plugin)
  }

  async notify(message: NotificationMessage, channels: string[]) {
    const results: SendResult[] = []

    for (const channelName of channels) {
      const plugin = this.plugins.get(channelName)
      if (plugin) {
        const result = await plugin.send(message)
        results.push(result)
      }
    }

    return results
  }
}
```

#### 3.2. Channel Implementations

##### Channel 1: Speaker/TTS (Loa thông báo tại quầy)

**Use Case**: Thu ngân nghe thông báo ngay lập tức khi nhận tiền

**Tech Stack**:
- **Frontend**: WebSocket connection to backend
- **Backend**: Socket.io or native WebSocket
- **TTS Engine**:
  - Option 1: Google Cloud Text-to-Speech API (best quality, $4/1M chars)
  - Option 2: Browser native `speechSynthesis` API (free, lower quality)
  - Option 3: Self-hosted TTS (Coqui TTS, Mozilla TTS)

**Implementation**:
```typescript
// Speaker Plugin
class SpeakerPlugin implements NotificationPlugin {
  name = 'speaker'
  priority = 1 // Highest priority

  async send(message: NotificationMessage): Promise<SendResult> {
    // Generate TTS audio
    const text = this.generateTextFromMessage(message)
    // Example: "Đã nhận 10 USDT từ khách John Doe, bàn số 5"

    const audioBuffer = await this.textToSpeech(text, 'vi-VN')

    // Send to POS device via WebSocket
    await this.socketServer.emit('play-audio', {
      merchantId: message.recipient.merchantId,
      audio: audioBuffer,
      priority: 'high'
    })

    return { success: true, channel: 'speaker' }
  }

  private generateTextFromMessage(msg: NotificationMessage): string {
    if (msg.type === 'payment_received') {
      return `Đã nhận ${msg.data.amount} ${msg.data.currency} từ khách ${msg.data.customer_name || 'ẩn danh'}`
    }
    // ... other templates
  }
}
```

**Merchant POS Integration**:
```javascript
// Merchant POS App (React/Next.js)
const socket = io('wss://gateway.example.com')

socket.on('play-audio', async (data) => {
  const audio = new Audio(data.audio)
  audio.volume = 0.8 // 80% volume
  await audio.play()

  // Show toast notification
  toast.success(`💰 Đã nhận ${data.amount} ${data.currency}`)
})
```

##### Channel 2: Telegram Bot

**Use Case**: Boss nhận thông báo real-time trên điện thoại

**Setup**:
1. Create bot via @BotFather
2. Get bot token
3. Merchant sends `/start` to bot
4. Bot saves `chat_id` to database

**Implementation**:
```typescript
class TelegramPlugin implements NotificationPlugin {
  name = 'telegram'
  priority = 2

  private bot: TelegramBot

  constructor(token: string) {
    this.bot = new TelegramBot(token, { polling: true })
    this.setupCommands()
  }

  async send(message: NotificationMessage): Promise<SendResult> {
    const chatId = message.recipient.telegram_chat_id

    if (!chatId) {
      return { success: false, error: 'No Telegram chat_id configured' }
    }

    const text = this.formatMessage(message)

    await this.bot.sendMessage(chatId, text, {
      parse_mode: 'Markdown',
      disable_notification: message.priority === 'low'
    })

    return { success: true, channel: 'telegram' }
  }

  private formatMessage(msg: NotificationMessage): string {
    if (msg.type === 'payment_received') {
      return `
💰 *Giao dịch mới*

Số tiền: ${msg.data.amount} ${msg.data.currency}
Giá trị VND: ${msg.data.amount_vnd} VND
Khách hàng: ${msg.data.customer_name || 'N/A'}
Thời gian: ${new Date().toLocaleString('vi-VN')}

Số dư hiện tại: ${msg.data.balance} USDT
      `.trim()
    }
    // ... other templates
  }

  private setupCommands() {
    this.bot.onText(/\/start/, async (msg) => {
      const chatId = msg.chat.id
      // Save chat_id to database for merchant
      await this.saveChatId(chatId)

      await this.bot.sendMessage(chatId,
        'Chào mừng! Bot đã được kích hoạt. Bạn sẽ nhận thông báo khi có giao dịch mới.'
      )
    })

    this.bot.onText(/\/balance/, async (msg) => {
      const chatId = msg.chat.id
      const balance = await this.getMerchantBalance(chatId)
      await this.bot.sendMessage(chatId, `Số dư: ${balance} USDT`)
    })
  }
}
```

##### Channel 3: Zalo OA/ZNS

**Use Case**: Thông báo cho merchant tại Việt Nam (Zalo is king!)

**Zalo Options**:
1. **Zalo OA (Official Account)**: Free messaging, needs approval
2. **Zalo ZNS (Notification Service)**: Template-based, $0.01/message

**Setup**:
1. Register Zalo OA: https://oa.zalo.me/
2. Get OA ID and access token
3. Create message templates (for ZNS)

**Implementation**:
```typescript
class ZaloPlugin implements NotificationPlugin {
  name = 'zalo'
  priority = 2

  private accessToken: string
  private oaId: string

  async send(message: NotificationMessage): Promise<SendResult> {
    const userId = message.recipient.zalo_user_id

    if (!userId) {
      return { success: false, error: 'No Zalo user_id' }
    }

    // Use ZNS template
    const response = await axios.post(
      'https://business.openapi.zalo.me/message/template',
      {
        phone: message.recipient.phone,
        template_id: '123456', // Template approved by Zalo
        template_data: {
          amount: message.data.amount,
          currency: message.data.currency,
          time: new Date().toISOString()
        }
      },
      {
        headers: {
          'access_token': this.accessToken
        }
      }
    )

    return { success: response.data.error === 0, channel: 'zalo' }
  }
}
```

**Zalo Message Template Example**:
```
Giao dịch mới - {{merchant_name}}

Số tiền: {{amount}} {{currency}}
Giá trị: {{amount_vnd}} VND
Thời gian: {{time}}

Số dư: {{balance}} USDT
```

##### Channel 4: Email

**Use Case**: Invoice, sao kê tháng, báo cáo

**Tech Stack**:
- **SendGrid** (recommended): 100 emails/day free, $15/mo for 40k emails
- **AWS SES**: $0.10/1000 emails
- **SMTP**: Self-hosted (Postfix, Sendmail)

**Implementation**:
```typescript
class EmailPlugin implements NotificationPlugin {
  name = 'email'
  priority = 4 // Lower priority (not urgent)

  private sgMail: any

  constructor(apiKey: string) {
    this.sgMail = require('@sendgrid/mail')
    this.sgMail.setApiKey(apiKey)
  }

  async send(message: NotificationMessage): Promise<SendResult> {
    const emailContent = this.generateEmailHTML(message)

    const msg = {
      to: message.recipient.email,
      from: 'noreply@gateway.example.com',
      subject: this.getSubject(message.type),
      html: emailContent,
      attachments: message.data.attachments || []
    }

    await this.sgMail.send(msg)

    return { success: true, channel: 'email' }
  }

  private generateEmailHTML(message: NotificationMessage): string {
    if (message.type === 'payment_received') {
      return `
        <!DOCTYPE html>
        <html>
        <body>
          <h2>Payment Received</h2>
          <p>Amount: ${message.data.amount} ${message.data.currency}</p>
          <p>Transaction ID: ${message.data.transaction_id}</p>
          <p>Time: ${new Date().toISOString()}</p>

          <a href="https://dashboard.example.com/payments/${message.data.payment_id}">
            View Details
          </a>
        </body>
        </html>
      `
    }
    // ... other templates
  }
}
```

##### Channel 5: Webhook

**Use Case**: Tích hợp với hệ thống POS khác, ERP, accounting software

**Implementation**:
```typescript
class WebhookPlugin implements NotificationPlugin {
  name = 'webhook'
  priority = 3

  async send(message: NotificationMessage): Promise<SendResult> {
    const webhookUrl = message.recipient.webhook_url
    const webhookSecret = message.recipient.webhook_secret

    if (!webhookUrl) {
      return { success: false, error: 'No webhook URL configured' }
    }

    // Create payload
    const payload = {
      event: message.type,
      data: message.data,
      timestamp: new Date().toISOString()
    }

    // Sign payload with HMAC
    const signature = crypto
      .createHmac('sha256', webhookSecret)
      .update(JSON.stringify(payload))
      .digest('hex')

    // Send webhook with retry logic
    const result = await this.sendWithRetry(webhookUrl, payload, signature)

    return result
  }

  private async sendWithRetry(
    url: string,
    payload: any,
    signature: string,
    maxRetries = 3
  ): Promise<SendResult> {
    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        const response = await axios.post(url, payload, {
          headers: {
            'X-Webhook-Signature': signature,
            'Content-Type': 'application/json'
          },
          timeout: 5000 // 5 seconds
        })

        if (response.status === 200) {
          return { success: true, channel: 'webhook' }
        }
      } catch (error) {
        if (attempt === maxRetries) {
          // Log failed webhook for manual review
          await this.logFailedWebhook(url, payload, error)
          return { success: false, error: error.message, channel: 'webhook' }
        }

        // Exponential backoff: 2s, 4s, 8s
        await this.sleep(2 ** attempt * 1000)
      }
    }
  }
}
```

#### 3.3. Notification Queue System

**Purpose**: Đảm bảo thông báo không bị mất khi system tải cao

**Tech Stack**: Redis Queue (Bull) or RabbitMQ

**Implementation**:
```typescript
import Queue from 'bull'

class NotificationQueue {
  private queue: Queue.Queue

  constructor() {
    this.queue = new Queue('notifications', {
      redis: {
        host: process.env.REDIS_HOST,
        port: 6379
      }
    })

    this.setupProcessors()
  }

  async enqueue(message: NotificationMessage, channels: string[]) {
    await this.queue.add('send', {
      message,
      channels
    }, {
      attempts: 3,
      backoff: {
        type: 'exponential',
        delay: 2000
      }
    })
  }

  private setupProcessors() {
    this.queue.process('send', async (job) => {
      const { message, channels } = job.data

      const dispatcher = new NotificationDispatcher()
      // Register all plugins
      dispatcher.registerPlugin(new SpeakerPlugin())
      dispatcher.registerPlugin(new TelegramPlugin(process.env.TELEGRAM_BOT_TOKEN))
      dispatcher.registerPlugin(new ZaloPlugin(/* config */))
      dispatcher.registerPlugin(new EmailPlugin(process.env.SENDGRID_API_KEY))
      dispatcher.registerPlugin(new WebhookPlugin())

      const results = await dispatcher.notify(message, channels)

      return results
    })
  }
}
```

### MODULE 4: DATA RETENTION (Banking-grade Storage)

#### 4.1. Infinite Retention Philosophy

**Principle**: "Never delete transaction data" (Không bao giờ xóa dữ liệu giao dịch)

**Legal Requirements**:
- Vietnam Law: 7 years minimum
- Banking Standard: 10+ years
- PRD v2.2: **Infinite** (Vô hạn)

**Rationale**:
- Regulatory compliance
- Dispute resolution
- Audit trail
- Historical analysis
- Legal protection

#### 4.2. Storage Tiers (Hot/Cold Architecture)

```
┌────────────────────────────────────────────────────────┐
│ HOT STORAGE (0-12 months)                              │
│ PostgreSQL + Read Replicas                             │
│ - Optimized for fast queries                           │
│ - Full-text search enabled                             │
│ - Used by: Dashboard, Reports, API                     │
│                                                         │
│ Cost: ~$200/month (500GB SSD)                          │
└─────────────────────┬──────────────────────────────────┘
                      │
           Archival Job (monthly)
           Move data older than 12 months
                      │
                      ↓
┌────────────────────────────────────────────────────────┐
│ COLD STORAGE (1+ years)                                │
│ Amazon S3 Glacier Deep Archive                         │
│ - Optimized for long-term retention                    │
│ - Retrieval time: 12-48 hours                          │
│ - Used by: Compliance audits, historical analysis      │
│                                                         │
│ Cost: ~$1/TB/month (99.999999999% durability)          │
└────────────────────────────────────────────────────────┘
```

**Storage Cost Comparison**:

| Storage Type | Cost/TB/month | Retrieval Cost | Retrieval Time | Use Case |
|--------------|---------------|----------------|----------------|----------|
| PostgreSQL SSD | $400 | Instant | <1ms | Active queries |
| S3 Standard | $23 | Free | Instant | Recent archives |
| S3 Glacier | $4 | $0.01/GB | 1-5 hours | Old archives |
| S3 Glacier Deep | $1 | $0.02/GB | 12-48 hours | Compliance only |

**Recommendation**: Use **S3 Glacier** (not Deep Archive) for balance of cost & retrieval speed.

#### 4.3. Archival Process

**Monthly Archival Job**:
```typescript
class DataArchivalWorker {
  async archiveOldData() {
    const cutoffDate = new Date()
    cutoffDate.setMonth(cutoffDate.getMonth() - 12) // 12 months ago

    // Archive payments
    const oldPayments = await this.db.payments.findMany({
      where: {
        created_at: { lt: cutoffDate },
        archived: false
      }
    })

    // Convert to compressed JSON
    const archiveData = {
      type: 'payments',
      month: cutoffDate.toISOString().slice(0, 7), // "2024-11"
      count: oldPayments.length,
      data: oldPayments
    }

    const compressed = gzip(JSON.stringify(archiveData))

    // Upload to S3 Glacier
    await this.s3.putObject({
      Bucket: 'payment-gateway-archives',
      Key: `payments/${cutoffDate.getFullYear()}/${cutoffDate.getMonth() + 1}/archive.json.gz`,
      Body: compressed,
      StorageClass: 'GLACIER'
    }).promise()

    // Mark as archived in DB
    await this.db.payments.updateMany({
      where: { id: { in: oldPayments.map(p => p.id) } },
      data: { archived: true, archived_at: new Date() }
    })

    // Keep IDs and hashes in DB for integrity check
    await this.db.archived_records.createMany({
      data: oldPayments.map(p => ({
        original_id: p.id,
        table_name: 'payments',
        archive_path: `s3://payment-gateway-archives/payments/...`,
        hash: this.hashRecord(p)
      }))
    })
  }

  private hashRecord(record: any): string {
    return crypto.createHash('sha256')
      .update(JSON.stringify(record))
      .digest('hex')
  }
}
```

**Restore Process**:
```typescript
class DataRestoreService {
  async restorePayment(paymentId: string): Promise<Payment> {
    // Check if archived
    const archiveRecord = await this.db.archived_records.findFirst({
      where: { original_id: paymentId }
    })

    if (!archiveRecord) {
      throw new Error('Payment not found in archives')
    }

    // Retrieve from S3 Glacier (initiate restore first)
    const restoreStatus = await this.s3.restoreObject({
      Bucket: 'payment-gateway-archives',
      Key: archiveRecord.archive_path.replace('s3://payment-gateway-archives/', ''),
      RestoreRequest: {
        Days: 7, // Keep in S3 Standard for 7 days
        GlacierJobParameters: {
          Tier: 'Expedited' // Faster: 1-5 hours, $0.03/GB
        }
      }
    }).promise()

    // Poll until restore completes
    await this.waitForRestore(archiveRecord.archive_path)

    // Download and decompress
    const object = await this.s3.getObject({
      Bucket: 'payment-gateway-archives',
      Key: archiveRecord.archive_path.replace('s3://payment-gateway-archives/', '')
    }).promise()

    const decompressed = gunzip(object.Body as Buffer)
    const archiveData = JSON.parse(decompressed.toString())

    // Find specific payment
    const payment = archiveData.data.find(p => p.id === paymentId)

    // Verify integrity
    const hash = this.hashRecord(payment)
    if (hash !== archiveRecord.hash) {
      throw new Error('Data integrity check failed! Archive corrupted.')
    }

    return payment
  }
}
```

#### 4.4. Transaction Hashing (Immutable Ledger)

**Purpose**: Chống sửa đổi dữ liệu quá khứ (tamper-proof)

**Mechanism**: Mỗi transaction được hash với SHA-256

**Schema**:
```sql
CREATE TABLE transaction_hashes (
    id UUID PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL, -- 'payments', 'payouts', 'ledger_entries'
    record_id UUID NOT NULL,

    -- Hash of record data
    data_hash VARCHAR(64) NOT NULL, -- SHA-256 hex

    -- Hash chain (link to previous hash)
    previous_hash VARCHAR(64),

    -- Merkle tree root (for batch verification)
    merkle_root VARCHAR(64),
    batch_id UUID,

    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(table_name, record_id)
);

CREATE INDEX idx_txn_hash_table ON transaction_hashes(table_name);
CREATE INDEX idx_txn_hash_batch ON transaction_hashes(batch_id);
```

**Hash Chain Implementation**:
```typescript
class TransactionHashService {
  async hashTransaction(tableName: string, record: any): Promise<string> {
    // Get previous hash for chain
    const previousHash = await this.getLatestHash(tableName)

    // Create hash input
    const hashInput = {
      table: tableName,
      record_id: record.id,
      data: record,
      previous_hash: previousHash,
      timestamp: new Date().toISOString()
    }

    // Compute SHA-256
    const hash = crypto.createHash('sha256')
      .update(JSON.stringify(hashInput))
      .digest('hex')

    // Store hash
    await this.db.transaction_hashes.create({
      data: {
        table_name: tableName,
        record_id: record.id,
        data_hash: hash,
        previous_hash: previousHash
      }
    })

    return hash
  }

  async verifyIntegrity(tableName: string, recordId: string): Promise<boolean> {
    // Get hash record
    const hashRecord = await this.db.transaction_hashes.findUnique({
      where: { table_name_record_id: { table_name: tableName, record_id: recordId } }
    })

    if (!hashRecord) {
      throw new Error('Hash not found')
    }

    // Get original record from DB (or archive)
    const originalRecord = await this.getRecord(tableName, recordId)

    // Recompute hash
    const recomputedHash = crypto.createHash('sha256')
      .update(JSON.stringify({
        table: tableName,
        record_id: recordId,
        data: originalRecord,
        previous_hash: hashRecord.previous_hash,
        timestamp: hashRecord.created_at.toISOString()
      }))
      .digest('hex')

    // Compare
    return recomputedHash === hashRecord.data_hash
  }

  async verifyChain(tableName: string): Promise<boolean> {
    const hashes = await this.db.transaction_hashes.findMany({
      where: { table_name: tableName },
      orderBy: { created_at: 'asc' }
    })

    for (let i = 1; i < hashes.length; i++) {
      const current = hashes[i]
      const previous = hashes[i - 1]

      if (current.previous_hash !== previous.data_hash) {
        console.error(`Chain broken at index ${i}`)
        return false
      }
    }

    return true
  }
}
```

**Merkle Tree for Batch Verification**:
```typescript
class MerkleTree {
  static buildTree(hashes: string[]): string {
    if (hashes.length === 0) return ''
    if (hashes.length === 1) return hashes[0]

    const tree: string[][] = [hashes]

    while (tree[tree.length - 1].length > 1) {
      const currentLevel = tree[tree.length - 1]
      const nextLevel: string[] = []

      for (let i = 0; i < currentLevel.length; i += 2) {
        const left = currentLevel[i]
        const right = currentLevel[i + 1] || left // Duplicate if odd

        const combined = crypto.createHash('sha256')
          .update(left + right)
          .digest('hex')

        nextLevel.push(combined)
      }

      tree.push(nextLevel)
    }

    return tree[tree.length - 1][0] // Root hash
  }
}

// Daily batch hash job
async function dailyMerkleRoot() {
  const today = new Date()
  today.setHours(0, 0, 0, 0)

  // Get all transaction hashes from today
  const hashes = await db.transaction_hashes.findMany({
    where: {
      created_at: { gte: today }
    },
    select: { data_hash: true }
  })

  // Build Merkle tree
  const root = MerkleTree.buildTree(hashes.map(h => h.data_hash))

  // Store root
  await db.merkle_roots.create({
    data: {
      date: today,
      root_hash: root,
      transaction_count: hashes.length
    }
  })

  console.log(`Merkle root for ${today.toISOString()}: ${root}`)
}
```

---

## 4. KIẾN TRÚC KỸ THUẬT (TECHNICAL ARCHITECTURE)

### 4.1. System Architecture (Updated for PRD v2.2)

```
┌──────────────────────────────────────────────────────────────────┐
│                        EXTERNAL ACTORS                            │
├───────────────┬──────────────────┬──────────────────┬────────────┤
│  End Users    │    Merchants     │   Blockchain     │ OTC Partner│
│  (Payers)     │    (Business)    │   Networks       │            │
└───────┬───────┴────────┬─────────┴─────────┬────────┴─────┬──────┘
        │                │                   │              │
┌───────▼────────────────▼───────────────────▼──────────────▼──────┐
│                      API GATEWAY LAYER                            │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐        │
│  │ Public API   │  │ Merchant API │  │ Internal Admin  │        │
│  │ - Payment    │  │ - Balance    │  │ - KYC Review    │        │
│  │ - KYC Check  │  │ - Payout     │  │ - Payout Approval│       │
│  └──────────────┘  └──────────────┘  └─────────────────┘        │
└───────────────────────────────┬──────────────────────────────────┘
                                │
┌───────────────────────────────▼──────────────────────────────────┐
│                    APPLICATION LAYER                              │
│                                                                    │
│  ┌────────────────────┐  ┌────────────────────┐                  │
│  │ Identity Mapping   │  │ Payment Service    │                  │
│  │ Service (NEW!)     │  │                    │                  │
│  │ - Wallet→User KYC  │  │ - Create payment   │                  │
│  │ - Face liveness    │  │ - Validate         │                  │
│  └────────────────────┘  │ - Confirm          │                  │
│                          └────────────────────┘                  │
│                                                                    │
│  ┌────────────────────┐  ┌────────────────────┐                  │
│  │ Notification       │  │ Treasury Service   │                  │
│  │ Center (NEW!)      │  │ (ENHANCED)         │                  │
│  │ - Speaker/TTS      │  │ - Custodial model  │                  │
│  │ - Telegram Bot     │  │ - Sweeping         │                  │
│  │ - Zalo OA/ZNS      │  │ - Multi-sig        │                  │
│  │ - Email            │  └────────────────────┘                  │
│  │ - Webhook          │                                           │
│  └────────────────────┘  ┌────────────────────┐                  │
│                          │ Off-ramp Manager   │                  │
│  ┌────────────────────┐  │ (NEW!)             │                  │
│  │ Data Archival      │  │ - On-demand        │                  │
│  │ Service (NEW!)     │  │ - Scheduled        │                  │
│  │ - Hot→Cold         │  │ - Threshold-based  │                  │
│  │ - S3 Glacier       │  └────────────────────┘                  │
│  │ - Merkle tree      │                                           │
│  └────────────────────┘  ┌────────────────────────────────────┐  │
│                          │     Ledger Service                  │  │
│                          │     - Double-entry accounting       │  │
│                          │     - Transaction hashing (NEW!)    │  │
│                          └────────────────────────────────────┘  │
│                                                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │              AML Engine (Self-built)                         │ │
│  │  - Customer risk scoring  - Wallet screening                │ │
│  │  - Transaction monitoring - Sanctions lists (OFAC, UN, EU)  │ │
│  └─────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
                                │
┌───────────────────────────────▼──────────────────────────────────┐
│                    BLOCKCHAIN LAYER                               │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │          Multi-Chain Listener Orchestrator                │   │
│  ├──────────────────────────────────────────────────────────┤   │
│  │  ┌───────┐  ┌────────┐  ┌──────┐  ┌────────┐  ┌──────┐ │   │
│  │  │ TRON  │  │Solana  │  │ BSC  │  │Polygon │  │ ETH  │ │   │
│  │  │Listener│ │Listener│  │Listnr│  │Listener│  │(P2)  │ │   │
│  │  └───┬───┘  └───┬────┘  └──┬───┘  └───┬────┘  └──┬───┘ │   │
│  │      └──────────┴───────────┴──────────┴───────────┘     │   │
│  │                          ↓                                │   │
│  │              Transaction Validator                        │   │
│  │              - Verify finality                            │   │
│  │              - Parse memo/reference                       │   │
│  │              - AML wallet screening                       │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Wallet Service (Custodial)                   │   │
│  │                                                            │   │
│  │  Hot Wallets (per chain):                                │   │
│  │  - TRON: TYs7...   (Receive payments)                    │   │
│  │  - Solana: 8xK7... (Receive payments)                    │   │
│  │  - BSC: 0xAb12...  (Receive payments)                    │   │
│  │                                                            │   │
│  │  Cold Wallet (Multi-sig 2-of-3):                         │   │
│  │  - Main treasury storage                                  │   │
│  │  - Sweeping Worker moves funds (every 6 hours)           │   │
│  └──────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
                                │
┌───────────────────────────────▼──────────────────────────────────┐
│                         DATA LAYER                                │
│                                                                    │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐ │
│  │ PostgreSQL 15    │  │ Redis 7          │  │ S3 / MinIO     │ │
│  │ (HOT DATA)       │  │                  │  │                │ │
│  │                  │  │ - Rate limiting  │  │ - KYC docs     │ │
│  │ - Merchants      │  │ - Session cache  │  │ - Audit files  │ │
│  │ - Payments       │  │ - KYC cache      │  │                │ │
│  │ - Payouts        │  │ - Wallet→User    │  └────────────────┘ │
│  │ - Ledger         │  │   mapping cache  │                     │
│  │ - AML tables     │  │                  │  ┌────────────────┐ │
│  │ - Identity map   │  └──────────────────┘  │ S3 Glacier     │ │
│  │ - Txn hashes     │                        │ (COLD DATA)    │ │
│  │                  │                        │                │ │
│  │ 0-12 months data │                        │ - Archived     │ │
│  └──────────────────┘                        │   payments     │ │
│                                               │ - Old ledger   │ │
│                                               │ - 1+ years     │ │
│                                               └────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

### 4.2. New Database Tables (PRD v2.2)

```sql
-- 1. Wallet Identity Mappings (already shown above)
CREATE TABLE wallet_identity_mappings (...);

-- 2. KYC Sessions (for tracking KYC attempts)
CREATE TABLE kyc_sessions (
    id UUID PRIMARY KEY,
    wallet_address VARCHAR(255) NOT NULL,
    blockchain VARCHAR(20) NOT NULL,

    -- Session status
    status VARCHAR(20) DEFAULT 'initiated', -- initiated, documents_uploaded, verifying, completed, failed

    -- User info (collected during KYC)
    full_name VARCHAR(255),
    id_type VARCHAR(20),
    id_number VARCHAR(100),
    date_of_birth DATE,
    nationality VARCHAR(10),

    -- Document uploads
    id_front_url VARCHAR(500),
    id_back_url VARCHAR(500),
    selfie_url VARCHAR(500),
    liveness_video_url VARCHAR(500),

    -- Verification results
    liveness_score DECIMAL(3, 2), -- 0.00-1.00
    face_match_score DECIMAL(3, 2),
    id_verification_score DECIMAL(3, 2),

    -- Provider info
    provider VARCHAR(50), -- 'sumsub', 'onfido', 'manual'
    provider_session_id VARCHAR(255),
    provider_response JSONB,

    -- Failure reason
    failure_reason TEXT,

    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    expires_at TIMESTAMP -- Session expires after 30 minutes
);

-- 3. Notification Log (track all notifications sent)
CREATE TABLE notification_logs (
    id UUID PRIMARY KEY,

    merchant_id UUID REFERENCES merchants(id),
    payment_id UUID REFERENCES payments(id),

    notification_type VARCHAR(50), -- 'payment_received', 'payout_approved', etc.
    channel VARCHAR(20), -- 'speaker', 'telegram', 'zalo', 'email', 'webhook'

    -- Status
    status VARCHAR(20) DEFAULT 'queued', -- queued, sent, delivered, failed

    -- Payload
    payload JSONB,

    -- Delivery info
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    failed_at TIMESTAMP,
    failure_reason TEXT,

    -- Retry info
    retry_count INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notif_log_merchant ON notification_logs(merchant_id);
CREATE INDEX idx_notif_log_payment ON notification_logs(payment_id);
CREATE INDEX idx_notif_log_status ON notification_logs(status);

-- 4. Payout Schedules (for scheduled/threshold withdrawals)
CREATE TABLE payout_schedules (
    id UUID PRIMARY KEY,
    merchant_id UUID UNIQUE REFERENCES merchants(id),

    -- Scheduled withdrawal
    scheduled_enabled BOOLEAN DEFAULT FALSE,
    scheduled_frequency VARCHAR(20), -- 'weekly', 'monthly'
    scheduled_day_of_week INTEGER, -- 0=Sunday, 6=Saturday (for weekly)
    scheduled_day_of_month INTEGER, -- 1-31 (for monthly)
    scheduled_time TIME, -- e.g., '16:00:00'
    scheduled_withdraw_percentage INTEGER DEFAULT 80, -- Withdraw 80% of balance

    -- Threshold withdrawal
    threshold_enabled BOOLEAN DEFAULT FALSE,
    threshold_usdt DECIMAL(20, 8), -- e.g., 5000.00 USDT
    threshold_withdraw_percentage INTEGER DEFAULT 90,

    -- Last execution
    last_scheduled_run_at TIMESTAMP,
    last_threshold_trigger_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 5. Archived Records (for data retention tracking)
CREATE TABLE archived_records (
    id UUID PRIMARY KEY,
    original_id UUID NOT NULL,
    table_name VARCHAR(50) NOT NULL, -- 'payments', 'ledger_entries', etc.

    -- Archive location
    archive_path VARCHAR(500) NOT NULL, -- S3 URL
    archive_format VARCHAR(20) DEFAULT 'json.gz',

    -- Integrity
    data_hash VARCHAR(64) NOT NULL, -- SHA-256 of original record

    -- Metadata
    archived_at TIMESTAMP DEFAULT NOW(),
    archive_size_bytes BIGINT,

    UNIQUE(table_name, original_id)
);

CREATE INDEX idx_archived_table ON archived_records(table_name);
CREATE INDEX idx_archived_date ON archived_records(archived_at);

-- 6. Merkle Roots (for batch verification)
CREATE TABLE merkle_roots (
    id UUID PRIMARY KEY,
    date DATE UNIQUE NOT NULL,
    root_hash VARCHAR(64) NOT NULL,
    transaction_count INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 7. Sweeping Logs (track hot wallet → cold wallet transfers)
CREATE TABLE sweeping_logs (
    id UUID PRIMARY KEY,

    blockchain VARCHAR(20) NOT NULL,
    from_address VARCHAR(255) NOT NULL, -- Hot wallet
    to_address VARCHAR(255) NOT NULL, -- Cold wallet

    amount_crypto DECIMAL(20, 8) NOT NULL,
    currency VARCHAR(10) NOT NULL, -- USDT, USDC

    tx_hash VARCHAR(255),
    status VARCHAR(20) DEFAULT 'pending', -- pending, confirmed, failed

    -- Reason
    trigger_reason VARCHAR(50), -- 'scheduled', 'threshold_exceeded', 'manual'
    threshold_value DECIMAL(20, 8), -- Threshold that triggered sweep

    -- Multi-sig info (if applicable)
    requires_multisig BOOLEAN DEFAULT FALSE,
    signers JSONB, -- Array of signer addresses
    signatures_collected INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW(),
    confirmed_at TIMESTAMP
);

CREATE INDEX idx_sweep_blockchain ON sweeping_logs(blockchain);
CREATE INDEX idx_sweep_status ON sweeping_logs(status);
```

---

## 5. LỘ TRÌNH THỰC THI (IMPLEMENTATION ROADMAP)

### Revised Timeline: 8-10 weeks (vs 4-6 weeks original MVP)

#### Week 1-2: Foundation + Identity Mapping

**Week 1**:
- [ ] Project setup (monorepo structure)
- [ ] Database schema (core + PRD v2.2 tables)
- [ ] Basic API structure
- [ ] Authentication system
- [ ] **NEW**: Create `wallet_identity_mappings` table
- [ ] **NEW**: Create `kyc_sessions` table

**Week 2**:
- [ ] **NEW**: Implement Identity Mapping Service
  - `CheckWalletKYC()` API endpoint
  - Redis caching for wallet→user lookup
- [ ] **NEW**: Integrate KYC provider (Sumsub/Onfido)
  - Face liveness detection
  - ID verification
- [ ] Update payment flow to check wallet mapping before KYC

**Deliverables**:
- Smart KYC system functional
- Returning users skip KYC automatically

---

#### Week 3-4: Core Payment + Multi-Chain

**Week 3**:
- [ ] Blockchain listener (Solana USDT/USDC)
- [ ] **NEW**: Add TRON listener (TRC20 USDT)
- [ ] **NEW**: Add BSC listener (BEP20 USDT/BUSD)
- [ ] Payment creation API
- [ ] QR code generation
- [ ] AML wallet screening integration

**Week 4**:
- [ ] Payment status tracking
- [ ] Webhook system
- [ ] Ledger system (double-entry)
- [ ] **NEW**: Transaction hashing service
- [ ] **NEW**: Merkle tree generation (daily job)

**Deliverables**:
- Multi-chain payment acceptance working
- Immutable transaction records

---

#### Week 5: Notification Center

**Week 5**:
- [ ] **NEW**: Build Notification Center architecture
  - Plugin-based dispatcher
  - Redis Queue for async delivery
- [ ] **NEW**: Implement channels:
  - [x] Telegram Bot (HIGH priority)
  - [x] Zalo OA/ZNS (HIGH priority)
  - [x] Email (SendGrid)
  - [ ] Webhook (existing)
  - [ ] Speaker/TTS (MEDIUM priority, can defer to Week 7)
- [ ] Notification logging & retry logic
- [ ] Admin dashboard for notification status

**Deliverables**:
- Merchants receive real-time notifications on Telegram/Zalo
- Email invoices sent automatically

---

#### Week 6: Treasury & Sweeping

**Week 6**:
- [ ] **NEW**: Custodial Treasury Service
  - Merchant balance tracking (USDT/USDC)
  - VND equivalent calculation
- [ ] **NEW**: Sweeping Worker
  - Auto-sweep hot wallets to cold storage
  - Threshold: $10k USD
  - Run every 6 hours
- [ ] **NEW**: Multi-sig wallet setup (Gnosis Safe or similar)
- [ ] **NEW**: `sweeping_logs` table
- [ ] Manual payout process (existing)

**Deliverables**:
- Secure custodial model operational
- Automatic sweeping working

---

#### Week 7: Advanced Off-ramp & Data Retention

**Week 7A: Off-ramp Strategies**:
- [ ] **NEW**: Payout Scheduler
  - Weekly/monthly scheduled withdrawals
  - `payout_schedules` table
- [ ] **NEW**: Threshold Monitor Worker
  - Auto-trigger payout when balance > threshold
- [ ] Merchant settings UI for schedule configuration

**Week 7B: Data Retention**:
- [ ] **NEW**: Data Archival Worker
  - Monthly job to move old data to S3 Glacier
  - Keep 0-12 months in PostgreSQL
- [ ] **NEW**: Restore service (for compliance requests)
- [ ] **NEW**: `archived_records` table

**Deliverables**:
- Flexible off-ramp options for merchants
- Infinite data retention system operational

---

#### Week 8: Admin Panel & Polish

**Week 8**:
- [ ] Admin panel enhancements:
  - KYC review dashboard (with face liveness results)
  - Payout approval queue
  - Notification logs viewer
  - Sweeping logs viewer
- [ ] Merchant dashboard:
  - Balance display (USDT + VND)
  - Payout schedule settings
  - Notification preferences
  - Transaction history with archive search
- [ ] **NEW**: Speaker/TTS integration (if not done in Week 5)
  - WebSocket setup
  - POS app integration guide

**Deliverables**:
- Complete admin & merchant dashboards
- Full notification system live

---

#### Week 9: Testing & Security Audit

**Week 9**:
- [ ] End-to-end testing on testnet:
  - Multi-chain payment flow
  - Identity mapping (new wallet + returning)
  - Notification delivery (all channels)
  - Sweeping worker
  - Data archival & restore
- [ ] Load testing (100+ concurrent payments)
- [ ] Security audit:
  - Multi-sig wallet security
  - KYC data encryption
  - Transaction hash integrity
  - AML screening
- [ ] Fix bugs & optimize performance

**Deliverables**:
- System passes all tests
- Security audit report clean

---

#### Week 10: Deployment & Pilot Launch

**Week 10**:
- [ ] Deploy to production:
  - Database migrations
  - Environment setup (TRON/Solana/BSC RPC nodes)
  - S3 Glacier bucket creation
  - Redis cluster for caching
- [ ] Set up monitoring:
  - Sweeping worker alerts
  - Notification delivery SLA
  - Data archival job success
- [ ] Onboard 3-5 pilot merchants:
  - Help configure Telegram/Zalo
  - Test payment flow end-to-end
  - Train on admin panel
- [ ] Documentation:
  - Merchant onboarding guide
  - API documentation
  - Ops runbook (KYC review, payout approval, sweeping management)

**Deliverables**:
- Production system live
- Pilot merchants processing real payments
- PRD v2.2 fully implemented

---

## 6. SUCCESS CRITERIA (PRD v2.2)

### Technical KPIs

| Metric | Target | Measurement |
|--------|--------|-------------|
| Payment Success Rate | > 98% | Confirmed / Total payments |
| Multi-chain Support | 3+ chains (TRON, Solana, BSC) | Active listeners |
| KYC Recognition Rate | > 95% | Returning users / Total users |
| Notification Delivery | > 95% (all channels) | Delivered / Sent |
| Speaker Latency | < 3 seconds | Payment confirmed → Audio played |
| Average Confirmation Time | < 30 seconds | Blockchain finality |
| System Uptime | > 99.5% | Monitored by Pingdom |
| Sweeping Success Rate | 100% | Successful sweeps / Attempted |
| Data Integrity | 100% | Hash verification pass rate |
| Archive Success Rate | 100% | Records archived / Total old records |

### Business KPIs

| Metric | Target (Month 1) | Target (Month 3) |
|--------|------------------|------------------|
| Pilot Merchants | 5 | 20 |
| Total Transactions | 100+ | 1,000+ |
| Total Volume | 1B+ VND ($40k USD) | 10B+ VND ($400k USD) |
| Revenue | 10M+ VND | 100M+ VND |
| Merchant NPS | > 30 | > 50 |
| User NPS (Payers) | > 40 | > 60 |

### Compliance KPIs

| Metric | Target | Frequency |
|--------|--------|-----------|
| KYC Completion Rate | > 90% | Weekly |
| AML Alerts Resolution Time | < 24 hours (HIGH severity) | Daily |
| SAR Filed (if any) | Within 12 hours of detection | As needed |
| Data Retention Compliance | 100% (infinite storage) | Monthly audit |
| Transaction Hash Integrity | 100% verification pass | Daily |
| Sanctions Screening Coverage | 100% of payments | Real-time |

### User Experience KPIs

| Metric | Target | Measurement |
|--------|--------|-------------|
| Returning User KYC Skip Rate | > 95% | Users skipping KYC / Total returning |
| First Payment Time (New User) | < 3 minutes (with KYC) | Payment created → Confirmed |
| First Payment Time (Returning) | < 30 seconds (no KYC) | Payment created → Confirmed |
| Notification Delivery Time | < 5 seconds | Payment confirmed → Notification received |
| Merchant Satisfaction (Notifications) | > 90% positive feedback | Survey |

---

## 7. RỦI RO & GIẢM THIỂU (RISKS & MITIGATION)

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Multi-chain complexity** | 🔴 HIGH | 🟡 MEDIUM | Start with 2 chains (TRON + Solana), add BSC in Week 4 |
| **KYC provider downtime** | 🟡 MEDIUM | 🟢 LOW | Implement fallback to manual KYC review |
| **Sweeping worker failure** | 🔴 HIGH | 🟢 LOW | Alerting + manual backup process, multi-sig requires 2 approvals |
| **S3 Glacier retrieval delay** | 🟢 LOW | 🟡 MEDIUM | Use S3 Glacier (not Deep Archive) for faster restore (1-5 hours) |
| **Notification delivery failure** | 🟡 MEDIUM | 🟡 MEDIUM | Retry logic (3 attempts), fallback to email if primary fails |
| **Transaction hash collision** | 🟢 LOW | ⚪ VERY LOW | Use SHA-256 (cryptographically secure) + previous hash in chain |

### Business Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Custodial model liability** | 🔴 HIGH | 🟡 MEDIUM | Consult lawyer, get insurance, clear T&C disclosure |
| **Regulatory change** | 🟡 MEDIUM | 🟡 MEDIUM | Monitor Đà Nẵng sandbox updates, maintain compliance docs |
| **OTC partner issues** | 🟡 MEDIUM | 🟢 LOW | Have 2-3 backup OTC partners |
| **Merchant adoption slow** | 🟡 MEDIUM | 🟡 MEDIUM | Focus on tourism sector, offer onboarding support |
| **User reluctance to KYC** | 🟡 MEDIUM | 🟡 MEDIUM | Emphasize "one-time only", show benefits (faster payments later) |

### Security Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Hot wallet compromise** | 🔴 HIGH | 🟢 LOW | Keep balance < $10k, sweep every 6 hours, multi-sig cold wallet |
| **KYC data breach** | 🔴 HIGH | 🟢 LOW | Encrypt at rest (AES-256), access control, audit logs |
| **Transaction tampering** | 🟡 MEDIUM | ⚪ VERY LOW | Hash chain + Merkle trees, daily verification |
| **Notification spam attack** | 🟢 LOW | 🟡 MEDIUM | Rate limiting on notification endpoints |

---

## 8. DEPENDENCIES & THIRD-PARTY SERVICES

### Critical Dependencies

| Service | Purpose | Cost | Alternatives |
|---------|---------|------|--------------|
| **Sumsub** | KYC & Face Liveness | $0.50/check | Onfido ($1/check), Self-hosted |
| **Telegram Bot API** | Real-time notifications | FREE | N/A (no alternative) |
| **Zalo OA/ZNS** | Vietnam notifications | $0.01/msg (ZNS) | SMS ($0.05/msg) |
| **SendGrid** | Email delivery | $15/mo (40k emails) | AWS SES ($0.10/1k), SMTP |
| **AWS S3 Glacier** | Long-term storage | $1/TB/month | Google Cloud Storage Coldline ($4/TB) |
| **Chainalysis Sanctions Oracle** | Wallet screening | FREE (on-chain) | TRM Labs ($500/mo), Elliptic ($1k/mo) |
| **TRON Full Node** | TRON blockchain data | $50/mo (VPS) | TronGrid API (free tier: 15k req/day) |
| **Solana RPC** | Solana blockchain data | $50/mo (Helius) | QuickNode ($49/mo), public RPC (rate limited) |
| **BSC RPC** | BSC blockchain data | FREE (public) | QuickNode ($49/mo) for higher rate limits |
| **Redis Cloud** | Caching & queues | $30/mo (5GB) | Self-hosted Redis on VPS |
| **PostgreSQL** | Primary database | Included in VPS | AWS RDS ($50/mo) |

**Total Monthly Cost** (MVP): ~$200-250/month

---

## 9. APPENDIX

### A. Supported Chains & Tokens (Launch)

| Chain | Token | Contract Address (Mainnet) | Decimals | Priority |
|-------|-------|----------------------------|----------|----------|
| **TRON** | USDT (TRC20) | TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t | 6 | 🔴 HIGH |
| **Solana** | USDT (SPL) | Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB | 6 | 🔴 HIGH |
| **Solana** | USDC (SPL) | EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v | 6 | 🔴 HIGH |
| **BSC** | USDT (BEP20) | 0x55d398326f99059fF775485246999027B3197955 | 18 | 🟡 MEDIUM |
| **BSC** | BUSD (BEP20) | 0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56 | 18 | 🟡 MEDIUM |
| **Polygon** | USDT | 0xc2132D05D31c914a87C6611C10748AEb04B58e8F | 6 | 🟢 LOW (Phase 2) |
| **Ethereum** | USDT | 0xdAC17F958D2ee523a2206206994597C13D831ec7 | 6 | ⚪ FUTURE (Phase 3) |

### B. Notification Templates

#### Telegram Message (Payment Received):
```
💰 Giao dịch mới

Số tiền: 10.50 USDT
Giá trị: 241,500 VND
Khách hàng: John Doe
Bàn/Phòng: #5
Thời gian: 19/11/2025 15:23

📊 Số dư hiện tại: 1,234.80 USDT (≈ 28.4M VND)

🔗 Chi tiết: https://dashboard.example.com/tx/abc123
```

#### Zalo ZNS Template:
```
[Tên Merchant] - Thanh toán mới

Số tiền: {{amount}} {{currency}}
Giá trị VND: {{amount_vnd}}
Thời gian: {{time}}

Số dư: {{balance}} USDT
```

#### Email Invoice (Payment Received):
```html
<!DOCTYPE html>
<html>
<head>
  <title>Payment Confirmation</title>
</head>
<body>
  <h2>Payment Received - Invoice #12345</h2>

  <table>
    <tr><td>Amount:</td><td>10.50 USDT</td></tr>
    <tr><td>VND Value:</td><td>241,500 VND</td></tr>
    <tr><td>Exchange Rate:</td><td>23,000 VND/USDT</td></tr>
    <tr><td>Transaction Hash:</td><td>5j7kL9...</td></tr>
    <tr><td>Time:</td><td>2025-11-19 15:23:45 UTC+7</td></tr>
  </table>

  <p>Thank you for your payment!</p>

  <a href="https://pay.example.com/receipt/12345">View Receipt</a>
</body>
</html>
```

#### Speaker/TTS Script (Vietnamese):
```
Đã nhận mười phẩy năm USDT từ khách John Doe.
```

### C. Glossary

| Term | Definition |
|------|------------|
| **Custodial** | Hệ thống giữ tiền thay cho user/merchant |
| **Sweeping** | Tự động chuyển tiền từ hot wallet sang cold wallet |
| **Multi-sig** | Ví cần nhiều chữ ký (>1 người) để giao dịch |
| **MPC** | Multi-Party Computation - chia private key thành nhiều phần |
| **KYC** | Know Your Customer - xác minh danh tính |
| **AML** | Anti-Money Laundering - chống rửa tiền |
| **SAR** | Suspicious Activity Report - báo cáo giao dịch khả nghi |
| **Off-ramp** | Chuyển crypto → fiat (VND) |
| **On-ramp** | Chuyển fiat → crypto |
| **Finality** | Giao dịch blockchain đã không thể đảo ngược |
| **Liveness** | Kiểm tra người thật (không phải ảnh/video giả) |
| **Merkle Tree** | Cấu trúc dữ liệu cho phép verify nhiều hash cùng lúc |
| **Hash Chain** | Mỗi hash tham chiếu hash trước đó (immutable) |
| **Hot Data** | Dữ liệu truy cập thường xuyên |
| **Cold Data** | Dữ liệu lưu trữ lâu dài, ít truy cập |

---

## 10. CONCLUSION & NEXT STEPS

### Summary

PRD v2.2 nâng cấp hệ thống thanh toán stablecoin với **5 tính năng then chốt**:

1. ✅ **Smart Identity Mapping**: KYC một lần, nhận diện vĩnh viễn
2. ✅ **Omni-channel Notifications**: Thông báo đa kênh (Telegram, Zalo, TTS, Email)
3. ✅ **Custodial Treasury**: Giữ tiền an toàn với sweeping + multi-sig
4. ✅ **Advanced Off-ramp**: Rút tiền linh hoạt (on-demand, scheduled, threshold)
5. ✅ **Infinite Data Retention**: Lưu trữ vô hạn với S3 Glacier + transaction hashing

### Implementation Priority

**Phase 1 (Critical)**: Identity Mapping, Multi-chain, Notification Center
**Phase 2 (Important)**: Treasury Sweeping, Data Retention
**Phase 3 (Nice-to-have)**: Advanced Off-ramp, Speaker TTS

### Development Timeline

- **Original MVP**: 4-6 weeks
- **PRD v2.2**: **8-10 weeks** (60% increase)
- **Estimated Cost**: $30k-50k (development) + $200-250/month (infrastructure)

### Success Definition

System is successful if:
- ✅ 5+ merchants processing real payments
- ✅ All 3 chains (TRON, Solana, BSC) operational
- ✅ Returning users skip KYC (>95% rate)
- ✅ Notifications delivered reliably (>95% success)
- ✅ Zero security incidents
- ✅ 100% data integrity (hash verification)

### Go/No-Go Decision

Before implementation, confirm:
- [ ] Legal approval for custodial model
- [ ] Insurance for holding merchant funds
- [ ] OTC partner contracts signed
- [ ] 2-3 pilot merchants committed
- [ ] 6+ months runway (budget)
- [ ] Technical team assembled (5-6 people)

---

**Document Status**: ✅ APPROVED
**Next Review**: 2025-12-01
**Owner**: Product Team
**Last Updated**: 2025-11-19

---

**END OF PRD v2.2**
