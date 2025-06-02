// ============================================================================
// BETTER STACK LOGGER
// ============================================================================
//
// SETUP:
// 1. Include this file: <script src="/static/js/better-stack-logger.js"></script>
// 2. Initialize from template: <script>initBetterStackLogger('{{.BetterStackToken}}', '{{.BetterStackEndpoint}}');</script>
//
// USAGE:
// console.log('Message', { data: 'optional' });
// console.error('Error occurred', error);
// console.warn('Warning message');
// console.info('Info message');
//
// NOTE: Normal console methods work immediately, Better Stack enhancement added after init
//
// ============================================================================

(function() {
    'use strict';
    
    // Better Stack logger instance
    let betterStack = null;
    let isInitialized = false;
    
    // Queue for messages before initialization
    const messageQueue = [];
    
    // Store original console methods
    const originalConsole = {
        log: console.log.bind(console),
        error: console.error.bind(console),
        warn: console.warn.bind(console),
        info: console.info.bind(console)
    };
    
    // Load Better Stack library dynamically
    function loadBetterStackCDN() {
        return new Promise((resolve, reject) => {
            const script = document.createElement('script');
            script.src = 'https://cdnjs.cloudflare.com/ajax/libs/logtail-browser/0.4.19/dist/umd/logtail.min.js';
            
            script.onload = () => {
                originalConsole.log('📦 Better Stack CDN loaded');
                resolve();
            };
            
            script.onerror = () => {
                originalConsole.error('❌ Failed to load Better Stack CDN');
                reject(new Error('CDN load failed'));
            };
            
            script.crossOrigin = 'anonymous';
            document.head.appendChild(script);
        });
    }
    
    // Process queued messages
    function flushMessageQueue() {
        while (messageQueue.length > 0) {
            const { method, args } = messageQueue.shift();
            sendToBetterStack(method, args);
        }
    }
    
    // Send to Better Stack
    function sendToBetterStack(method, args) {
        if (!betterStack) return;
        
        const [message, ...data] = args;
        const logData = data.length > 0 ? data[0] : {};
        
        switch (method) {
            case 'error':
                const errorData = {
                    error: logData?.message || logData,
                    stack: logData?.stack,
                    timestamp: new Date().toISOString()
                };
                betterStack.error(message, errorData);
                break;
            case 'warn':
                betterStack.info(message, { level: 'warn', ...logData });
                break;
            case 'info':
            case 'log':
            default:
                betterStack.info(message, logData);
                break;
        }
    }
    
    // Enhanced console methods
    function createEnhancedMethod(method, originalMethod) {
        return function(...args) {
            // Always call original console method first
            originalMethod(...args);
            
            // Send to Better Stack if initialized, otherwise queue
            if (isInitialized && betterStack) {
                sendToBetterStack(method, args);
            } else {
                messageQueue.push({ method, args });
            }
        };
    }
    
    // Override console methods immediately
    console.log = createEnhancedMethod('log', originalConsole.log);
    console.error = createEnhancedMethod('error', originalConsole.error);
    console.warn = createEnhancedMethod('warn', originalConsole.warn);
    console.info = createEnhancedMethod('info', originalConsole.info);
    
    // Initialize Better Stack logger
    async function initBetterStack(token, endpoint) {
        try {
            await loadBetterStackCDN();
            
            betterStack = new Logtail(token, { endpoint });
            isInitialized = true;
            
            originalConsole.log('✅ Better Stack logger initialized');
            
            // Process any queued messages
            flushMessageQueue();
            
            // Log initialization
            betterStack.info('Better Stack logger ready', {
                userAgent: navigator.userAgent,
                url: window.location.href
            });
            
        } catch (error) {
            originalConsole.error('❌ Failed to initialize Better Stack:', error);
            // Clear queue on failure
            messageQueue.length = 0;
        }
    }
    
    // Global initialization function
    window.initBetterStackLogger = function(token, endpoint) {
        if (!token || !endpoint) {
            originalConsole.warn('⚠️ Better Stack token or endpoint missing, console logging only');
            return;
        }
        initBetterStack(token, endpoint);
    };
    
    originalConsole.log('📦 Better Stack console enhancer loaded');
})();