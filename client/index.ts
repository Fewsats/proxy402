import { config } from "dotenv";
import { createWalletClient, http, publicActions } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { withPaymentInterceptor } from "x402-axios";
import axios from "axios";
import { baseSepolia, base } from "viem/chains";

config();

// Parse command line arguments
const args = process.argv.slice(2);
let targetUrl = '';
let httpMethod = 'GET';
let network = 'base-sepolia'; // Default to testnet

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
  console.error("Usage: npm run client -- <url> [--method GET|POST] [--network base-sepolia|base-mainnet]");
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
  console.log('Starting Request:', JSON.stringify(request, null, 2));
  return request;
});

// Log responses or errors
baseApi.interceptors.response.use(response => {
  console.log('Response:', JSON.stringify(response.status, null, 2));
  console.log('Response Headers:', JSON.stringify(response.headers, null, 2));
  return response;
}, error => {
  console.error('Response Error:', JSON.stringify(error.response?.status, null, 2));
  console.error('Response Error Headers:', JSON.stringify(error.response?.headers, null, 2));
  return Promise.reject(error);
});

// Apply the payment interceptor to the logging-enabled instance
const api = withPaymentInterceptor(
  baseApi, // Use the instance with logging
  client,
);

api
  // Use the specified method
  .request({ method: httpMethod, url: '' })
  .then(response => {
    console.log("Response Headers:", response.headers);
    console.log("Response Data:", response.data);
  })
  .catch(error => {
    console.error("Full Error Object:", JSON.stringify(error, null, 2));
    console.error("Error:", error.response?.data?.error || error.message || 'Unknown error occurred');
  });
