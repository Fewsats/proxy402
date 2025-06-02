// ============================================================================
// BETTER STACK LOGGER - Single File with Dynamic CDN Loading
// ============================================================================
//
// SETUP:
// IMPORTANT: Include this file BEFORE any other scripts that use logging
// 1. Include this file: <script src="/static/js/better-stack-logger.js"></script>
// 2. Include other scripts after: <script src="/other-scripts.js"></script>
// 3. Set your source token below
//
// USAGE:
// logInfo('Message', { data: 'optional' });
// logError('Error occurred', error);
// logWarn('Warning message');
// logDebug('Debug info');
//
// NOTE: Functions are available immediately and fallback to console if Better Stack fails
//
// ============================================================================

(function() {
    'use strict';
    
    // Configuration
    const BETTER_STACK_SOURCE_TOKEN = '2zhG9nAWv8FKTESJdDX383AC';
    const BETTER_STACK_ENDPOINT = 'https://s1329399.eu-nbg-2.betterstackdata.com';
    
    // Better Stack logger instance
    let betterStack = null;
    
    // Load Better Stack library dynamically
    function loadBetterStackCDN() {
        return new Promise((resolve, reject) => {
            const script = document.createElement('script');
            script.src = 'https://cdnjs.cloudflare.com/ajax/libs/logtail-browser/0.4.19/dist/umd/logtail.min.js';
            
            script.onload = () => {
                console.log('📦 Better Stack CDN loaded');
                resolve();
            };
            
            script.onerror = () => {
                console.error('❌ Failed to load Better Stack CDN');
                reject(new Error('CDN load failed'));
            };
            
            script.crossOrigin = 'anonymous';
            document.head.appendChild(script);
        });
    }
    
    // Initialize Better Stack logger
    async function initBetterStack() {
        try {
            await loadBetterStackCDN();
            
            betterStack = new Logtail(BETTER_STACK_SOURCE_TOKEN, {
                    endpoint: BETTER_STACK_ENDPOINT
                });
            console.log('✅ Better Stack logger initialized');
        } catch (error) {
            console.error('❌ Failed to initialize Better Stack:', error);
        }
    }
    
    // Global logging functions
    window.logInfo = function(message, data = {}) {
        console.log(`[INFO] ${message}`, data);
        if (betterStack) {
            betterStack.info(message, data);
        }
    };
    
    window.logError = function(message, error = {}) {
        console.error(`[ERROR] ${message}`, error);
        if (betterStack) {
            const errorData = {
                error: error.message || error,
                stack: error.stack,
                timestamp: new Date().toISOString()
            };
            betterStack.error(message, errorData);
        }
    };
    
    window.logWarn = function(message, data = {}) {
        console.warn(`[WARN] ${message}`, data);
        if (betterStack) {
            betterStack.info(message, { level: 'warn', ...data });
        }
    };
    
    window.logDebug = function(message, data = {}) {
        console.log(`[DEBUG] ${message}`, data);
        if (betterStack) {
            betterStack.info(message, { level: 'debug', ...data });
        }
    };
    
    // Initialize when DOM is ready or immediately if already loaded
    async function initialize() {
        await initBetterStack();
        logInfo('Better Stack logger ready', {
            userAgent: navigator.userAgent,
            url: window.location.href
        });
    }
    
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initialize);
    } else {
        initialize();
    }
    
    console.log('📦 Better Stack logger script loaded');
})();