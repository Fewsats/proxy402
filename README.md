# Proxy402 - Monetize APIs with x402

Proxy402 lets you monetize APIs by requiring x402 payments on Base before accessing your endpoints.

## Quick Start (Client)

Test Proxy402 instantly with our pre-configured client:

```bash
# Clone the repo
git clone https://github.com/Fewsats/proxy402.git
cd proxy402/client

# Copy sample env file
cp .env-local .env

# Install dependencies
npm install

# Run client against a test endpoint (returns Bitcoin whitepaper)
npm run client https://proxy402.com/wUUbqudYsM
```

This test route requires Base Sepolia testnet ETH and USDC. Get them from these faucets:
- [Base Sepolia ETH](https://portal.cdp.coinbase.com/products/faucet)
- [Base Sepolia USDC](https://faucet.circle.com/)

## Running Your Own Server

### Prerequisites

- [Docker & Docker Compose](https://docs.docker.com/engine/install/)

### Setup

```bash
# Clone the repo (if you haven't already)
git clone https://github.com/Fewsats/proxy402.git
cd proxy402

# Configure environment
cp .env.example .env
# Edit .env file with your details

# Start the server with Docker Compose
docker compose up
```

Important .env variables:
- `X402_TESTNET_PAYMENT_ADDRESS` & `X402_MAINNET_PAYMENT_ADDRESS`: Your Base addresses for receiving payments
- `GOOGLE_CLIENT_ID` & `GOOGLE_CLIENT_SECRET`: For auth (obtain from [Google Cloud Console](https://console.cloud.google.com/apis/credentials))

### Creating Monetized Routes

1. Visit `http://localhost:8080` and log in
2. Fill the form with:
   - Target URL to monetize
   - Price in USDC
   - HTTP method
   - Test mode (on/off)
3. Use your new link: `http://localhost:8080/YOUR_SHORT_CODE`

## Target URL Verification

Proxy402 adds a `Proxy402-Secret` header to forwarded requests:

```javascript
// Node.js example
app.get('/api/data', (req, res) => {
  if (req.headers['proxy402-secret'] !== 'YOUR_SECRET_FROM_USER_SETTINGS') {
    return res.status(403).json({ error: 'Unauthorized' });
  }
  res.json({ data: 'Your protected data' });
});
```

## Additional Information

### Client Setup Details

The client uses your private key to make payments on Base:

```dotenv
# client/.env
PRIVATE_KEY="YOUR_CLIENT_WALLET_PRIVATE_KEY" # Never commit this to Git
```

### Server API Example

```bash
# Create route via API
curl -X POST http://localhost:8080/links/shrink \
     -H "Authorization: Bearer YOUR_JWT_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
           "target_url": "https://raw.githubusercontent.com/ibz/bitcoin-whitepaper-markdown/refs/heads/master/bitcoin-whitepaper.md",
           "method": "GET",
           "price": "0.0001", 
           "is_test": true
         }'
```

### How It Works

1. Client requests protected resource
2. Server returns 402 with x402 payment token
3. Client pays using Base and x402-axios library
4. Client retries with proof of payment
5. Server forwards request to target URL

---