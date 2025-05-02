# Proxy402 - Monetize APIs with x402

Proxy402 is a Go service that allows users to create monetized proxy endpoints for existing APIs. It intercepts requests, requires x402 payments (using Lightning Network or compatible infrastructure), and then forwards the request to the target URL upon successful payment verification.

It features:

*   User authentication via Google OAuth.
*   Dashboard for creating and managing paid routes.
*   Dynamic proxying based on unique short codes.
*   Integration with x402 payment protocol.
*   Test and Live modes for payment configuration.
*   Basic client example for interacting with paid routes.

## Getting Started

### Prerequisites

*   **Go:** Version 1.21 or higher ([Installation Guide](https://go.dev/doc/install))
*   **Node.js & npm:** Version 18 or higher ([Installation Guide](https://nodejs.org/))
*   **Docker & Docker Compose:** For running the database ([Installation Guide](https://docs.docker.com/engine/install/))
*   **Git:** For cloning the repository.
*   **Wallet:** You need a way to generate payment addresses for both testnet and mainnet.

### 1. Clone the Repository

```bash
git clone https://github.com/fewsats/proxy402.git # Replace with your repo URL
cd proxy402
```

### 2. Backend Setup

#### Environment Variables

The backend requires several environment variables. Create a `.env` file in the project root based on the `.env.example` file (if one exists) or create it manually:

```dotenv
# Database Configuration (adjust if needed)
DB_HOST=localhost
DB_PORT=5432
DB_USER=user
DB_PASSWORD=password
DB_NAME=linkshrink
DB_SSLMODE=disable

# Application Configuration
APP_PORT=8080
JWT_SECRET="your-strong-jwt-secret" # Replace with a strong secret
JWT_EXPIRATION_HOURS=72

# Google OAuth Credentials
# Obtain from Google Cloud Console: https://console.cloud.google.com/apis/credentials
GOOGLE_CLIENT_ID="YOUR_GOOGLE_CLIENT_ID"
GOOGLE_CLIENT_SECRET="YOUR_GOOGLE_CLIENT_SECRET"
GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback" # Adjust port if needed

# x402 Payment Addresses (Required)
X402_TESTNET_PAYMENT_ADDRESS="your_testnet_payment_address" # e.g., your Alby testnet address
X402_MAINNET_PAYMENT_ADDRESS="your_mainnet_payment_address" # e.g., your Alby mainnet address
# Optional: Override the default x402 facilitator
# X402_FACILITATOR_URL="https://your-facilitator.com"
```

**Important:**

*   Replace placeholders like `"your-strong-jwt-secret"`, `"YOUR_GOOGLE_CLIENT_ID"`, etc., with your actual credentials.
*   The `X402_TESTNET_PAYMENT_ADDRESS` and `X402_MAINNET_PAYMENT_ADDRESS` are crucial. You must generate these using a service like Alby or your own Lightning node setup. These addresses tell the service where *it* should receive payments.

#### Database

The service uses PostgreSQL. A `docker-compose.yml` file is typically provided for easy setup:

```bash
docker-compose up -d db # Start the database service in the background
```

Wait a few moments for the database to initialize.

#### Build and Run

**Option 1: Build Executable (Recommended for Production/Stable Use)**

Build the Go application:

```bash
go build -o proxy402-server ./cmd/server
```

Run the server (it will load the `.env` file automatically):

```bash
./proxy402-server
```

**Option 2: Run Directly (Convenient for Development)**

Alternatively, during development, you can compile and run the server in one step using `go run`:

```bash
go run ./cmd/server/main.go
```

This command also loads the `.env` file automatically.

The server should start on `http://localhost:8080` (or the `APP_PORT` you specified).

#### Creating Your First Paid Route (Example)

Once the server is running, you can create paid routes.

**Method 1: Using the Web UI (Recommended)**

1.  Navigate to `http://localhost:8080` (or your `APP_PORT`) in your browser.
2.  Log in using Google OAuth.
3.  On the dashboard, use the "Add URL" form:
    *   Enter the **Target URL** you want to monetize (e.g., `https://raw.githubusercontent.com/ibz/bitcoin-whitepaper-markdown/refs/heads/master/bitcoin-whitepaper.md`).
    *   Enter the **Price** in USDC (e.g., `0.0001`).
    *   Select the HTTP **Method** (e.g., `GET`).
    *   Toggle **Test Mode** on if you want to use testnet payments.
4.  Click "Add URL". Your new route will appear in the table below.

**Method 2: Using `curl` (Programmatic Example)**

If you need to create routes programmatically, you can use `curl` after obtaining a JWT token (e.g., by inspecting browser cookies after logging in via the UI, or if you implement password-based auth).

This example creates a route to the Bitcoin whitepaper in test mode, costing 0.0001 USDC (adjust price as needed):

```bash
# Replace YOUR_JWT_TOKEN_HERE with the actual token you received after logging in
AUTH_TOKEN="YOUR_JWT_TOKEN_HERE"

curl -X POST http://localhost:8080/links/shrink \
     -H "Authorization: Bearer $AUTH_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
           "target_url": "https://raw.githubusercontent.com/ibz/bitcoin-whitepaper-markdown/refs/heads/master/bitcoin-whitepaper.md",
           "method": "GET",
           "price": "0.0001", 
           "is_test": true
         }'
```

*   `-X POST`: Specifies the HTTP POST method.
*   `-H "Authorization: Bearer ..."`: Provides your authentication token.
*   `-H "Content-Type: application/json"`: Indicates the request body format.
*   `-d '...'`: Contains the JSON payload defining the route.
    *   `target_url`: The URL you want to proxy to.
    *   `method`: The HTTP method allowed for this route.
    *   `price`: The cost in USDC (formatted as a string).
    *   `is_test`: `true` enables test mode (uses testnet payment address).

The response will contain the details of the created route, including the `short_code` and the full `access_url` (e.g., `http://localhost:8080/YOUR_SHORT_CODE`).

### 3. Getting Test Funds (Base Sepolia)

The example client uses the Base Sepolia test network. To make test payments, the client wallet (specified by `PRIVATE_KEY` in `client/.env`) needs testnet funds:

1.  **Base Sepolia ETH (for gas):**
    *   Use a faucet like the Coinbase Developer Platform Faucet: [https://portal.cdp.coinbase.com/products/faucet](https://portal.cdp.coinbase.com/products/faucet)
    *   Other options: [https://www.alchemy.com/faucets/base-sepolia](https://www.alchemy.com/faucets/base-sepolia), [https://learnweb3.io/faucets/base_sepolia/](https://learnweb3.io/faucets/base_sepolia/)

2.  **Testnet USDC on Base Sepolia:** (Required for paying routes priced in USDC)
    *   Use the Circle USDC Faucet (Select Base Sepolia): [https://faucet.circle.com/](https://faucet.circle.com/)
    *   The Coinbase Faucet also provides Base Sepolia USDC: [https://portal.cdp.coinbase.com/products/faucet](https://portal.cdp.coinbase.com/products/faucet)

### 4. Client Setup & Usage

The repository includes a basic TypeScript client in the `client/` directory to demonstrate interaction with a paid route.

**Using the Client**

Whether you run the backend locally (steps above) or use the hosted production version, you'll need to set up the client:

#### Environment Variables (Client)

Navigate to the `client/` directory and set up its environment. Create a `.env` file:

```dotenv
# client/.env
PRIVATE_KEY="YOUR_CLIENT_WALLET_PRIVATE_KEY" # Private key for the wallet PAYING for the API call
```

*   Replace `YOUR_CLIENT_WALLET_PRIVATE_KEY` with the private key of the wallet you funded with Base Sepolia ETH/USDC (see section 3).
*   **Never commit this private key to Git.**

#### Install Dependencies

```bash
cd client
npm install
```

#### Run the Client

**Option A: Against Your Local Server**

If you followed the backend setup (Section 2) and created a route, run the client pointing to your local server:

```bash
# Replace YOUR_SHORT_CODE with one from your local dashboard
npm run client http://localhost:8080/YOUR_SHORT_CODE [METHOD]
```

**Option B: Against the Production Server (No Local Backend Needed)**

To test the client without setting up the backend, you can point it directly to the hosted production service. Use this example route (which is configured for **test mode** on the production server):

```bash
# Test against a live production route (in test mode)
npm run client https://proxy402.com/ax_MWAH
```

*   `[METHOD]` is optional (e.g., `POST`, defaults to `GET`).
*   Because this specific route (`/ax_MWAH`) is in **test mode**, your client wallet (defined by `PRIVATE_KEY`) must have **Base Sepolia** ETH for gas and **Base Sepolia** USDC for the payment itself. Using a production route in *live* mode would require mainnet funds.

The client will:
1.  Attempt to make the request (GET or POST).
2.  Receive a `402 Payment Required` response with an x402 token.
3.  Use the `x402-axios` interceptor and your `PRIVATE_KEY` to pay the invoice embedded in the token (using Base Sepolia).
4.  Retry the original request with the paid preimage included in the `Authorization` header.
5.  Log the final response from the target API.

---