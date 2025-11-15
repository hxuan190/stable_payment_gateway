# Tourism & Hospitality Use Cases - Da Nang

**Target**: Restaurants, Hotels, Tourist Services in Da Nang accepting crypto payments

---

## 🏨 Hotel Use Cases

### 1. Room Reservation Deposit

**Scenario**: International guest books room online, needs to pay deposit

**Flow**:
```
1. Guest books room on hotel website
2. Hotel system creates payment request:
   - Amount: 5,000,000 VND (2 nights deposit)
   - Reservation ID: RES-20251120-001
   - Guest email: john@example.com

3. System generates payment link + QR code
4. Send via email to guest
5. Guest opens email → clicks link → chooses crypto (USDT Solana)
6. Guest sends 217 USDT from Binance/Phantom wallet
7. Payment confirmed in 15 seconds
8. Hotel receives notification → reservation confirmed
9. Hotel settles to VND next day
```

**Benefits**:
- ✅ Guest doesn't need credit card
- ✅ No chargeback risk (crypto is final)
- ✅ Lower fee than booking.com (1% vs 15-20%)
- ✅ Instant confirmation (vs 1-3 days bank transfer)

**API Integration**:
```go
POST /api/v1/hotels/reservations
{
  "merchantId": "hotel_beach_resort",
  "reservationId": "RES-20251120-001",
  "guestName": "John Doe",
  "guestEmail": "john@example.com",
  "roomType": "Deluxe Ocean View",
  "checkIn": "2025-11-20",
  "checkOut": "2025-11-22",
  "depositVND": 5000000,
  "metadata": {
    "roomNumber": "302",
    "adults": 2,
    "children": 0
  }
}

Response:
{
  "paymentId": "pay_hotel_xxx",
  "paymentUrl": "https://pay.gateway.com/pay_hotel_xxx",
  "qrCode": "data:image/png;base64,...",
  "expiresAt": "2025-11-19T23:59:59Z",
  "acceptedTokens": [
    {
      "chain": "solana",
      "token": "USDT",
      "amount": "217.39",
      "wallet": "8xK7JVq...",
      "qrCode": "..."
    },
    {
      "chain": "bsc",
      "token": "USDT",
      "amount": "217.39",
      "wallet": "0xABC...",
      "qrCode": "..."
    }
  ]
}
```

---

### 2. Hotel Check-In Payment (Walk-in)

**Scenario**: Guest arrives without reservation, pays at reception

**Flow**:
```
1. Receptionist: "Total for 2 nights is 10,000,000 VND"
2. Guest: "Can I pay with crypto?"
3. Receptionist creates payment on tablet/POS
4. Show QR code on tablet screen
5. Guest scans with phone → sends crypto
6. Payment confirmed → print receipt
7. Guest gets room key
```

**Hardware Setup**:
- iPad/Android tablet at reception desk
- Payment app (responsive web or native app)
- WiFi connection
- Thermal printer (optional, for receipt)

**Features**:
- Large QR code display (easy to scan)
- Real-time confirmation status
- Print receipt with payment details
- Link to hotel PMS (Property Management System)

---

### 3. Hotel Extras & Minibar

**Scenario**: Guest wants to pay for spa, restaurant, minibar

**Flow**:
```
1. Guest finishes spa treatment
2. Staff: "That will be 1,500,000 VND"
3. Staff shows QR code (on phone or printed card)
4. Guest scans → pays
5. Charge automatically added to guest folio
6. At checkout, guest sees all crypto payments itemized
```

**Benefits**:
- No need to carry room key for charging
- Real-time payment tracking
- Automatic reconciliation
- Multi-currency support (show USD equivalent)

---

## 🍜 Restaurant Use Cases

### 1. Bill Payment (Dine-in)

**Scenario**: Customer finishes meal at beachfront restaurant

**Flow**:
```
1. Customer: "Can I have the bill?"
2. Waiter brings bill with QR code printed at bottom
3. Customer scans QR with phone
4. Selects crypto (USDT on Solana)
5. Sends payment
6. Waiter's tablet shows "Payment received ✓"
7. Customer leaves (no waiting for card machine)
```

