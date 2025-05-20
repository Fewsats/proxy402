import {
    createWalletClient,
    createPublicClient,
    http,
    custom,
    toHex,
} from 'https://esm.sh/viem'

import {
    createConfig,
    connect,
    disconnect,
    signMessage,
    getBalance,
} from 'https://esm.sh/@wagmi/core'

import { injected, coinbaseWallet } from 'https://esm.sh/@wagmi/connectors'

import { base, baseSepolia } from 'https://esm.sh/viem/chains'

const authorizationTypes = {
    EIP712Domain: [
        { name: "name", type: "string" },
        { name: "version", type: "string" },
        { name: "chainId", type: "uint256" },
        { name: "verifyingContract", type: "address" },
    ],
    TransferWithAuthorization: [
        { name: "from", type: "address" },
        { name: "to", type: "address" },
        { name: "value", type: "uint256" },
        { name: "validAfter", type: "uint256" },
        { name: "validBefore", type: "uint256" },
        { name: "nonce", type: "bytes32" },
    ],
};

// USDC ABI for version function
const usdcABI = [{
    "inputs": [],
    "name": "version",
    "outputs": [{ "internalType": "string", "name": "", "type": "string" }],
    "stateMutability": "view",
    "type": "function"
}];

window.x402.utils = {
    createNonce: () => {
        return toHex(crypto.getRandomValues(new Uint8Array(32)));
    },
    safeBase64Encode: (data) => {
        if (typeof window !== "undefined") {
            return window.btoa(data);
        }
        return Buffer.from(data).toString("base64");
    },
    getUsdcAddressForChain: (chainId) => {
        return window.x402.config.chainConfig[chainId.toString()].usdcAddress;
    },
    getNetworkId: (network) => {
        const chainId = window.x402.config.networkToChainId[network];
        if (!chainId) {
            throw new Error('Unsupported network: ' + network);
        }
        return chainId;
    },
    getVersion: async (publicClient, usdcAddress) => {
        const version = await publicClient.readContract({
            address: usdcAddress,
            abi: usdcABI,
            functionName: "version"
        });
        return version;
    },
    encodePayment: (payment) => {
        const safe = {
            ...payment,
            payload: {
                ...payment.payload,
                authorization: Object.fromEntries(
                    Object.entries(payment.payload.authorization).map(([key, value]) => [
                        key,
                        typeof value === "bigint" ? value.toString() : value,
                    ])
                ),
            },
        };
        return window.x402.utils.safeBase64Encode(JSON.stringify(safe));
    },
    createPaymentHeader: async (client, publicClient) => {
        const payment = await window.x402.utils.createPayment(client, publicClient);
        return window.x402.utils.encodePayment(payment);
    },
}

window.x402.utils.signAuthorization = async (walletClient, authorizationParameters, paymentRequirements, publicClient) => {
    const chainId = window.x402.utils.getNetworkId(paymentRequirements.network);
    const name = paymentRequirements.extra?.name ?? window.x402.config.chainConfig[chainId].usdcName;
    const erc20Address = paymentRequirements.asset;
    const version = paymentRequirements.extra?.version ?? await window.x402.utils.getVersion(publicClient, erc20Address);
    const { from, to, value, validAfter, validBefore, nonce } = authorizationParameters;
    const data = {
        account: walletClient.account,
        types: authorizationTypes,
        domain: {
            name,
            version,
            chainId,
            verifyingContract: erc20Address,
        },
        primaryType: "TransferWithAuthorization",
        message: {
            from,
            to,
            value,
            validAfter,
            validBefore,
            nonce,
        },
    };

    const signature = await walletClient.signTypedData(data);

    return {
        signature,
    };
}

window.x402.utils.createPayment = async (client, publicClient) => {
    if (!window.x402.paymentRequirements) {
        throw new Error('Payment requirements not initialized');
    }

    const nonce = window.x402.utils.createNonce();
    const version = await window.x402.utils.getVersion(publicClient, window.x402.utils.getUsdcAddressForChain(window.x402.utils.getNetworkId(window.x402.paymentRequirements.network)));
    const from = client.account.address;

    const validAfter = BigInt(
        Math.floor(Date.now() / 1000) - 5 // 1 block (2s) before to account for block timestamping
    );
    const validBefore = BigInt(
        Math.floor(Date.now() / 1000 + window.x402.paymentRequirements.maxTimeoutSeconds)
    );

    const { signature } = await window.x402.utils.signAuthorization(
        client,
        {
            from,
            to: window.x402.paymentRequirements.payTo,
            value: window.x402.paymentRequirements.maxAmountRequired,
            validAfter,
            validBefore,
            nonce,
            version,
        },
        window.x402.paymentRequirements,
        publicClient
    );

    return {
        x402Version: 1,
        scheme: window.x402.paymentRequirements.scheme,
        network: window.x402.paymentRequirements.network,
        payload: {
            signature,
            authorization: {
                from,
                to: window.x402.paymentRequirements.payTo,
                value: window.x402.paymentRequirements.maxAmountRequired,
                validAfter,
                validBefore,
                nonce,
            },
        },
    };
}


