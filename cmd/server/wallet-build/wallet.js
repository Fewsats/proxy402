// ============================================================================
// REOWN APPKIT WALLET - X402 Payment Library
// ============================================================================
//
// SETUP:
// 1. Include: <script src="/static/js/wallet-reown-bundle.umd.js"></script>
// 2. Include: <link rel="stylesheet" href="/static/css/wallet.css">
// 3. Add buttons: <button class="wallet-connect btn btn-primary">Connect Wallet</button>
//                 <button class="wallet-pay btn btn-success" data-payment="">Pay</button>
//
// USAGE:
// Connect wallet using Connect Wallet button (becomes "Manage Wallet" when connected).
// Set data-payment attribute on Pay button with JSON response from X402 endpoint.
// Click Pay button to automatically create signature with connected wallet, auto-switch
// networks if needed, and generate X-Payment header for proof of payment to X402 endpoint.
// Code emits 'wallet-payment-response' event with payment results for calling code.
//
// ============================================================================

import { createAppKit } from '@reown/appkit';
import { WagmiAdapter } from '@reown/appkit-adapter-wagmi';
import { mainnet, base, baseSepolia } from '@reown/appkit/networks';

console.log('🔌 Loading Reown AppKit wallet...');

// Project configuration
const projectId = 'ff34950ff2d2b0caf78f3d3e0dde8735';

// AppKit instance
let modal = null;

// ========================================================================
// APPKIT SETUP - Following exact docs pattern
// ========================================================================

function initAppKit() {
    console.log('🔧 Initializing AppKit...');
    
    try {
        // Create Wagmi adapter - include mainnet to avoid "unsupported" errors
        const wagmiAdapter = new WagmiAdapter({
            projectId,
            networks: [base, baseSepolia, mainnet]
        });
        
        // Metadata - exact docs pattern
        const metadata = {
            name: 'Proxy402',
            description: 'Payment required proxy',
            url: 'https://proxy402.com',
            icons: ['/static/img/logo.svg']
        };
        
        // Create AppKit - include mainnet but prefer Base
        modal = createAppKit({
            adapters: [wagmiAdapter],
            networks: [base, baseSepolia, mainnet],
            defaultNetwork: base,
            metadata,
            projectId,
            features: {
                analytics: false
            }
        });
        
        console.log('✅ AppKit initialized');
        console.log('Modal instance:', modal);
        return true;
    } catch (error) {
        console.error('❌ AppKit initialization failed:', error);
        return false;
    }
}

// ========================================================================
// SIMPLE BUTTON HANDLERS
// ========================================================================

function setupButtons() {
    // Connect buttons
    document.querySelectorAll('.wallet-connect').forEach(btn => {
        btn.onclick = () => {
            if (modal) modal.open();
        };
    });
    
    // Payment buttons
    document.querySelectorAll('.wallet-pay').forEach(btn => {
        btn.onclick = () => handlePayment(btn, btn.getAttribute('data-payment'));
    });
    
    // Listen for connection state changes to update button text
    if (modal) {
        modal.subscribeAccount((account) => {
            const isConnected = account?.isConnected || false;
            document.querySelectorAll('.wallet-connect').forEach(btn => {
                btn.textContent = isConnected ? 'Manage Wallet' : 'Connect Wallet';
            });
        });
    }
    
    console.log('✅ Button handlers set up');
}

// ========================================================================
// X402 PAYMENT PROCESSING
// ========================================================================

function createCurlCommand(header, resource) {
return `curl -H "X-Payment: ${header}" ${resource}`;
}

function disableButton(btn) {
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner"></span> Paying...';
}

function enableButton(btn) {
    btn.disabled = false;
    btn.innerHTML = 'Pay';
}

async function handlePayment(btn, paymentData) {
    disableButton(btn);

    console.log('💳 Pay button clicked');
    
    // Check wallet connection
    const account = modal.getAccount();
    if (!account?.isConnected) {
        alert('Please connect your wallet first');
        enableButton(btn);
        return;
    }
    
    if (!paymentData || paymentData.trim() === '') {
        alert('No payment data available - make a request that returns a 402 first');
        enableButton(btn);
        return;
    }
    
    let x402Header = null;
    let payment = null;
    try {
        payment = JSON.parse(paymentData);
        console.log('💳 Processing payment:', payment);
        
        // Step 1: Sign the payment
        const signResult = await signPayment(payment, account);
        
        // Step 2: Create X402 header
        x402Header = createX402Header(signResult, payment);
        
        // Step 3: Make request with payment
        console.log('🚀 Making request with payment:', payment);
        const response = await fetch(payment.resource, {
            headers: { 'X-PAYMENT': x402Header }
        });
        
        // Send event with payment response
        document.dispatchEvent(new CustomEvent('wallet-payment-response', {
            detail: { 
                response, 
                success: response.ok,
                paymentHeader: x402Header,
                curlCommand: createCurlCommand(x402Header, payment.resource)
            }
        }));
        
    } catch (error) {
        console.error('❌ Payment failed:', error);
        
        // Send event with error
        document.dispatchEvent(new CustomEvent('wallet-payment-response', {
            detail: { 
                error: error.message, 
                success: false,
                paymentHeader: x402Header,
                curlCommand: x402Header && payment?.resource ? createCurlCommand(x402Header, payment.resource) : ''
            }
        }));
    } finally {
        enableButton(btn);
    }
}

