# LinkShrink L402 Client Example

This is an example client that demonstrates how to use the `x402-axios` package to make HTTP requests to endpoints protected by the L402 payment protocol, specifically targeting a LinkShrink server instance.

## Prerequisites

- Node.js (v20 or higher) and npm/yarn/pnpm
- A running LinkShrink server instance (see the root [README.md](../README.md) for setup instructions).
- A valid Ethereum private key (on the correct network, e.g., Base Sepolia) with funds for making payments.

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

Execute the client script using `npm run client`, providing the *full URL* of the LinkShrink paid route you want to access as a command-line argument.

```bash
npm run client -- <linkshrink_paid_route_url>
```

**Example:**

If your LinkShrink server is running locally on port 8080 and you want to access a route with the short code `aBc1De2`, you would run:

```bash
npm run client -- http://localhost:8080/aBc1De2
```

The client will attempt to make the request, automatically handling the L402 payment flow using the private key from your `.env` file.

## How It Works

The client script (`index.ts`) does the following:

1.  Loads the `PRIVATE_KEY` from the `.env` file.
2.  Reads the target URL from the command-line arguments.
3.  Creates a Viem wallet client using the private key.
4.  Wraps an Axios instance with the `withPaymentInterceptor` from `x402-axios`, providing the wallet client.
5.  Makes a GET request to the specified URL using the wrapped Axios instance.
6.  The interceptor handles any L402 challenges by generating and sending the necessary payment.
7.  Prints the response headers and data upon success, or an error message if the request fails.