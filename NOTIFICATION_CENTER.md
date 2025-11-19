# Omni-channel Notification Center

**Project**: Stablecoin Payment Gateway - Notification Module
**Last Updated**: 2025-11-19
**Status**: Design Phase (PRD v2.2)

---

## 🎯 Overview

The **Notification Center** provides multi-channel, plugin-based notification delivery to ensure merchants never miss a payment.

### Philosophy

**"Bủa vây" (Surround)** merchants with notifications across all channels:
- 🔊 **Speaker/TTS**: Immediate audible alert at POS counter
- 📱 **Telegram Bot**: Real-time push to boss's phone
- 💬 **Zalo OA/ZNS**: Vietnam-specific messaging (king of VN market)
- 📧 **Email**: Invoice and monthly statements
- 🔗 **Webhook**: Integration with merchant POS/ERP systems

---

## 🏗️ Architecture

### Plugin-Based Design

```typescript
interface NotificationPlugin {
  name: string
  priority: number // 1=highest, 5=lowest

  send(message: NotificationMessage): Promise<SendResult>
  validateConfig(config: any): boolean
  getStatus(): PluginStatus
}

class NotificationDispatcher {
  private plugins: Map<string, NotificationPlugin> = new Map()

  registerPlugin(plugin: NotificationPlugin) {
    this.plugins.set(plugin.name, plugin)
  }

  async notify(message: NotificationMessage, channels: string[]) {
    const promises = channels.map(channel => {
      const plugin = this.plugins.get(channel)
      return plugin?.send(message)
    })

    return Promise.allSettled(promises)
  }
}
```

### Queue System

Use **Bull** (Redis-based queue) for reliable async delivery:

```typescript
import Queue from 'bull'

const notificationQueue = new Queue('notifications', {
  redis: { host: 'localhost', port: 6379 }
})

// Enqueue notification
await notificationQueue.add('send', {
  message: { type: 'payment_received', data: {...} },
  channels: ['speaker', 'telegram', 'zalo', 'email']
}, {
  attempts: 3,
  backoff: { type: 'exponential', delay: 2000 }
})

// Process queue
notificationQueue.process('send', async (job) => {
  const dispatcher = new NotificationDispatcher()
  // Register plugins...
  await dispatcher.notify(job.data.message, job.data.channels)
})
```

---

## 📡 Channel Implementations

### 1. Speaker/TTS (Loa thông báo)

**Use Case**: Thu ngân nghe ngay tại quầy

**Tech Stack**:
- WebSocket (Socket.io) for real-time delivery
- Google Cloud TTS API or Browser `speechSynthesis`

**Implementation**:
```typescript
class SpeakerPlugin implements NotificationPlugin {
  name = 'speaker'
  priority = 1

  async send(message: NotificationMessage) {
    const text = `Đã nhận ${message.data.amount} ${message.data.currency}`
    const audioBuffer = await this.textToSpeech(text, 'vi-VN')

    // Send via WebSocket to POS device
    this.socketServer.emit('play-audio', {
      merchantId: message.recipient.merchantId,
      audio: audioBuffer
    })

    return { success: true, channel: 'speaker' }
  }

  private async textToSpeech(text: string, lang: string): Promise<Buffer> {
    // Google Cloud TTS
    const [response] = await ttsClient.synthesizeSpeech({
      input: { text },
      voice: { languageCode: lang, name: 'vi-VN-Standard-A' },
      audioConfig: { audioEncoding: 'MP3' }
    })

    return response.audioContent
  }
}
```

**Frontend (Merchant POS)**:
```javascript
const socket = io('wss://gateway.example.com')

socket.on('play-audio', async (data) => {
  const audio = new Audio(data.audio)
  await audio.play()

  toast.success(`💰 ${data.amount} ${data.currency}`)
})
```

---

### 2. Telegram Bot

**Setup**:
1. Create bot via @BotFather → Get token
2. Merchant sends `/start` to bot
3. Bot saves `chat_id` to database

**Implementation**:
```typescript
import TelegramBot from 'node-telegram-bot-api'

class TelegramPlugin implements NotificationPlugin {
  name = 'telegram'
  priority = 2
  private bot: TelegramBot

  constructor(token: string) {
    this.bot = new TelegramBot(token, { polling: true })
    this.setupCommands()
  }

  async send(message: NotificationMessage) {
    const chatId = message.recipient.telegram_chat_id
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

📊 Số dư: ${msg.data.balance} USDT
      `.trim()
    }
  }

  private setupCommands() {
    this.bot.onText(/\/start/, async (msg) => {
      const chatId = msg.chat.id
      await this.saveChatId(chatId)
      await this.bot.sendMessage(chatId, 'Bot đã kích hoạt!')
    })

    this.bot.onText(/\/balance/, async (msg) => {
      const balance = await this.getMerchantBalance(msg.chat.id)
      await this.bot.sendMessage(msg.chat.id, `Số dư: ${balance} USDT`)
    })
  }
}
```

---

### 3. Zalo OA/ZNS

**Options**:
- **Zalo OA (Official Account)**: Free, needs approval
- **Zalo ZNS (Notification Service)**: Template-based, $0.01/msg

**Setup**:
1. Register at https://oa.zalo.me/
2. Create message templates
3. Get OA ID & access token

**Implementation**:
```typescript
class ZaloPlugin implements NotificationPlugin {
  name = 'zalo'
  priority = 2