function getUSDCDomainInfo(network, assetAddress) {
    // USDC domain info varies by network - TODO: verify actual domain names
    const usdcDomains = {
        "base": {
            "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913": { name: "USD Coin", version: "2" }
        },
        "base-sepolia": {
            "0x036CbD53842c5426634e7929541eC2318f3dCF7e": { name: "USDC", version: "2" }
        }
    };
    
    
    const domainInfo = usdcDomains[network]?.[assetAddress];
    if (!domainInfo) {
        console.warn(`Unknown USDC asset: ${assetAddress} on ${network}, using default`);
        return { name: "USD Coin", version: "2" };
    }
    
    return domainInfo;
}

async function signPayment(paymentReqs, account) {
    console.log('✏️ Signing payment...');
    
    // Switch to required network if needed
    await switchNetwork(paymentReqs.network);
    
    // Get current chain ID properly
    const provider = modal.getWalletProvider();
    const currentChainIdHex = await provider.request({ method: 'eth_chainId' });
    const currentChainId = parseInt(currentChainIdHex, 16);
    
    console.log('🔍 EIP-712 data check:', {
        account: account.address,
        payTo: paymentReqs.payTo,
        asset: paymentReqs.asset,
        maxAmountRequired: paymentReqs.maxAmountRequired,
        chainId: currentChainId
    });
    
    // Get USDC domain info based on network and asset
    const usdcDomain = getUSDCDomainInfo(paymentReqs.network, paymentReqs.asset);
    
    // Create EIP-712 typed data for USDC authorization
    const typedData = {
        types: {
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
        },
        domain: {
            name: usdcDomain.name,
            version: usdcDomain.version,
            chainId: currentChainId,
            verifyingContract: paymentReqs.asset,
        },
        primaryType: "TransferWithAuthorization",
        message: {
            from: account.address,
            to: paymentReqs.payTo,
            value: String(paymentReqs.maxAmountRequired),
            validAfter: String(Math.floor(Date.now() / 1000) - 5),
            validBefore: String(Math.floor(Date.now() / 1000) + Number(paymentReqs.maxTimeoutSeconds || 300)),
            nonce: '0x' + [...crypto.getRandomValues(new Uint8Array(32))].map(b => b.toString(16).padStart(2, '0')).join(''),
        },
    };
    
    console.log('📝 Final typed data:', JSON.stringify(typedData, null, 2));
    
    // Sign using AppKit's provider
    const signature = await provider.request({
        method: 'eth_signTypedData_v4',
        params: [account.address, JSON.stringify(typedData)],
    });
    
    return { signature, authorization: typedData.message };
}

function createX402Header(signResult, paymentReqs) {
    console.log('📝 Creating X402 payment header...');
    
    const payment = {
        x402Version: 1,
        scheme: paymentReqs.scheme,
        network: paymentReqs.network,
        payload: {
            signature: signResult.signature,
            authorization: signResult.authorization,
        },
    };
    
    return btoa(JSON.stringify(payment));
}

async function switchNetwork(targetNetwork) {
    const networkMap = {
        "base": 8453,
        "base-sepolia": 84532
    };
    
    const requiredChainId = networkMap[targetNetwork];
    if (!requiredChainId) {
        throw new Error(`Unsupported network: ${targetNetwork}`);
    }
    
    const account = modal.getAccount();
    console.log(`🔍 Network check: target=${targetNetwork}, requiredChainId=${requiredChainId}, currentChainId=${account.chainId}, type=${typeof account.chainId}`);
    
    if (Number(account.chainId) === requiredChainId) {
        console.log('✅ Already on correct network');
        return;
    }
    
    console.log(`🔄 Switching to network: ${targetNetwork} (${requiredChainId})`);
    
    // Use direct provider method instead of AppKit's switchNetwork
    const provider = modal.getWalletProvider();
    const hexChainId = `0x${requiredChainId.toString(16)}`;
    
    try {
        await provider.request({
            method: 'wallet_switchEthereumChain',
            params: [{ chainId: hexChainId }],
        });
    } catch (error) {
        if (error.code === 4902) {
            // Network not in wallet, add it
            const networks = {
                "base": {
                    chainId: hexChainId,
                    chainName: 'Base',
                    rpcUrls: ['https://mainnet.base.org'],
                    blockExplorerUrls: ['https://basescan.org'],
                    nativeCurrency: { name: 'ETH', symbol: 'ETH', decimals: 18 }
                },
                "base-sepolia": {
                    chainId: hexChainId,
                    chainName: 'Base Sepolia',
                    rpcUrls: ['https://sepolia.base.org'],
                    blockExplorerUrls: ['https://sepolia.basescan.org'],
                    nativeCurrency: { name: 'ETH', symbol: 'ETH', decimals: 18 }
                }
            };
            
            await provider.request({
                method: 'wallet_addEthereumChain',
                params: [networks[targetNetwork]],
            });
        } else {
            throw error;
        }
    }
}

// ========================================================================
// INITIALIZATION
// ========================================================================

async function initialize() {
    console.log('🚀 Initializing Reown wallet...');
    
    // Initialize AppKit
    if (!initAppKit()) {
        console.error('❌ Failed to initialize AppKit');
        return;
    }
    
    // Set up button handlers
    setupButtons();
    
    console.log('✅ Reown wallet ready!');
}

// ========================================================================
// GLOBAL API
// ========================================================================

// Expose the initialization function globally
window.ReownWallet = {
    init: initialize,
    modal: () => modal
};

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initialize);
} else {
    initialize();
}

console.log('📦 Reown wallet script loaded');