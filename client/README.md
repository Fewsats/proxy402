# Proxy402 Client Example

This is an example client that demonstrates how to use the `x402-axios` package to make HTTP requests to endpoints protected by the x402 payment protocol, specifically targeting a Proxy402 server instance.

## Prerequisites

- Node.js (v20 or higher) and npm/yarn/pnpm
- A running Proxy402 server instance (see the root [README.md](../README.md) for setup instructions).
- A valid Base account & private key with funds for making payments.

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

Execute the client script using `npm run client`, providing the *full URL* of the Proxy402 paid route you want to access as a command-line argument. You can optionally provide `POST` as a second argument to make a POST request; otherwise, it defaults to GET.

```bash
npm run client -- <proxy402_paid_route_url> [GET|POST]
```

**Examples:**

*   Make a GET request (default):
    ```bash
    npm run client -- http://localhost:8080/aBc1De2
    ```
*   Make a POST request:
    ```bash
    npm run client -- http://localhost:8080/xyz789 POST
    ```
*   Using an existing payment credit (e.g., from a previous transaction's `x-payment-response` header):
    ```bash
    npm run client -- http://localhost:8080/someRoute --payment-header "your_x_payment_header_value_here"
    ```

The client will attempt to make the request, automatically handling the x402 payment flow using the private key from your `.env` file if no valid `X-Payment` header is provided for existing credit.

## How It Works

The client script (`index.ts`) does the following:

1.  Loads the `PRIVATE_KEY` from the `.env` file.
2.  Reads the target URL from the first command-line argument.
3.  Reads the optional HTTP method (GET/POST) from the second command-line argument, defaulting to GET.
4.  Creates a Viem wallet client using the private key.
5.  Wraps an Axios instance with the `withPaymentInterceptor` from `x402-axios`, providing the wallet client.
6.  Makes a request (using the specified or default method) to the specified URL using the wrapped Axios instance.
7.  The interceptor handles any x402 challenges by generating and sending the necessary payment.
8.  Prints the response headers and data upon success, or an error message if the request fails.

## Example Walkthrough

This section illustrates the client's behavior with and without using an existing payment credit, using illustrative snippets from actual client logs.

### 1. Initial Payment for a Resource

When you access a paid route for the first time without providing an existing payment credit, `x402-axios` automatically handles the payment challenge.

Running the following command:
```bash
npm run client -- http://localhost:8080/ElMbSMtY9v
```

Will produce output similar to this (key parts highlighted):

First, the client makes a request without any payment information:
```log
> tsx index.ts http://localhost:8080/ElMbSMtY9v
Using TESTNET (Base Sepolia) chain
Making a GET request to: http://localhost:8080/ElMbSMtY9v
Starting Request: {
  ...
  "headers": { "Accept": "application/json, text/plain, */*" },
  "baseURL": "http://localhost:8080/ElMbSMtY9v",
  "method": "get",
  ...
}
```
The server responds with a `402 Payment Required` status:
```log
Response Error: 402
Response Error Headers: {
  "content-type": "application/json; charset=utf-8",
  "payment-protocol": "X402", // Server indicates X402 protocol
  // The WWW-Authenticate header (not shown for brevity) contains the payment challenge
  ...
}
```
`x402-axios` then automatically constructs and sends a new request including the `X-PAYMENT` header:
```log
Starting Request: {
  ...
  "headers": {
    "Accept": "application/json, text/plain, */*",
    "X-PAYMENT": "eyJ4NDAyVm...(auto-generated payment header value)...",
    "Access-Control-Expose-Headers": "X-PAYMENT-RESPONSE"
  },
  "baseURL": "http://localhost:8080/ElMbSMtY9v",
  "method": "get",
  "__is402Retry": true
  ...
}
```
Upon successful payment, the server responds with `200 OK`. Crucially, it includes an `x-payment-response` header. This header contains a token that can often be used as an `X-Payment` value for future requests to the same or related resources if the server supports credit reuse.
```log
Response: 200
Response Headers: {
  "content-type": "application/json; charset=utf-8",
  "payment-protocol": "X402",
  "x-payment-response": "eyJzdWNjZXNzIjp0cnVlLCJl...(this is your credit for future use)...",
  ...
}
Response Data: {
  "download_url": "https://...r2.cloudflarestorage.com/.../bitcoin-paper.pdf?X-Amz...",
  "filename": "bitcoin-paper.pdf"
}
```

**Using the Download URL:**
If the response data includes a `download_url` (as shown above, common for file routes), you can copy this URL and paste it into your web browser to download the file. The browser should then suggest the `filename` provided in the response (e.g., "bitcoin-paper.pdf").

### 2. Using an Existing Payment Credit

If you have a valid payment token from the `x-payment-response` header of a previous successful transaction, you can use it by providing it with the `--payment-header` flag. This attempts to use the existing credit, potentially bypassing a new payment.

Running the following command with your payment token:
```bash
npm run client -- http://localhost:8080/ElMbSMtY9v --payment-header "eyJzdWNjZXNzIjp0cnVlLCJl...(value from a previous x-payment-response)..."
```

Will produce output similar to this:

The client makes the request, including your provided `X-Payment` header from the start:
```log
> tsx index.ts http://localhost:8080/ElMbSMtY9v --payment-header "eyJzdWNjZXNzIjp0cnVlLCJl..."
Using TESTNET (Base Sepolia) chain
Making a GET request to: http://localhost:8080/ElMbSMtY9v
Including X-Payment header: eyJzdWNjZXNzIjp0cnVlLCJl...

Starting Request: {
  ...
  "headers": {
    "Accept": "application/json, text/plain, */*",
    "X-Payment": "eyJzdWNjZXNzIjp0cnVlLCJl...(your provided payment header value)..."
  },
  "baseURL": "http://localhost:8080/ElMbSMtY9v",
  "method": "get",
  ...
}
```
If the server accepts the credit, it responds directly with `200 OK`:
```log
Response: 200
Response Headers: {
  "content-type": "application/json; charset=utf-8",
  // The x-payment-response header may or may not be present, depending on server logic.
  ...
}
Response Data: {
  "download_url": "https://...r2.cloudflarestorage.com/.../bitcoin-paper.pdf?X-Amz...",
  "filename": "bitcoin-paper.pdf"
}
```
This demonstrates how providing a valid payment token can lead to a direct successful response, consuming the previously established credit.
