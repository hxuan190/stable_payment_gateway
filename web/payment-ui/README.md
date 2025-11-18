# Payment UI - Payer Experience Layer

Next.js 14 frontend for the stablecoin payment gateway payer experience.

## Features

- 🎯 **Payment Status Page** - Real-time payment tracking
- 📱 **QR Code Display** - Scan to pay with crypto wallet
- ⏱️ **Countdown Timer** - 30-minute payment expiry
- 🔄 **WebSocket Updates** - Real-time status notifications
- ✅ **Success Page** - Payment receipt and confirmation
- 📱 **Mobile Responsive** - Optimized for all devices

## Tech Stack

- **Framework**: Next.js 14 (App Router)
- **Language**: TypeScript
- **Styling**: TailwindCSS
- **State Management**: React Hooks
- **Real-time**: WebSocket
- **QR Code**: qrcode.react

## Development

```bash
# Install dependencies
npm install

# Run development server
npm run dev

# Build for production
npm run build

# Start production server
npm start
```

## Environment Variables

Create a `.env.local` file:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080
```

## Project Structure

```
payment-ui/
├── app/
│   ├── order/
│   │   └── [id]/
│   │       ├── page.tsx          # Payment status page
│   │       └── success/
│   │           └── page.tsx      # Success page
│   ├── layout.tsx                # Root layout
│   └── globals.css               # Global styles
├── components/
│   ├── PaymentStatus.tsx         # Main payment status component
│   ├── QRCode.tsx                # QR code display
│   └── CountdownTimer.tsx        # Payment expiry timer
├── hooks/
│   └── usePaymentStatus.ts       # Custom hook for payment data
├── lib/
│   ├── api.ts                    # API client
│   └── websocket.ts              # WebSocket client
└── types/
    └── payment.ts                # TypeScript types
```

## API Integration

### Payment Status Endpoint

```
GET /api/v1/payments/{payment_id}/status
```

Response:
```json
{
  "data": {
    "id": "uuid",
    "status": "created|pending|confirming|completed|expired|failed",
    "amount_crypto": "100.00",
    "amount_vnd": "2300000",
    "currency": "USDT",
    "chain": "solana",
    "wallet_address": "...",
    "payment_memo": "...",
    "qr_code_data": "solana:...",
    "tx_hash": "...",
    "confirmations": 0,
    "expires_at": "2025-11-18T10:30:00Z",
    "created_at": "2025-11-18T10:00:00Z"
  },
  "error": null,
  "timestamp": "2025-11-18T10:00:00Z"
}
```

### WebSocket Events

```
ws://localhost:8080/ws/payments/{payment_id}
```

Events:
- `payment.pending` - Transaction detected
- `payment.confirming` - Awaiting confirmations
- `payment.completed` - Payment finalized
- `payment.expired` - Payment expired
- `payment.failed` - Payment failed

## Docker Deployment

```bash
# Build image
docker build -t payment-ui .

# Run container
docker run -p 3000:3000 \
  -e NEXT_PUBLIC_API_URL=http://api:8080 \
  -e NEXT_PUBLIC_WS_URL=ws://api:8080 \
  payment-ui
```

## Usage Flow

1. **User receives payment link**: `/order/{payment_id}`
2. **Page loads payment details** via API
3. **QR code displayed** for wallet scanning
4. **User sends crypto** from wallet
5. **WebSocket receives updates** as transaction confirms
6. **Auto-redirect to success page** when complete
7. **Receipt displayed** with transaction details

## Browser Support

- Chrome/Edge (latest)
- Firefox (latest)
- Safari (latest)
- Mobile browsers (iOS Safari, Chrome Mobile)

## Performance

- Lighthouse Score: 90+
- First Contentful Paint: < 1.5s
- Time to Interactive: < 3s
- WebSocket latency: < 500ms

## Security

- No sensitive data stored in frontend
- Payment IDs are public (non-sensitive)
- API calls use HTTPS in production
- WebSocket uses WSS in production
- CORS properly configured

## License

MIT
