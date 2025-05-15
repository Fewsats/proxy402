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

Execute the client script using `npm run client`, providing the *full URL* of the Proxy402 paid route you want to access as a command-line argument. You can provide additional options to customize the request.

```bash
npm run client -- <proxy402_paid_route_url> [--method GET|POST] [--payment-header <value>] [--verbose]
```

**Examples:**

*   Make a GET request (default):
    ```bash
    npm run client -- http://localhost:8080/aBc1De2
    ```
*   Make a POST request:
    ```bash
    npm run client -- http://localhost:8080/xyz789 --method POST
    ```
*   Using an existing X-Payment token:
    ```bash
    npm run client -- http://localhost:8080/someRoute --payment-header "your_X_Payment_header_value_here"
    ```
*   Enable verbose mode for detailed request/response information:
    ```bash
    npm run client -- http://localhost:8080/someRoute --verbose
    ```

The client will attempt to make the request, automatically handling the x402 payment flow using the private key from your `.env` file if no valid `X-Payment` header is provided.

## How It Works

The client script (`index.ts`) does the following:

1.  Loads the `PRIVATE_KEY` from the `.env` file.
2.  Reads the target URL and optional parameters from the command-line arguments.
3.  Creates a Base wallet client using the private key.
4.  Wraps an Axios instance with the `withPaymentInterceptor` from `x402-axios`, providing the wallet client.
5.  Makes a request to the specified URL using the wrapped Axios instance.
6.  The interceptor handles any x402 challenges by generating and sending the necessary payment.
7.  Prints the essential response information upon success, or an error message if the request fails.

## Example Walkthrough

This section illustrates the client's behavior with and without using an existing payment token, using illustrative snippets from actual client logs.

### 1. Initial Payment for a Resource

When you access a paid route for the first time without providing an existing payment token, `x402-axios` automatically handles the payment challenge.

Running the following command:
```bash
npm run client -- http://localhost:8080/ElMbSMtY9v
```

Will produce output similar to this:

First, the client makes a request without any payment information:
```log
> tsx index.ts http://localhost:8080/ElMbSMtY9v
Using TESTNET (Base Sepolia) chain
Making a GET request to: http://localhost:8080/ElMbSMtY9v
Request Headers: { "Accept": "application/json, text/plain, */*" }
```
The server responds with a `402 Payment Required` status:
```log
Response Error: 402
```
`x402-axios` then automatically constructs and sends a new request including the `X-PAYMENT` header:
```log
Request Headers: {
  "Accept": "application/json, text/plain, */*",
  "X-PAYMENT": "eyJ4NDAyVm...(auto-generated payment header value)...",
  "Access-Control-Expose-Headers": "X-PAYMENT-RESPONSE"
}
```
Upon successful payment, the server responds with `200 OK`:
```log
Response Status: 200
Response Data: {
  "download_url": "https://...r2.cloudflarestorage.com/.../bitcoin-paper.pdf?X-Amz...",
  "filename": "bitcoin-paper.pdf"
}
```

**Important Note:** When the server responds to a successful payment request, it includes an `X-Payment` header in the response. This is the token you should save and reuse for future requests using the `--payment-header` flag.

**Using the Download URL:**
If the response data includes a `download_url` (as shown above, common for file routes), you can copy this URL and paste it into your web browser to download the file. The browser should then suggest the `filename` provided in the response (e.g., "bitcoin-paper.pdf").

### 2. Using an Existing Payment Token

If you have a valid payment token from a previous successful transaction, you can use it by providing it with the `--payment-header` flag. This attempts to use the existing token, potentially bypassing a new payment.

Running the following command with your payment token:
```bash
npm run client -- http://localhost:8080/ElMbSMtY9v --payment-header "eyJhbGciOi...(your X-Payment token)..."
```

Will produce output similar to this:

The client makes the request, including your provided `X-Payment` header from the start:
```log
> tsx index.ts http://localhost:8080/ElMbSMtY9v --payment-header "eyJhbGciOi..."
Using TESTNET (Base Sepolia) chain
Making a GET request to: http://localhost:8080/ElMbSMtY9v
Including X-Payment header: eyJhbGciOi...
Request Headers: {
  "Accept": "application/json, text/plain, */*",
  "X-Payment": "eyJhbGciOi...(your provided payment header value)..."
}
```
If the server accepts the token, it responds directly with `200 OK`:
```log
Response Status: 200
Response Data: {
  "download_url": "https://...r2.cloudflarestorage.com/.../bitcoin-paper.pdf?X-Amz...",
  "filename": "bitcoin-paper.pdf"
}
```
This demonstrates how providing a valid payment token can lead to a direct successful response, avoiding the need for a new payment transaction.

### Verbose Mode

If you want to see more detailed information about the request and response, you can use the `--verbose` flag:

```bash
npm run client -- http://localhost:8080/ElMbSMtY9v --verbose
```

This will show complete request and response details, which can be helpful for debugging or understanding the payment protocol more deeply.
