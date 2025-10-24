# Payment System

This package provides a payment-agnostic payment system with support for multiple payment providers.

## Architecture

The payment system follows a provider-agnostic architecture:

```
┌─────────────────┐
│   Handlers      │  HTTP endpoints for payment operations
│  (payments.go)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ PaymentProvider │  Interface for payment providers
│   Interface     │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌────────┐  ┌────────┐
│ Stripe │  │ Future │  (PayPal, Square, etc.)
└────────┘  └────────┘
         │
         ▼
┌─────────────────┐
│   Database      │  Store payment records and invoices
│  (db/payments)  │
└─────────────────┘
```

## Supported Payment Methods

Currently using **Stripe** as the payment provider, which supports:

- ✅ **Credit Cards** (Visa, Mastercard, Amex, etc.)
- ✅ **Apple Pay** (automatically enabled via Stripe Payment Element)
- ✅ **Google Pay** (automatically enabled via Stripe Payment Element)
- ✅ **ACH Direct Debit** (for US bank accounts)
- ✅ **Other payment methods** supported by Stripe

The implementation uses Stripe's [Automatic Payment Methods](https://docs.stripe.com/payments/payment-methods/integration-options#automatic-payment-methods-integration) which automatically displays the most relevant payment methods to customers based on their location and device.

## Features

### Current Features
- ✅ One-time payments with receipt/invoice generation
- ✅ Payment status tracking
- ✅ Invoice generation with unique invoice numbers
- ✅ Webhook handling for payment lifecycle events
- ✅ Payment refunds
- ✅ Customer management
- ✅ Payment history per user

### Subscription Support (Planned)
The system is designed to support subscriptions:
- Payment type field: `one_time` vs `subscription`
- Metadata storage for subscription details
- Database structure supports recurring payments

**To implement full subscription support:**
1. Use Stripe Subscriptions API instead of Payment Intents
2. Add subscription management endpoints
3. Handle subscription lifecycle webhooks

