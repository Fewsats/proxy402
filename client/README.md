# LinkShrink L402 Client Example

This is an example client that demonstrates how to use the `x402-axios` package to make HTTP requests to endpoints protected by the L402 payment protocol, specifically targeting a LinkShrink server instance.

## Prerequisites

- Node.js (v20 or higher) and npm/yarn/pnpm
- A running LinkShrink server instance (see the root [README.md](../README.md) for setup instructions).
- A valid Ethereum private key with funds for making payments:
  - USDC on Base Sepolia testnet for test endpoints
  - USDC on Base mainnet for production endpoints

## Setup

1.  Create a `.env` file in this directory (`client/`) with your private key:
    ```env
    # client/.env
    PRIVATE_KEY=0xYourPrivateKeyHere
    ```
    *   **Security:** Treat your private key with extreme care. Ensure this file is listed in your `.gitignore` and never committed.

2.  Install dependencies:
    ```bash
    # Make sure you are in the client/ directory
    npm install
    # or yarn install / pnpm install
    ```

## Running the Client

### Basic Usage

Use the client script with the URL of the LinkShrink paid route you want to access:

```bash
npm run client -- <url> [--method GET|POST] [--network base-sepolia|base-mainnet]
```

By default, the client uses the Base Sepolia testnet. For production URLs, you need to explicitly specify the mainnet.

### Network-Specific Commands

We provide convenience scripts for each network:

- **For testnet (Base Sepolia):**
  ```bash
  npm run testnet -- <url> [--method GET|POST]
  ```

- **For mainnet (Base):**
  ```bash
  npm run mainnet -- <url> [--method GET|POST]
  ```

### Examples:

**Testnet (default):**
```bash
# Using default client command with testnet
npm run client -- http://localhost:8080/aBc1De2

# Explicitly specifying testnet
npm run client -- http://localhost:8080/aBc1De2 --network base-sepolia

# Using the testnet convenience script
npm run testnet -- http://localhost:8080/aBc1De2
```

**Mainnet:**
```bash
# Specifying mainnet with the client command
npm run client -- https://linkshrink.io/xyz789 --network base-mainnet

# Using the mainnet convenience script
npm run mainnet -- https://linkshrink.io/xyz789

# Making a POST request on mainnet
npm run mainnet -- https://linkshrink.io/xyz789 --method POST
```

## Troubleshooting

### Network Mismatch Issues

The most common issue is using a testnet wallet to pay for mainnet resources or vice versa. Errors you might see include:

- 500 Internal Server Error with empty response
- JSON parsing errors in the payment protocol
- Transaction verification failures

Make sure you're:
1. Using the correct network flag for your target URL
2. Using a wallet with funds on the appropriate network
3. Setting proper gas prices for the network

## How It Works

The client script (`index.ts`) does the following:

1.  Loads the `PRIVATE_KEY` from the `.env` file.
2.  Parses command-line arguments to determine the target URL, HTTP method, and network.
3.  Creates a Viem wallet client using the private key on the appropriate chain.
4.  Wraps an Axios instance with the `withPaymentInterceptor` from `x402-axios`, providing the wallet client.
5.  Makes a request to the specified URL using the wrapped Axios instance.
6.  The interceptor handles any L402 challenges by generating and sending the necessary payment.
7.  Prints the response headers and data upon success, or an error message if the request fails.