  async send(message: NotificationMessage) {
    // Use Zalo ZNS template API
    const response = await axios.post(
      'https://business.openapi.zalo.me/message/template',
      {
        phone: message.recipient.phone,
        template_id: '123456',
        template_data: {
          amount: message.data.amount,
          currency: message.data.currency,
          time: new Date().toISOString()
        }
      },
      { headers: { 'access_token': this.accessToken } }
    )

    return { success: response.data.error === 0 }
  }
}
```

**Zalo Template Example**:
```
Giao dịch mới - {{merchant_name}}

Số tiền: {{amount}} {{currency}}
Giá trị: {{amount_vnd}} VND
Thời gian: {{time}}

Số dư: {{balance}} USDT
```

---

### 4. Email

**Tech Stack**: SendGrid (100 emails/day free)

**Implementation**:
```typescript
import sendgrid from '@sendgrid/mail'

class EmailPlugin implements NotificationPlugin {
  name = 'email'
  priority = 4

  async send(message: NotificationMessage) {
    const html = this.generateHTML(message)

    await sendgrid.send({
      to: message.recipient.email,
      from: 'noreply@gateway.example.com',
      subject: this.getSubject(message.type),
      html,
      attachments: message.data.attachments || []
    })

    return { success: true }
  }

  private generateHTML(msg: NotificationMessage): string {
    if (msg.type === 'payment_received') {
      return `
        <h2>Payment Received</h2>
        <table>
          <tr><td>Amount:</td><td>${msg.data.amount} ${msg.data.currency}</td></tr>
          <tr><td>VND Value:</td><td>${msg.data.amount_vnd} VND</td></tr>
          <tr><td>Transaction ID:</td><td>${msg.data.transaction_id}</td></tr>
        </table>
        <a href="https://dashboard.example.com/tx/${msg.data.payment_id}">View Details</a>
      `
    }
  }
}
```

---

### 5. Webhook

**Implementation**:
```typescript
class WebhookPlugin implements NotificationPlugin {
  name = 'webhook'
  priority = 3

  async send(message: NotificationMessage) {
    const payload = {
      event: message.type,
      data: message.data,
      timestamp: new Date().toISOString()
    }

    // Sign with HMAC
    const signature = crypto
      .createHmac('sha256', message.recipient.webhook_secret)
      .update(JSON.stringify(payload))
      .digest('hex')

    // Send with retry
    return this.sendWithRetry(
      message.recipient.webhook_url,
      payload,
      signature,
      maxRetries: 3
    )
  }

  private async sendWithRetry(url, payload, signature, maxRetries) {
    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        const res = await axios.post(url, payload, {
          headers: { 'X-Webhook-Signature': signature },
          timeout: 5000
        })

        if (res.status === 200) return { success: true }
      } catch (error) {
        if (attempt === maxRetries) {
          return { success: false, error: error.message }
        }
        await this.sleep(2 ** attempt * 1000) // Exponential backoff
      }
    }
  }
}
```

---

## 🗄️ Database Schema

```sql
CREATE TABLE notification_logs (
    id UUID PRIMARY KEY,
    merchant_id UUID REFERENCES merchants(id),
    payment_id UUID REFERENCES payments(id),

    notification_type VARCHAR(50), -- 'payment_received', 'payout_approved'
    channel VARCHAR(20), -- 'speaker', 'telegram', 'zalo', 'email', 'webhook'

    status VARCHAR(20) DEFAULT 'queued', -- queued, sent, delivered, failed

    payload JSONB,

    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    failed_at TIMESTAMP,
    failure_reason TEXT,

    retry_count INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notif_log_merchant ON notification_logs(merchant_id);
CREATE INDEX idx_notif_log_status ON notification_logs(status);
CREATE INDEX idx_notif_log_channel ON notification_logs(channel);
```

---

## 📊 Monitoring

### Delivery SLA

| Channel | Target Delivery Time | Retry Count | Priority |
|---------|---------------------|-------------|----------|
| **Speaker** | < 1 second | 0 (fail fast) | 🔴 CRITICAL |
| **Telegram** | < 5 seconds | 3 | 🟡 HIGH |
| **Zalo** | < 10 seconds | 3 | 🟡 HIGH |
| **Email** | < 30 seconds | 3 | 🟢 MEDIUM |
| **Webhook** | < 10 seconds | 5 | 🟢 MEDIUM |

### KPIs

```sql
-- Daily notification stats
SELECT
    channel,
    COUNT(*) as total,
    COUNT(CASE WHEN status = 'delivered' THEN 1 END) as delivered,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
    ROUND(100.0 * COUNT(CASE WHEN status = 'delivered' THEN 1 END) / COUNT(*), 2) as delivery_rate
FROM notification_logs
WHERE created_at >= NOW() - INTERVAL '1 day'
GROUP BY channel;
```

---

## ✅ Success Criteria

- [x] **Delivery Rate > 95%** for all channels
- [x] **Speaker latency < 3 seconds** (payment confirmed → audio played)
- [x] **Zero critical failures** (all channels have fallback)
- [x] **Merchant satisfaction > 90%** (from surveys)

---

**Document Status**: Design Phase
**Owner**: Backend Team
**Last Updated**: 2025-11-19