**Bill Format**:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
        BEACHSIDE RESTAURANT
         Da Nang, Vietnam
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Table: 12                Date: 2025-11-15
Server: Linh                Time: 19:35

2x Phở bò                     160,000 VND
2x Bánh xèo                   140,000 VND
1x Grilled fish               350,000 VND
2x Bia Sài Gòn                 70,000 VND

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Subtotal:                     720,000 VND
Service (10%):                 72,000 VND
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
TOTAL:                        792,000 VND
                           ≈ 34.43 USDT

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
      PAY WITH CRYPTO

      [QR CODE]

Scan to pay with:
- USDT (Solana) - 34.43 USDT
- USDT (BSC) - 34.43 USDT
- USDC (Solana) - 34.43 USDC

Or visit:
pay.gateway.com/pay_rest_xxx

Bill ID: BILL-20251115-012
Expires: 15 minutes

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   Thank you! Come again! 🌴
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Thermal Printer Integration**:
```go
// Print bill with QR code
func PrintBill(payment Payment, items []BillItem) {
    printer := escpos.New("/dev/usb/lp0")

    printer.SetFontSize(2, 2)
    printer.Write("BEACHSIDE RESTAURANT\n")
    printer.SetFontSize(1, 1)

    // Print items
    for _, item := range items {
        printer.Write(fmt.Sprintf("%dx %-20s %10s\n",
            item.Qty, item.Name, formatVND(item.Price)))
    }

    // Print QR code
    printer.WriteQRCode(payment.PaymentURL)

    printer.Cut()
    printer.End()
}
```

---

### 2. Takeaway Orders

**Scenario**: Customer orders online for pickup

**Flow**:
```
1. Customer browses menu on restaurant website
2. Adds items to cart
3. Checkout → enters phone number
4. Selects "Pay with Crypto"
5. Shows QR code on screen
6. Customer pays immediately
7. Restaurant receives order + payment confirmation
8. Prepares food
9. Customer picks up (shows payment confirmation)
```

**Integration with Ordering Systems**:
```go
// Webhook from ordering system → create payment
POST /api/v1/restaurants/orders
{
  "merchantId": "restaurant_beachside",
  "orderId": "ORDER-20251115-042",
  "customerPhone": "+84901234567",
  "items": [
    {"name": "Phở bò", "qty": 2, "price": 80000},
    {"name": "Bánh mì", "qty": 3, "price": 25000}
  ],
  "totalVND": 235000,
  "pickupTime": "2025-11-15T20:00:00Z"
}

// Send payment link via SMS
→ "Your order #042 is confirmed. Pay here: pay.gateway.com/pay_xxx"
```

---

### 3. Tourist Group Payments

**Scenario**: Tour guide brings group of 20 tourists to restaurant

**Flow**:
```
1. Tour guide pre-orders set menu for group
2. Restaurant creates single payment for entire group
3. Tour guide pays with crypto (from tour company wallet)
4. Group enjoys meal
5. Restaurant settles to VND
```

**Benefits for Tour Companies**:
- Pay in crypto (avoid carrying large amounts of cash)
- Lower fees than corporate credit cards
- Instant confirmation
- Automatic expense tracking

---

## 🛍️ Tourist Services

### 1. Water Sports Rental

**Scenario**: Tourist wants to rent jet ski at beach

**Flow**:
```
1. Tourist: "How much for 30 minutes?"
2. Operator: "500,000 VND"
3. Operator shows laminated QR code card
4. Tourist scans → pays
5. Operator verifies payment on phone
6. Tourist gets life jacket + jet ski
```

**QR Code Card** (waterproof laminated):
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━
   MY KHE WATER SPORTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━

[LARGE QR CODE]

Scan to pay:
✓ Jet Ski: 500,000 VND
✓ Parasailing: 800,000 VND
✓ Banana Boat: 300,000 VND

