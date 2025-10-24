# Stripe Setup Guide for Fandemic

**Complete guide for setting up Stripe accounts for platform, creators, and subscribers**

---

## Table of Contents

1. [Overview](#overview)
2. [Account #1: Platform Owner (You)](#account-1-platform-owner-you)
3. [Account #2: Channel Owner / Creator](#account-2-channel-owner--creator)
4. [Account #3: Regular Client / Subscriber](#account-3-regular-client--subscriber)
5. [Testing Flow](#testing-flow)
6. [Production Deployment](#production-deployment)

---

## Overview

### The Three Account Types

```
┌─────────────────────────────────────────────────────────┐
│ 1. PLATFORM OWNER (You - Fandemic Organization)        │
│    - Main Stripe account                                │
│    - Owns API keys                                      │
│    - Collects platform fees (30%)                      │
│    - Manages Connect accounts                           │
└─────────────────────────────────────────────────────────┘
                          │
                          │ Creates & manages
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 2. CHANNEL OWNER / CREATOR (Stripe Connect Express)    │
│    - Stripe Connect Express account                     │
│    - Monetizes their groups/channels                    │
│    - Receives 70% of subscription revenue              │
│    - Gets automatic payouts to bank account            │
└─────────────────────────────────────────────────────────┘
                          │
                          │ Provides content to
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 3. REGULAR CLIENT / SUBSCRIBER (Stripe Customer)       │
│    - NO Stripe account needed                          │
│    - Just needs a credit/debit card                    │
│    - Subscribes to channels                            │
│    - Charged monthly via Stripe                        │
└─────────────────────────────────────────────────────────┘
```

### Money Flow

```
Subscriber pays $9.99
        │
        ▼
Stripe processes payment
        │
        ├──> Platform keeps $3.00 (30%)
        │
        └──> Creator receives $6.99 (70%)
                 │
                 ▼
            Transfers to creator's bank account
```

---

## Account #1: Platform Owner (You)

**This is YOUR Stripe account that controls everything.**

### Step 1.1: Create Platform Stripe Account

**Purpose**: This is the master account that manages all payments and creator accounts.

**Action**: Sign up at https://dashboard.stripe.com/register

**Information Needed**:
```
Business Information:
- Business name: Fandemic Inc. (or your legal name)
- Country: [Your country of operation]
- Business type: Private company / LLC / Individual
- Email: [Your business email]
- Website: https://fandemic.io
- Industry: Social Media / Content Platforms

Contact Information:
- Full name: [Your name]
- Phone number: [Your phone]
- Address: [Your business address]

Bank Account (for receiving platform fees):
- Bank name: [Your bank]
- Routing number: [Your routing #]
- Account number: [Your account #]

Tax Information:
- Tax ID (EIN for companies, SSN for individuals): [Your tax ID]
- Legal business name: [Must match tax documents]
```

**Timeline**:
- Initial signup: 5 minutes
- Identity verification: 1-2 business days
- Bank account verification: 2-3 business days (micro-deposits)

---

### Step 1.2: Enable Stripe Connect

**Purpose**: Allows you to create and manage creator accounts.

**Action**:
1. Go to: https://dashboard.stripe.com/connect/settings
2. Click "Get started" with Connect
3. Select platform type: **"Platform or marketplace"**
4. Configure settings:

```
Platform Settings:
┌────────────────────────────────────────────────────────┐
│ Platform name: Fandemic                                │
│ Platform website: https://fandemic.io                  │
│ Support email: support@fandemic.io                     │
│ Webhook URL: https://api.fandemic.io/v3/webhooks/...  │
└────────────────────────────────────────────────────────┘

Account Type Settings:
┌────────────────────────────────────────────────────────┐
│ ✓ Express accounts (RECOMMENDED for creators)         │
│   - Creators complete Stripe-hosted onboarding        │
│   - Fastest setup, least friction                     │
│   - Creators access their own Stripe dashboard        │
│                                                        │
│ ☐ Custom accounts (Advanced - NOT needed)             │
│   - You control everything                            │
│   - More complex, requires more development           │
│                                                        │
│ ☐ Standard accounts (NOT recommended)                 │
│   - Creators leave your platform                      │
│   - Complicated payout management                     │
└────────────────────────────────────────────────────────┘

Branding:
┌────────────────────────────────────────────────────────┐
│ Logo: [Upload your logo - 512x512px minimum]          │
│ Icon: [Upload icon - 128x128px]                       │
│ Brand color: [Your primary brand color - hex code]    │
└────────────────────────────────────────────────────────┘
```

---

### Step 1.3: Obtain API Keys

**Purpose**: These keys allow your application to interact with Stripe.

**Action**: Go to https://dashboard.stripe.com/test/apikeys

**Two modes available**:

#### Test Mode (Development & Testing)
```
┌────────────────────────────────────────────────────────┐
│ Test Mode Keys (for development)                       │
├────────────────────────────────────────────────────────┤
│ Publishable key: pk_test_xxxxxxxxxxxxx                │
│   → Use in: Frontend JavaScript                        │
│   → Safe to expose publicly                           │
│   → Included in HTML/JS                               │
│                                                        │
│ Secret key: sk_test_xxxxxxxxxxxxx                     │
│   → Use in: Backend server                            │
│   → NEVER expose publicly                             │
│   → Store in environment variables                    │
│   → DO NOT commit to Git                              │
└────────────────────────────────────────────────────────┘
```

#### Live Mode (Production)
```
┌────────────────────────────────────────────────────────┐
│ Live Mode Keys (for production)                        │
├────────────────────────────────────────────────────────┤
│ Publishable key: pk_live_xxxxxxxxxxxxx                │
│ Secret key: sk_live_xxxxxxxxxxxxx                     │
│                                                        │
│ ⚠️  WARNING: Only switch to live mode after thorough  │
│    testing. Live mode processes REAL money.           │
└────────────────────────────────────────────────────────┘
```

**Security Best Practices**:
```bash
# ✓ CORRECT: Store in environment variables
export STRIPE_SECRET_KEY=sk_test_xxxxxxxxxxxxx
export STRIPE_PUBLISHABLE_KEY=pk_test_xxxxxxxxxxxxx

# ✓ CORRECT: Use .env file (add to .gitignore)
# .env
STRIPE_SECRET_KEY=sk_test_xxxxxxxxxxxxx
STRIPE_PUBLISHABLE_KEY=pk_test_xxxxxxxxxxxxx

# ✗ WRONG: Hardcoding in source code
const stripeKey = "sk_test_xxxxxxxxxxxxx";  // NEVER DO THIS!

# ✗ WRONG: Committing to Git
git add .env  # NEVER DO THIS!
```

---

### Step 1.4: Configure Webhooks

**Purpose**: Webhooks notify your server when events occur (payments succeed, subscriptions created, etc.)

**Action**: Go to https://dashboard.stripe.com/test/webhooks

**Setup**:
```
Add Endpoint Configuration:
┌────────────────────────────────────────────────────────┐
│ Endpoint URL:                                          │
│   Development: Use Stripe CLI forwarding              │
│     $ stripe listen --forward-to localhost:8080/...   │
│                                                        │
│   Production: https://api.fandemic.io/v3/webhooks/payments/stripe
│                                                        │
│ Events to listen for:                                  │
│   ✓ payment_intent.succeeded                          │
│   ✓ payment_intent.payment_failed                     │
│   ✓ customer.subscription.created                     │
│   ✓ customer.subscription.updated                     │
│   ✓ customer.subscription.deleted                     │
│   ✓ invoice.payment_succeeded                         │
│   ✓ invoice.payment_failed                            │
│   ✓ charge.refunded                                   │
│   ✓ account.updated (for Connect accounts)           │
│                                                        │
│ After creation, copy:                                  │
│   Webhook signing secret: whsec_xxxxxxxxxxxxx         │
│     → Store as: STRIPE_WEBHOOK_SECRET                │
└────────────────────────────────────────────────────────┘
```

**Development Webhook Setup**:
```bash
# Terminal 1: Start your API server
make dev

# Terminal 2: Forward webhooks to local server
stripe listen --forward-to http://localhost:8080/v3/webhooks/payments/stripe

# Output will show:
# > Ready! Your webhook signing secret is whsec_xxxxxxxxxxxxx
# Copy this to your .env file
```

---

### Step 1.5: Configure Payment Settings

**Purpose**: Set up how payments are processed.

**Action**: Go to https://dashboard.stripe.com/settings/payment_methods

**Recommended Settings**:
```
Payment Methods:
┌────────────────────────────────────────────────────────┐
│ ✓ Cards (Visa, Mastercard, Amex, Discover)           │
│ ✓ Apple Pay                                           │
│ ✓ Google Pay                                          │
│ ☐ ACH Direct Debit (Optional - for US customers)     │
│ ☐ SEPA Direct Debit (Optional - for EU customers)    │
└────────────────────────────────────────────────────────┘

Security Settings:
┌────────────────────────────────────────────────────────┐
│ ✓ Request 3D Secure when recommended                 │
│   (Required for European customers - SCA compliance)  │
│                                                        │
│ ✓ Stripe Radar (fraud detection)                     │
│   - Uses machine learning to block fraudulent cards   │
│   - Automatically enabled for all payments            │
│                                                        │
│ ✓ Automatic email receipts                           │
│   - Customers receive receipt after payment           │
│   - Includes invoice PDF                              │
└────────────────────────────────────────────────────────┘

Subscription Settings:
┌────────────────────────────────────────────────────────┐
│ Billing retries: Smart retries (recommended)          │
│   - Stripe automatically retries failed payments      │
│   - 4 attempts over ~2 weeks                          │
│                                                        │
│ Failed payment emails: Enabled                        │
│   - Notifies customers when payment fails             │
│                                                        │
│ Default currency: USD (or your preference)            │
└────────────────────────────────────────────────────────┘
```

---

### Step 1.6: Platform Application Configuration

**Update your application's environment**:

```bash
# Development (.env.development)
STRIPE_SECRET_KEY=sk_test_xxxxxxxxxxxxx
STRIPE_PUBLISHABLE_KEY=pk_test_xxxxxxxxxxxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxxx
PLATFORM_FEE_PERCENT=30
APP_URL=http://localhost:3000
API_URL=http://localhost:8080

# Production (.env.production)
STRIPE_SECRET_KEY=sk_live_xxxxxxxxxxxxx
STRIPE_PUBLISHABLE_KEY=pk_live_xxxxxxxxxxxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxxx  # Different from dev!
PLATFORM_FEE_PERCENT=30
APP_URL=https://fandemic.io
API_URL=https://api.fandemic.io
```

**Frontend Configuration**:
```javascript
// React / Next.js / Vue
// .env.local
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_xxxxxxxxxxxxx
REACT_APP_STRIPE_PUBLISHABLE_KEY=pk_test_xxxxxxxxxxxxx
VITE_STRIPE_PUBLISHABLE_KEY=pk_test_xxxxxxxxxxxxx

// Load Stripe.js
import { loadStripe } from '@stripe/stripe-js';
const stripe = await loadStripe(process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY);
```

---

## Account #2: Channel Owner / Creator

**Creators monetize their channels/groups through Stripe Connect Express accounts.**

### What is Stripe Connect Express?

**Key Points**:
- **NOT a separate Stripe account signup** - Created automatically through your platform
- **Hosted by Stripe** - Stripe handles the onboarding form and compliance
- **Minimal friction** - Creators complete onboarding in ~5 minutes
- **Automatic payouts** - Money transfers directly to their bank account
- **Own dashboard** - Creators can view earnings in their Stripe Express dashboard

### Step 2.1: Creator Initiates Account Creation

**User Flow**:
```
1. Creator signs up on your platform (username, email, password)
2. Creator navigates to "Monetization" or "Earnings" section
3. Creator clicks "Enable Subscriptions" or "Connect Stripe"
4. Your API creates a Stripe Connect Express account
5. Creator is redirected to Stripe-hosted onboarding form
```

**API Endpoint**:
```bash
POST /stripe/connect/account
Authorization: Bearer {creator_token}

# Response:
{
  "success": true,
  "data": {
    "account_id": "acct_1234567890",
    "onboarding_url": "https://connect.stripe.com/setup/s/xxxxx",
    "created": true
  }
}
```

**Frontend Code Example**:
```javascript
// React example
const handleConnectStripe = async () => {
  const response = await fetch('/stripe/connect/account', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${authToken}`,
    },
  });

  const data = await response.json();

  // Redirect to Stripe onboarding
  window.location.href = data.data.onboarding_url;
};
```

---

### Step 2.2: Creator Completes Onboarding

**Stripe will ask for this information**:

#### Business Information
```
Business Type:
┌────────────────────────────────────────────────────────┐
│ ○ Individual (Most common for creators)               │
│   - Content creator, influencer, artist               │
│   - Uses personal info and SSN                        │
│                                                        │
│ ○ Company                                             │
│   - Registered business entity                        │
│   - Uses business name and EIN                        │
└────────────────────────────────────────────────────────┘

If Individual:
- Legal name: [Full legal name]
- Date of birth: [MM/DD/YYYY]
- SSN: [Social Security Number]
- Phone: [Phone number]
- Address: [Home address]

If Company:
- Business name: [Legal business name]
- EIN: [Employer Identification Number]
- Business address: [Business address]
- Business phone: [Phone number]
- Representative info: [Person managing account]
```

#### Bank Account Information
```
Payout Destination:
┌────────────────────────────────────────────────────────┐
│ Bank account (for receiving payouts)                  │
│                                                        │
│ Bank name: [e.g., Chase, Bank of America]            │
│ Account holder name: [Full name on account]          │
│ Routing number: [9-digit routing number]             │
│ Account number: [Bank account number]                │
│ Account type: ○ Checking  ○ Savings                  │
│                                                        │
│ ⚠️  IMPORTANT:                                        │
│   - Account must be in creator's name                 │
│   - Must be US bank account for US creators           │
│   - Verify numbers carefully to avoid payout delays   │
└────────────────────────────────────────────────────────┘
```

#### Test Data (Development Only)
```
For testing with Stripe test mode:

Bank Account:
- Routing number: 110000000 (Stripe test routing number)
- Account number: 000123456789 (any digits work)

SSN/EIN:
- Use any 9 digits: 123456789

Phone:
- Any valid format: (555) 555-5555

Address:
- Any US address: 123 Main St, San Francisco, CA 94102
```

---

### Step 2.3: Verify Onboarding Status

**Check if creator completed onboarding**:

```bash
GET /stripe/connect/account/status
Authorization: Bearer {creator_token}

# Response - Before onboarding:
{
  "has_account": true,
  "account_id": "acct_xxx",
  "onboarding_completed": false,
  "charges_enabled": false,
  "payouts_enabled": false,
  "requirements_pending": ["bank_account", "tax_id", "dob"]
}

# Response - After onboarding:
{
  "has_account": true,
  "account_id": "acct_xxx",
  "onboarding_completed": true,
  "charges_enabled": true,
  "payouts_enabled": true,
  "details_submitted": true
}
```

**Frontend Example**:
```javascript
// Show onboarding status in UI
const OnboardingStatus = ({ creator }) => {
  const { charges_enabled, payouts_enabled, requirements_pending } = creator;

  if (charges_enabled && payouts_enabled) {
    return (
      <div className="status-complete">
        ✓ Your Stripe account is active. You can now monetize your channels!
      </div>
    );
  }

  return (
    <div className="status-incomplete">
      ⚠️ Complete Stripe onboarding to receive payments
      {requirements_pending && (
        <ul>
          {requirements_pending.map(req => (
            <li key={req}>Missing: {req}</li>
          ))}
        </ul>
      )}
      <button onClick={continueOnboarding}>
        Complete Onboarding
      </button>
    </div>
  );
};
```

---

### Step 2.4: Creator Enables Channel Subscriptions

**Once onboarding is complete**, creators can enable subscriptions on their channels:

```bash
POST /groups/{groupId}/subscription/enable
Authorization: Bearer {creator_token}
Content-Type: application/json

{
  "price_cents": 999,        # $9.99
  "currency": "usd",
  "interval": "month"        # or "year"
}

# Response:
{
  "success": true,
  "data": {
    "group_id": "group-uuid",
    "subscription_enabled": true,
    "subscription_price_cents": 999,
    "subscription_currency": "usd",
    "subscription_interval": "month",
    "stripe_product_id": "prod_xxxxx",
    "stripe_price_id": "price_xxxxx"
  }
}
```

**What happens behind the scenes**:
1. Your API creates a Stripe Product: "Subscription to [Channel Name]"
2. Your API creates a Stripe Price: $9.99/month
3. Channel is marked as monetized in your database
4. Subscribers can now purchase subscriptions

---

### Step 2.5: Creator Views Earnings

**Creators can track their earnings** through your platform:

```bash
GET /stripe/connect/earnings
Authorization: Bearer {creator_token}

# Response:
{
  "earnings": {
    "total_subscriptions": 15,
    "total_subscribers": 15,
    "total_gross_amount": 14985,     # $149.85
    "total_platform_fees": 4496,     # $44.96 (30%)
    "total_creator_earnings": 10489, # $104.89 (70%)
    "completed_transfers": 10489,
    "pending_transfers": 0,
    "currency": "usd"
  },
  "recent_fees": [
    {
      "subscription_id": "sub-uuid",
      "subscriber_username": "fan123",
      "gross_amount": 999,
      "platform_fee_amount": 300,
      "creator_amount": 699,
      "status": "completed",
      "stripe_transfer_id": "tr_xxxxx",
      "transferred_at": "2025-10-22T10:05:00Z"
    }
  ]
}
```

**Creator can also access Stripe Express Dashboard**:
- Direct link: `https://dashboard.stripe.com/express/{account_id}`
- View all transactions, payouts, and bank account info
- Download tax documents (1099-K at year-end)

---

### Step 2.6: Creator Receives Payouts

**Automatic payout schedule**:

```
Default Payout Schedule:
┌────────────────────────────────────────────────────────┐
│ Frequency: Daily (automatic)                           │
│ Timeline: Funds arrive in 2-3 business days           │
│ Minimum: $1 USD                                       │
│                                                        │
│ Example Timeline:                                      │
│ Monday: Subscriber pays $9.99                         │
│ Monday: Stripe transfers $6.99 to creator balance     │
│ Tuesday: Payout initiated to bank account             │
│ Thursday: Money arrives in creator's bank             │
└────────────────────────────────────────────────────────┘
```

**Payout can be customized** by creator in their Stripe dashboard:
- Weekly payouts (every Monday)
- Monthly payouts (1st of month)
- Manual payouts (on-demand)

---

## Account #3: Regular Client / Subscriber

**Subscribers need NO Stripe account - just a payment method!**

### Key Points

```
What Subscribers Need:
┌────────────────────────────────────────────────────────┐
│ ✓ Account on your platform (username/email/password)  │
│ ✓ Valid credit/debit card                            │
│ ✗ NO Stripe account required                         │
│ ✗ NO bank account required                           │
│ ✗ NO business information required                    │
└────────────────────────────────────────────────────────┘
```

### Step 3.1: Subscriber Signs Up

**Normal platform registration**:
```bash
POST /signup
Content-Type: application/json

{
  "username": "subscriber123",
  "email": "subscriber@example.com",
  "password": "securepass123"
}

# Standard user account creation - nothing Stripe-specific
```

---

### Step 3.2: Subscriber Browses Monetized Channels

**Channels show subscription requirement**:

```javascript
// Frontend display
const ChannelCard = ({ channel }) => {
  return (
    <div className="channel-card">
      <h3>{channel.name}</h3>
      <p>{channel.description}</p>

      {channel.subscription_enabled && (
        <div className="subscription-badge">
          💎 Premium Channel
          <span className="price">
            ${(channel.subscription_price_cents / 100).toFixed(2)}/month
          </span>
        </div>
      )}

      <button onClick={() => subscribe(channel.id)}>
        Subscribe Now
      </button>
    </div>
  );
};
```

---

### Step 3.3: Subscriber Initiates Subscription

**User clicks "Subscribe" button**:

```bash
POST /subscriptions
Authorization: Bearer {subscriber_token}
Content-Type: application/json

{
  "group_id": "channel-uuid-here"
}

# Response:
{
  "success": true,
  "data": {
    "subscription_id": "sub-uuid",
    "stripe_subscription_id": "sub_xxxxx",
    "stripe_customer_id": "cus_xxxxx",
    "client_secret": "pi_xxxxx_secret_xxxxx",
    "status": "incomplete",
    "amount_cents": 999,
    "currency": "usd",
    "next_steps": "Use client_secret to confirm payment with Stripe Elements"
  }
}
```

**Behind the scenes**:
1. Your API creates a Stripe Customer (cus_xxxxx) for the subscriber
2. Your API creates a Stripe Subscription (sub_xxxxx)
3. Your API returns a `client_secret` for payment confirmation

---

### Step 3.4: Subscriber Enters Payment Information

**Frontend uses Stripe Elements** to collect card information securely:

```javascript
// React + Stripe Elements example
import { CardElement, useStripe, useElements } from '@stripe/react-stripe-js';

const SubscriptionPaymentForm = ({ clientSecret, onSuccess }) => {
  const stripe = useStripe();
  const elements = useElements();

  const handleSubmit = async (event) => {
    event.preventDefault();

    if (!stripe || !elements) return;

    // Confirm payment with Stripe
    const { error, paymentIntent } = await stripe.confirmCardPayment(
      clientSecret,
      {
        payment_method: {
          card: elements.getElement(CardElement),
          billing_details: {
            name: 'Subscriber Name',
            email: 'subscriber@example.com'
          }
        }
      }
    );

    if (error) {
      // Show error to user
      console.error('Payment failed:', error.message);
      alert(`Payment failed: ${error.message}`);
    } else if (paymentIntent.status === 'succeeded') {
      // Payment successful!
      console.log('Payment succeeded!', paymentIntent);
      onSuccess();
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <h3>Subscribe for $9.99/month</h3>

      <CardElement
        options={{
          style: {
            base: {
              fontSize: '16px',
              color: '#424770',
              '::placeholder': {
                color: '#aab7c4',
              },
            },
            invalid: {
              color: '#9e2146',
            },
          },
        }}
      />

      <button type="submit" disabled={!stripe}>
        Subscribe Now
      </button>

      <p className="security-note">
        🔒 Secured by Stripe. Your card information is never stored on our servers.
      </p>
    </form>
  );
};
```

**What the subscriber sees**:
```
┌──────────────────────────────────────────────────┐
│ Subscribe to Premium Channel                     │
│ $9.99/month                                      │
├──────────────────────────────────────────────────┤
│                                                  │
│ Card Number:  [4242 4242 4242 4242]            │
│ Expiry:       [12/25]  CVC: [123]              │
│                                                  │
│ Billing Name: [John Doe                    ]    │
│                                                  │
│ [ Subscribe Now ]                               │
│                                                  │
│ 🔒 Secured by Stripe                            │
│ Cancel anytime. Renews monthly.                 │
└──────────────────────────────────────────────────┘
```

---

### Step 3.5: Payment Processes & Webhook Fires

**Payment flow**:

```
1. User submits card info → Stripe validates card
2. Stripe charges $9.99 → Payment succeeds
3. Stripe sends webhook → Your server receives notification
4. Your server processes webhook:
   - Updates subscription status to "active"
   - Creates payment record ($9.99)
   - Creates invoice record (PDF receipt)
   - Calculates platform fee ($3.00)
   - Calculates creator amount ($6.99)
   - Transfers $6.99 to creator
   - Grants user access to channel
5. Frontend redirects → User can now access premium content
```

**Webhook event received**:
```json
{
  "type": "invoice.payment_succeeded",
  "data": {
    "object": {
      "id": "in_xxxxx",
      "subscription": "sub_xxxxx",
      "amount_paid": 999,
      "customer": "cus_xxxxx",
      "status": "paid"
    }
  }
}
```

---

### Step 3.6: Subscriber Has Access

**Subscription is now active**:

```bash
GET /subscriptions
Authorization: Bearer {subscriber_token}

# Response:
{
  "data": [
    {
      "id": "sub-uuid",
      "group_id": "channel-uuid",
      "group_name": "Premium Channel",
      "status": "active",
      "stripe_subscription_id": "sub_xxxxx",
      "current_period_start": "2025-10-22T10:00:00Z",
      "current_period_end": "2025-11-22T10:00:00Z",
      "price_cents": 999,
      "currency": "usd",
      "interval": "month",
      "cancel_at_period_end": false
    }
  ]
}
```

**User can now**:
- Access premium channel content
- View subscription in account settings
- Cancel subscription anytime
- Receive email receipts for each payment

---

### Step 3.7: Automatic Renewals

**Monthly recurring payments**:

```
Timeline:
┌────────────────────────────────────────────────────────┐
│ Month 1 (Oct 22): Subscribe - Pay $9.99              │
│ Month 2 (Nov 22): Auto-renew - Pay $9.99             │
│ Month 3 (Dec 22): Auto-renew - Pay $9.99             │
│ ...                                                    │
│                                                        │
│ Each month:                                           │
│ - Stripe charges card automatically                   │
│ - Webhook fires → Your server updates records         │
│ - Creator receives $6.99                             │
│ - User receives email receipt                         │
│ - Access continues without interruption               │
└────────────────────────────────────────────────────────┘
```

**If payment fails**:
```
Day 1: Payment attempt fails → Status: "past_due"
Day 3: Stripe retries → Still fails
Day 7: Stripe retries → Still fails
Day 14: Stripe retries → Still fails
Day 21: Final retry → Fails → Status: "canceled"

Your server:
- Receives webhook: customer.subscription.updated
- Marks subscription as "past_due" or "canceled"
- Revokes access to premium channel
- Sends notification to user: "Update payment method"
```

---

### Step 3.8: Cancellation

**User can cancel anytime**:

```bash
POST /subscriptions/{subscription_id}/cancel
Authorization: Bearer {subscriber_token}
Content-Type: application/json

{
  "cancel_at_period_end": true  # Access until end of paid period
  # OR
  "cancel_immediately": true    # Revoke access now
}

# Response:
{
  "success": true,
  "message": "Subscription canceled",
  "data": {
    "subscription_id": "sub-uuid",
    "status": "canceled",
    "cancel_at_period_end": true,
    "current_period_end": "2025-11-22T10:00:00Z",
    "access_until": "2025-11-22T10:00:00Z"
  }
}
```

**What happens**:
- If `cancel_at_period_end: true`:
  - User keeps access until end of paid period (Nov 22)
  - No more charges after that
  - Can resubscribe later

- If `cancel_immediately: true`:
  - Access revoked immediately
  - No refund (unless you manually process one)
  - Can resubscribe later

---

## Testing Flow

### Quick End-to-End Test (5 Minutes)

**Prerequisites**:
```bash
# 1. Set environment variables
export STRIPE_SECRET_KEY=sk_test_xxxxxxxxxxxxx
export STRIPE_PUBLISHABLE_KEY=pk_test_xxxxxxxxxxxxx

# 2. Start webhook forwarding
stripe listen --forward-to http://localhost:8080/v3/webhooks/payments/stripe

# 3. Start your API server
make dev

# 4. Start your frontend
npm run dev
```

**Test Steps**:

#### 1. Create Creator Account (1 minute)
```bash
# Sign up as creator
curl -X POST http://localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test_creator",
    "email": "creator@test.com",
    "password": "test123"
  }'

# Login
TOKEN=$(curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test_creator","password":"test123"}' \
  | jq -r '.access_token')
```

#### 2. Creator Onboarding (2 minutes)
```bash
# Create Connect account
ONBOARDING=$(curl -s -X POST http://localhost:8080/stripe/connect/account \
  -H "Authorization: Bearer $TOKEN")

echo $ONBOARDING | jq '.data.onboarding_url'

# Open URL in browser and complete form with test data:
# Name: Test Creator
# DOB: 01/01/1990
# SSN: 000000000
# Bank routing: 110000000
# Bank account: 000123456789
```

#### 3. Enable Channel Subscription (30 seconds)
```bash
# Create channel
GROUP_ID=$(curl -s -X POST http://localhost:8080/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Channel","description":"Testing"}' \
  | jq -r '.data.id')

# Enable subscriptions
curl -X POST http://localhost:8080/groups/$GROUP_ID/subscription/enable \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"price_cents":999,"currency":"usd","interval":"month"}'
```

#### 4. Create Subscriber & Subscribe (1 minute)
```bash
# Sign up as subscriber
curl -X POST http://localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test_sub",
    "email": "sub@test.com",
    "password": "test123"
  }'

# Login
SUB_TOKEN=$(curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test_sub","password":"test123"}' \
  | jq -r '.access_token')

# Initiate subscription
curl -X POST http://localhost:8080/subscriptions \
  -H "Authorization: Bearer $SUB_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"group_id\":\"$GROUP_ID\"}"

# Use frontend to complete payment with test card:
# Card: 4242 4242 4242 4242
# Expiry: 12/34
# CVC: 123
```

#### 5. Verify Everything Works (30 seconds)
```bash
# Check subscription active
curl -X GET http://localhost:8080/subscriptions \
  -H "Authorization: Bearer $SUB_TOKEN"

# Check creator earnings
curl -X GET http://localhost:8080/stripe/connect/earnings \
  -H "Authorization: Bearer $TOKEN"
```

**Success Criteria**:
- ✅ Subscription status: "active"
- ✅ User has access to channel
- ✅ Creator sees $6.99 earnings (70%)
- ✅ Platform fee: $3.00 (30%)
- ✅ Webhook received and processed

---

## Production Deployment

### Checklist Before Going Live

#### 1. Platform Account Ready
- [ ] Stripe account fully verified (identity + bank account)
- [ ] Connect enabled and configured
- [ ] Live API keys obtained and stored securely
- [ ] Branding configured (logo, colors)
- [ ] Business profile complete

#### 2. Webhook Endpoint Configured
- [ ] Production webhook URL: `https://api.fandemic.io/v3/webhooks/payments/stripe`
- [ ] HTTPS with valid SSL certificate
- [ ] Webhook signing secret stored: `STRIPE_WEBHOOK_SECRET`
- [ ] Test webhook sent successfully
- [ ] Endpoint responding < 5 seconds

#### 3. Application Configuration
- [ ] Environment variables updated for production
  ```bash
  STRIPE_SECRET_KEY=sk_live_xxxxxxxxxxxxx
  STRIPE_PUBLISHABLE_KEY=pk_live_xxxxxxxxxxxxx
  STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxxx
  PLATFORM_FEE_PERCENT=30
  ```
- [ ] Frontend using live publishable key
- [ ] Database migrations applied
- [ ] Monitoring and logging configured

#### 4. Settings Configured
- [ ] Payment methods enabled (cards, Apple Pay, Google Pay)
- [ ] 3D Secure enabled (required for EU)
- [ ] Email receipts enabled
- [ ] Fraud detection (Radar) active
- [ ] Subscription retry logic configured

#### 5. Testing Complete
- [ ] Develop environment tested with live keys
- [ ] Real $1 test transaction processed
- [ ] Webhook events verified
- [ ] Creator received test payout
- [ ] Cancellation flow works
- [ ] Refund processed successfully

#### 6. Launch
- [ ] Switch to live mode
- [ ] Monitor first 10 transactions closely
- [ ] Watch for webhook failures
- [ ] Check error logs hourly (first 24 hours)
- [ ] Verify creator payouts processing

---

## Summary

### What Each Account Needs

**Platform Owner (You)**:
```
1. Sign up for Stripe account
2. Complete business verification
3. Enable Stripe Connect
4. Obtain API keys (test + live)
5. Configure webhooks
6. Set up payment settings
Time: 1-2 days (including verification)
Cost: Free (Stripe takes 2.9% + $0.30 per transaction)
```

**Channel Owner / Creator**:
```
1. Sign up on YOUR platform (not Stripe directly)
2. Click "Enable Monetization" in your app
3. Complete Stripe onboarding form (5 minutes)
   - Provide: Name, DOB, SSN, bank account
4. Done! Can now monetize channels
Time: 5-10 minutes
Cost: Free (receives 70% of subscription revenue)
```

**Regular Subscriber**:
```
1. Sign up on YOUR platform
2. Browse channels and click "Subscribe"
3. Enter credit/debit card information
4. Done! Has access to premium content
Time: 2 minutes
Cost: Subscription price (e.g., $9.99/month)
```

---

## Quick Reference

### Test Cards
```
Success: 4242 4242 4242 4242
3D Secure: 4000 0025 0000 3155
Declined: 4000 0000 0000 9995
```

### API Endpoints
```
Creator Onboarding:
  POST /stripe/connect/account
  GET  /stripe/connect/account/status

Monetization:
  POST /groups/{id}/subscription/enable
  GET  /stripe/connect/earnings

Subscriptions:
  POST /subscriptions
  GET  /subscriptions
  POST /subscriptions/{id}/cancel

Payments:
  GET  /payments
  POST /payments/{id}/refund
```

### Key URLs
```
Stripe Dashboard: https://dashboard.stripe.com
Test Mode: https://dashboard.stripe.com/test
API Keys: https://dashboard.stripe.com/test/apikeys
Webhooks: https://dashboard.stripe.com/test/webhooks
Connect: https://dashboard.stripe.com/connect
Documentation: https://stripe.com/docs
```

---

## Need Help?

**Stripe Support**:
- Dashboard: https://dashboard.stripe.com/support
- Docs: https://stripe.com/docs
- Email: support@stripe.com

**Integration Issues**:
- Check server logs for errors
- Verify webhook signature
- Test with Stripe CLI: `stripe listen`
- Use Stripe Dashboard logs tab

**Common Issues**:
1. "Webhook signature verification failed"
   → Check `STRIPE_WEBHOOK_SECRET` matches dashboard

2. "Payment requires authentication"
   → User needs to complete 3D Secure challenge

3. "Connected account not onboarded"
   → Creator must complete Stripe onboarding form

4. "Subscription already exists"
   → User already subscribed to this channel