async function initializeApp() {
    const x402 = window.x402;
    const wagmiConfig = createConfig({
        chains: [base, baseSepolia],
        connectors: [
            coinbaseWallet({ appName: 'Create Wagmi' }),
            injected(),
        ],
        transports: {
            [base.id]: http(),
            [baseSepolia.id]: http(),
        },
    });

    // DOM Elements
    const connectWalletBtn = document.getElementById('connect-wallet');
    const paymentSection = document.getElementById('payment-section');
    const payButton = document.getElementById('pay-button');
    const statusDiv = document.getElementById('status');

    if (!connectWalletBtn || !paymentSection || !payButton || !statusDiv) {
        // console.error('Required DOM elements not found');
        return;
    }

    let walletClient = null;
    const chain = x402.isTestnet ? baseSepolia : base;

    const publicClient = createPublicClient({
        chain,
        transport: custom(window.ethereum),
    });

    // Connect wallet handler
    connectWalletBtn.addEventListener('click', async () => {
        // If wallet is already connected, disconnect it
        if (walletClient) {
            try {
                await disconnect(wagmiConfig);
                walletClient = null;
                connectWalletBtn.textContent = 'Connect Wallet';
                paymentSection.classList.add('hidden');
                statusDiv.textContent = 'Wallet disconnected';
                return;
            } catch (error) {
                statusDiv.textContent = 'Failed to disconnect wallet';
                return;
            }
        }

        try {
            statusDiv.textContent = 'Connecting wallet...';

            const result = await connect(wagmiConfig, {
                connector: injected(),
                chainId: chain.id,
            });
            if (!result.accounts?.[0]) {
                throw new Error('Please select an account in your wallet');
            }
            walletClient = createWalletClient({
                account: result.accounts[0],
                chain,
                transport: custom(window.ethereum)
            });

            const address = result.accounts[0]

            connectWalletBtn.textContent = `${address.slice(0, 6)}...${address.slice(-4)}`;
            paymentSection.classList.remove('hidden');
            statusDiv.textContent =
                'Wallet connected! You can now proceed with payment.';
        } catch (error) {
            console.error('Connection error:', error);
            statusDiv.textContent =
                error instanceof Error ? error.message : 'Failed to connect wallet';
            // Reset UI state
            connectWalletBtn.textContent = 'Connect Wallet';
            paymentSection.classList.add('hidden');
        }
    });

    // Payment handler
    payButton.addEventListener('click', async () => {
        if (!walletClient) {
            statusDiv.textContent = 'Please connect your wallet first';
            return;
        }

        try {
            const usdcAddress = window.x402.config.chainConfig[chain.id].usdcAddress;
            try {
                statusDiv.textContent = 'Checking USDC balance...';
                const balance = await publicClient.readContract({
                    address: usdcAddress,
                    abi: [{
                        inputs: [{ internalType: "address", name: "account", type: "address" }],
                        name: "balanceOf",
                        outputs: [{ internalType: "uint256", name: "", type: "uint256" }],
                        stateMutability: "view",
                        type: "function"
                    }],
                    functionName: "balanceOf",
                    args: [walletClient.account.address]
                });

                if (balance === 0n) {
                    statusDiv.textContent = `Your USDC balance is 0. Please make sure you have USDC tokens on ` + (x402.isTestnet ? 'Base Sepolia' : 'Base') + '.';
                    return;
                }

                statusDiv.textContent = 'Creating payment signature...';

                const paymentHeader = await x402.utils.createPaymentHeader(walletClient, publicClient);

                statusDiv.textContent = 'Requesting content with payment...';

                const response = await fetch(x402.currentUrl, {
                    headers: {
                        'X-PAYMENT': paymentHeader,
                        'Access-Control-Expose-Headers': 'X-PAYMENT-RESPONSE',
                    },
                });

                if (response.ok) {
                    const jsonResponse = await response.json();
                    if (jsonResponse.download_url) {
                        window.location.href = jsonResponse.download_url;
                    } else {
                        statusDiv.textContent = 'Download failed: ' + response.statusText;
                        throw new Error('Download failed: ' + response.statusText);
                    }
                }
            } catch (error) {
                statusDiv.textContent = error instanceof Error ? error.message : 'Failed to check USDC balance';
            }
        } catch (error) {
            statusDiv.textContent = error instanceof Error ? error.message : 'Payment failed';
        }
    });
}

window.addEventListener('load', initializeApp);