Payment link:
pay.gateway.com/mykhe
━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Dynamic Pricing**:
```go
// Payment link with amount selector
GET /pay/mykhe
→ Shows menu:
  - Jet Ski (30 min): 500k VND = 21.74 USDT
  - Parasailing: 800k VND = 34.78 USDT
  - Banana Boat: 300k VND = 13.04 USDT

Customer selects → generates specific payment
```

---

### 2. Spa & Massage

**Scenario**: Tourist books 90-minute massage

**Flow**:
```
1. Reception: "90-minute oil massage is 1,200,000 VND"
2. Create payment on tablet
3. Customer pays before treatment
4. Treatment starts
5. After treatment, optional tip via QR code
```

**Tip Feature**:
```go
// After main payment, show tip options
Payment ID: pay_spa_xxx (PAID ✓)

━━━━━━━━━━━━━━━━━━━━━━━
   Add a tip? (Optional)
━━━━━━━━━━━━━━━━━━━━━━━

[ 10% - 120k VND ]
[ 15% - 180k VND ]
[ 20% - 240k VND ]
[ Custom amount    ]

Goes directly to your therapist: Linh
━━━━━━━━━━━━━━━━━━━━━━━
```

**Benefits**:
- Therapists get tips instantly (in VND, next day)
- Transparent (customer knows where tip goes)
- Higher tip rates (easy to tap vs counting cash)

---

### 3. Motorbike Rental

**Scenario**: Tourist rents motorbike for 3 days

**Flow**:
```
1. Tourist picks motorbike
2. Rental: "300,000 VND/day + 2,000,000 VND deposit"
3. Total: 2,900,000 VND
4. Tourist pays with crypto
5. After 3 days, returns bike
6. Rental refunds 2,000,000 VND deposit (via crypto or VND)
```

**Deposit Refund Flow**:
```go
// Initial payment
POST /api/v1/rentals/checkout
{
  "rentalDays": 3,
  "dailyRate": 300000,
  "deposit": 2000000,
  "totalVND": 2900000
}

// After return
POST /api/v1/rentals/return
{
  "rentalId": "RENT-xxx",
  "damageCharge": 0,  // No damage
  "refundAmount": 2000000,
  "refundMethod": "crypto"  // or "vnd_bank"
}

// If customer wants crypto refund:
→ "Send us your USDT wallet address"
→ System sends 86.96 USDT to customer's wallet
```

---

## 🎫 Tours & Activities

### 1. Day Tour Booking

**Scenario**: Tourist books Marble Mountains + Hoi An tour

**Flow**:
```
1. Tourist finds tour on website
2. Selects date, number of people
3. Total: 1,500,000 VND per person × 2 = 3,000,000 VND
4. Checkout → Pay with crypto
5. Receive confirmation email + QR code ticket
6. On tour day, show QR code to guide
```

**QR Code Ticket**:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━
  DA NANG DISCOVERY TOURS
━━━━━━━━━━━━━━━━━━━━━━━━━━━

MARBLE MOUNTAINS + HOI AN

Date: November 20, 2025
Pickup: 8:00 AM
Location: Beach Resort Hotel
Guests: 2 adults

[QR CODE - TICKET ID]

Booking: TOUR-20251115-008
Paid: 3,000,000 VND (130.43 USDT)
━━━━━━━━━━━━━━━━━━━━━━━━━━━
Show this to your tour guide
━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 📊 Dashboard for Tourism Merchants

### Daily Sales Report (Restaurant/Hotel)

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
           BEACHSIDE RESTAURANT
         Sales Report - Nov 15, 2025
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

CRYPTO PAYMENTS TODAY

Total transactions: 23
Total volume: 18,450,000 VND (≈ 802 USDT)
Average ticket: 802,173 VND

By chain:
  Solana USDT:  14 txns → 11,200,000 VND (61%)
  BSC USDT:      7 txns →  5,600,000 VND (30%)
  Solana USDC:   2 txns →  1,650,000 VND (9%)

By source:
  Dine-in bills:     18 txns
  Takeaway orders:    4 txns
  Tips:               1 txn

