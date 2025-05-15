import { config } from "dotenv";
import { createWalletClient, http, publicActions } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { withPaymentInterceptor } from "x402-axios";
import axios, { AxiosRequestConfig } from "axios";
import { baseSepolia, base } from "viem/chains";

config();

// Parse command line arguments
const args = process.argv.slice(2);
let targetUrl = '';
let httpMethod = 'GET';
let network = 'base-sepolia'; // Default to testnet
let paymentHeader: string | undefined; // Variable to hold the payment header
let verbose = false; // Flag for verbose mode

// Parse arguments
for (let i = 0; i < args.length; i++) {
  const arg = args[i];
  
  if (arg === '--network' && i + 1 < args.length) {
    network = args[++i];
    continue;
  }
  
  if (arg === '--method' && i + 1 < args.length) {
    httpMethod = args[++i].toUpperCase();
    continue;
  }

  if (arg === '--payment-header' && i + 1 < args.length) { // Check for payment header flag
    paymentHeader = args[++i];
    continue;
  }
  
  if (arg === '--verbose') {
    verbose = true;
    continue;
  }
  
  // If not a flag, assume it's the target URL
  if (!arg.startsWith('--') && !targetUrl) {
    targetUrl = arg;
  }
}

const { PRIVATE_KEY } = process.env;

if (!PRIVATE_KEY) {
  console.error("Missing required environment variable: PRIVATE_KEY");
  process.exit(1);
}

if (!targetUrl) {
  console.error("Missing required target URL");
  console.error("Usage: npm run client -- <url> [--method GET|POST] [--network base-sepolia|base-mainnet] [--payment-header <header_value>] [--verbose]");
  process.exit(1);
}

// Parse network string to chain
let chain;
if (network === 'base-mainnet' || network === 'base') {
  chain = base;
  console.log('Using MAINNET (Base) chain');
} else {
  // Default to testnet for any other value
  chain = baseSepolia;
  console.log('Using TESTNET (Base Sepolia) chain');
}

console.log(`Making a ${httpMethod} request to: ${targetUrl}`);
if (paymentHeader) {
  console.log(`Including X-Payment header: ${paymentHeader}`);
}

const account = privateKeyToAccount(PRIVATE_KEY as `0x${string}`);
const client = createWalletClient({
  account,
  transport: http(),
  chain,
}).extend(publicActions);

// Create a base axios instance for logging
const baseApi = axios.create({
  baseURL: targetUrl,
});

// Log requests before they are sent
baseApi.interceptors.request.use(request => {
  if (verbose) {
    console.log('Starting Request (Full):', JSON.stringify(request, null, 2));
  } else {
    console.log('Request Headers:', JSON.stringify(request.headers, null, 2));
  }
  return request;
});

// Log responses or errors
baseApi.interceptors.response.use(response => {
  console.log('Response Status:', response.status);
  
  if (verbose) {
    console.log('Response Headers (Full):', JSON.stringify(response.headers, null, 2));
  } else if (response.headers['x-payment']) {
    console.log('X-Payment header:', response.headers['x-payment']);
  }
  
  return response;
}, error => {
  console.error('Response Error:', JSON.stringify(error.response?.status, null, 2));
  
  if (verbose) {
    console.error('Response Error Headers (Full):', JSON.stringify(error.response?.headers, null, 2));
  } else if (error.response?.headers['www-authenticate']) {
    console.error('Payment Required. Challenge:', error.response.headers['www-authenticate']);
  }
  
  return Promise.reject(error);
});

// Apply the payment interceptor to the logging-enabled instance
const api = withPaymentInterceptor(
  baseApi, // Use the instance with logging
  client,
);

// Prepare request config
const requestConfig: AxiosRequestConfig = {
  method: httpMethod,
  url: '', // Use empty url if baseURL is set to the full targetUrl
         // Or set url: targetUrl and baseURL: undefined if targetUrl includes path
  headers: {}
};

// Conditionally add the X-Payment header
if (paymentHeader) {
  requestConfig.headers!['X-Payment'] = paymentHeader;
}

api
  .request(requestConfig) // Use the prepared request config
  .then(response => {
    if (verbose) {
      console.log("Response Headers (Full):", response.headers);
    }
    console.log("Response Data:", response.data);
  })
  .catch(error => {
    if (verbose) {
      console.error("Full Error Object:", JSON.stringify(error, null, 2));
    }
    console.error("Error:", error.response?.data?.error || error.message || 'Unknown error occurred');
  });
