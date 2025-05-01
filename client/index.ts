import { config } from "dotenv";
import { createWalletClient, http, publicActions } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { withPaymentInterceptor } from "x402-axios";
import axios from "axios";
import { baseSepolia } from "viem/chains";

config();

const { PRIVATE_KEY } = process.env;
const targetUrl = process.argv[2]; // Get URL from command line argument

if (!PRIVATE_KEY) {
  console.error("Missing required environment variable: PRIVATE_KEY");
  process.exit(1);
}

if (!targetUrl) {
  console.error("Missing required command line argument: targetUrl");
  process.exit(1);
}

const account = privateKeyToAccount(PRIVATE_KEY as `0x${string}`);
const client = createWalletClient({
  account,
  transport: http(),
  chain: baseSepolia,
}).extend(publicActions);

const api = withPaymentInterceptor(
  axios.create({
    baseURL: targetUrl, // Use the command line URL directly
  }),
  client,
);

api
  .get("") // Make GET request to the base URL
  .then(response => {
    console.log("Response Headers:", response.headers);
    console.log("Response Data:", response.data);
  })
  .catch(error => {
    console.error("Error:", error.response?.data?.error);
  });