SETTLEMENT

Available to withdraw: 18,265,500 VND
  (18,450,000 - 1% fee = 18,265,500)

Pending confirmation: 0 VND

[Request Payout to Bank]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 🎯 Special Features for Tourism

### 1. Multi-Language Support

```
Payment page automatically detects browser language:
- Vietnamese (for locals)
- English (for tourists)
- Chinese (popular in Da Nang)
- Korean (popular in Da Nang)
```

```go
func GetPaymentPage(paymentID string, lang string) {
    translations := map[string]map[string]string{
        "en": {
            "title": "Complete Your Payment",
            "instruction": "Scan the QR code with your crypto wallet",
            "amount": "Amount",
        },
        "vi": {
            "title": "Hoàn Tất Thanh Toán",
            "instruction": "Quét mã QR bằng ví crypto của bạn",
            "amount": "Số tiền",
        },
        "zh": {
            "title": "完成支付",
            "instruction": "用加密钱包扫描二维码",
            "amount": "金额",
        },
    }
}
```

---

### 2. Currency Display

**Show 3 currencies simultaneously**:
```
Amount: 2,300,000 VND
        ≈ $100 USD
        ≈ 100 USDT (Solana)
```

**API**:
```go
GET /api/v1/exchange-rates

Response:
{
  "timestamp": "2025-11-15T10:00:00Z",
  "rates": {
    "VND_USD": 23000,
    "VND_USDT": 23000,
    "USDT_USD": 1.0
  }
}
```

---

### 3. Tourist-Friendly UX

**Payment Page Features**:
- ✅ Large QR code (easy to scan from distance)
- ✅ Copy buttons for address/amount (no typing errors)
- ✅ Support all major wallets (Phantom, MetaMask, Trust Wallet, Binance)
- ✅ Network selector (Solana vs BSC) with clear icons
- ✅ Real-time confirmation (no need to refresh)
- ✅ Multi-language instructions
- ✅ Tourist-friendly error messages

**Example Error Message**:
```
❌ Wrong amount received

We expected: 100 USDT
You sent:    99.5 USDT

Please send an additional 0.5 USDT to the same address.

Need help?
WhatsApp: +84 905 123 456
Email: support@gateway.com
```

---

## 📈 Business Benefits for Tourism Merchants

### Cost Comparison

**Current: Visa/Mastercard**
- Transaction fee: 3-4%
- Currency conversion: 2-3%
- Chargeback risk: 1-2% of revenue
- Settlement: 1-3 days
- **Total cost: 6-9% + risk**

**With Crypto Gateway**
- Transaction fee: 1%
- No currency conversion (USDT = USD)
- No chargebacks (crypto is final)
- Settlement: <24 hours
- **Total cost: 1%**

**For a hotel doing 1B VND/month**:
- Current cost: 60-90M VND/month
- With crypto: 10M VND/month
- **Savings: 50-80M VND/month**

---

### Marketing Opportunity

**"First hotel in Da Nang to accept crypto payments!"**

- Attract crypto-savvy travelers
- PR coverage (news, blogs, social media)
- Listed on crypto travel websites
- Competitive advantage

**Integration with Crypto Travel Platforms**:
- Travala.com (crypto booking platform)
- Crypto.com travel
- Binance travel

---

## 🚀 Quick Start for Tourism Merchants

### 1. Sign Up (5 minutes)
1. Visit dashboard.gateway.com/signup
2. Enter business info
3. Upload license + ID
4. Wait for KYC approval (24-48 hours)

### 2. Get QR Code (2 minutes)
1. Login to dashboard
2. Click "Create Payment"
3. Enter amount
4. Get QR code
5. Print or display on screen

### 3. Receive Money (24 hours)
1. Customer scans QR → pays crypto
2. You get notification
3. Next day, request payout to bank
4. Receive VND within 24 hours

---

**Ready to accept crypto at your hotel/restaurant? 🏖️**

Contact: sales@gateway.com | WhatsApp: +84 905 XXX XXX
