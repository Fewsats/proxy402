import { config } from "dotenv";
import { createWalletClient, http, publicActions } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { withPaymentInterceptor } from "x402-axios";
import axios from "axios";
import { baseSepolia } from "viem/chains";

config();

const { PRIVATE_KEY } = process.env;
const targetUrl = process.argv[2]; // Get URL from command line argument
const methodArg = process.argv[3]; // Get optional method argument

if (!PRIVATE_KEY) {
  console.error("Missing required environment variable: PRIVATE_KEY");
  process.exit(1);
}

if (!targetUrl) {
  console.error("Missing required command line argument: targetUrl");
  process.exit(1);
}

// Determine the HTTP method, default to GET
const httpMethod = (methodArg?.toUpperCase() === 'POST') ? 'POST' : 'GET';
console.log(`Making a ${httpMethod} request to: ${targetUrl}`);

const account = privateKeyToAccount(PRIVATE_KEY as `0x${string}`);
const client = createWalletClient({
  account,
  transport: http(),
  chain: baseSepolia,
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
  // Use the specified or default method
  .request({ method: httpMethod, url: '' })
  .then(response => {
    console.log("Response Headers:", response.headers);
    console.log("Response Data:", response.data);
  })
  .catch(error => {
    console.error("Full Error Object:", JSON.stringify(error, null, 2));
    console.error("Error:", error.response?.data?.error || error.message || 'Unknown error occurred');
  });