### Payouts to Users (Future)
The system can be extended to distribute money to users using [Stripe Connect](https://stripe.com/connect):
- Onboard users as connected accounts
- Split payments between platform and users
- Automatic payouts to user bank accounts
- Cross-border payout support

## Database Schema

### Payments Table
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    provider VARCHAR(50) NOT NULL,           -- 'stripe', 'paypal', etc.
    external_payment_id VARCHAR(255) NOT NULL, -- Provider's payment ID
    type VARCHAR(50) NOT NULL,               -- 'one_time', 'subscription', 'payout'
    payment_method VARCHAR(50) NOT NULL,     -- 'card', 'apple_pay', 'google_pay', etc.
    amount BIGINT NOT NULL,                  -- Amount in cents
    currency VARCHAR(3) NOT NULL,            -- ISO 4217 currency code
    status VARCHAR(50) NOT NULL,             -- 'pending', 'succeeded', 'failed', 'refunded'
    description TEXT,
    metadata JSONB,
    invoice_id UUID REFERENCES invoices(id),
    failed_at TIMESTAMP,
    failure_reason TEXT,
    refunded_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_external_id ON payments(provider, external_payment_id);
CREATE INDEX idx_payments_status ON payments(status);
```

### Invoices Table
```sql
CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    payment_id UUID NOT NULL REFERENCES payments(id),
    invoice_number VARCHAR(50) UNIQUE NOT NULL, -- 'INV-2024-001234'
    user_id UUID NOT NULL REFERENCES users(id),
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    line_items JSONB NOT NULL,
    status VARCHAR(50) NOT NULL,             -- 'draft', 'paid', 'void'
    pdf_url TEXT,                            -- URL to download invoice PDF
    issued_at TIMESTAMP NOT NULL,
    paid_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_user_id ON invoices(user_id);
CREATE INDEX idx_invoices_payment_id ON invoices(payment_id);
```

## API Endpoints

### Create Payment
```http
POST /api/payments
Authorization: Bearer <token>
Content-Type: application/json

{
  "amount": 1000,        // Amount in cents ($10.00)
  "currency": "usd",
  "type": "one_time",    // or "subscription"
  "description": "League subscription for Premier League",
  "metadata": {
    "league_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

Response:
```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_secret": "pi_xxx_secret_xxx",
  "external_payment_id": "pi_xxx",
  "status": "pending"
}
```

### Get Payment
```http
GET /api/payments/{payment_id}
Authorization: Bearer <token>
```

### Get User Payments
```http
GET /api/payments?page=1&page_size=20
Authorization: Bearer <token>
```

### Get Invoice
```http
GET /api/invoices/{invoice_id}
Authorization: Bearer <token>
```

### Get User Invoices
```http
GET /api/invoices?page=1&page_size=20
Authorization: Bearer <token>
```

### Webhook (Stripe)
```http
POST /api/webhooks/payments/stripe
Content-Type: application/json
Stripe-Signature: xxx

{
  "type": "payment_intent.succeeded",
  "data": { ... }
}
```

## Usage Example

### Frontend Integration (React/TypeScript)

```typescript
import { loadStripe } from '@stripe/stripe-js';

// 1. Create payment on backend
const response = await fetch('/api/payments', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    amount: 1000,
    currency: 'usd',
    type: 'one_time',
    description: 'League subscription'
  })
});

const { client_secret, payment_id } = await response.json();

// 2. Initialize Stripe
const stripe = await loadStripe('pk_test_xxx');

// 3. Confirm payment (supports card, Apple Pay, Google Pay automatically)
const { error } = await stripe.confirmPayment({
  clientSecret: client_secret,
  confirmParams: {
    return_url: 'https://yourapp.com/payment-success',
  }
});

if (error) {
  console.error('Payment failed:', error.message);
} else {
  console.log('Payment successful!');
}
```

### Apple Pay / Google Pay
The Stripe Payment Element automatically detects and displays Apple Pay or Google Pay buttons when:
- User's browser/device supports it
- User has payment methods set up
- Your domain is verified with Apple/Google

No additional code required!

## Environment Variables

```bash
# Stripe API keys
STRIPE_SECRET_KEY=sk_test_xxx  # For backend
STRIPE_PUBLIC_KEY=pk_test_xxx  # For frontend

# Stripe webhook signing secret
STRIPE_WEBHOOK_SECRET=whsec_xxx
```

## Webhook Setup

1. Go to Stripe Dashboard → Developers → Webhooks
2. Add endpoint: `https://yourapi.com/api/webhooks/payments/stripe`
3. Select events to listen:
   - `payment_intent.succeeded`
   - `payment_intent.payment_failed`
   - `charge.refunded`
4. Copy webhook signing secret to `STRIPE_WEBHOOK_SECRET`

## Testing

Stripe provides test card numbers:
- **Success**: `4242 4242 4242 4242`
- **Decline**: `4000 0000 0000 0002`
- **3D Secure**: `4000 0025 0000 3155`

Expiry: Any future date
CVC: Any 3 digits
ZIP: Any 5 digits

## Future Enhancements

### Subscriptions
```go
// Add subscription management
func (s *StripeProvider) CreateSubscription(customerID, priceID string) error
func (s *StripeProvider) CancelSubscription(subscriptionID string) error
func (s *StripeProvider) UpdateSubscription(subscriptionID, priceID string) error
```

### Payouts (Stripe Connect)
```go
// Add payout functionality
func (s *StripeProvider) CreateConnectedAccount(userID uuid.UUID) error
func (s *StripeProvider) PayoutToUser(accountID string, amount int64) error
```

### Multiple Providers
```go
// Add PayPal support
type PayPalProvider struct {}
func (p *PayPalProvider) CreatePaymentIntent(...) error

// Add Square support
type SquareProvider struct {}
func (s *SquareProvider) CreatePaymentIntent(...) error
```

## Security Considerations

1. **API Keys**: Never expose `STRIPE_SECRET_KEY` in frontend code
2. **Webhook Verification**: Always verify webhook signatures
3. **Amount Validation**: Validate amounts server-side (never trust client)
4. **Idempotency**: Use idempotency keys for payment operations
5. **PCI Compliance**: Never handle raw card data (Stripe handles this)

## Support

For payment provider-specific issues:
- Stripe: https://support.stripe.com
- Documentation: https://docs.stripe.com
