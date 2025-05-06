// Helper function to format USDC values with appropriate decimal places
function formatUSDC(price) {
    const value = parseFloat(price) / 1000000;
    
    // If the value is zero, just return $0.00 USDC
    if (value === 0) {
        return '$0.00 USDC';
    }
    
    // For non-zero values, show up to 6 decimal places but trim trailing zeros
    let formattedValue;
    if (value >= 0.01) {
        // For larger values, show 2 decimal places
        formattedValue = value.toFixed(2);
    } else {
        // For very small values, show up to 6 decimal places
        formattedValue = value.toFixed(6);
        // Trim trailing zeros
        formattedValue = formattedValue.replace(/\.?0+$/, '');
    }
    
    return `$${formattedValue} USDC`;
}

// Function to format decimal values for chart display with appropriate precision
function formatDecimalValue(value) {
    if (value === 0) {
        return '0.00';
    }
    
    if (value >= 0.01) {
        return value.toFixed(2);
    } else {
        // For very small values, show up to 6 decimal places
        let formatted = value.toFixed(6);
        // Trim trailing zeros
        return formatted.replace(/\.?0+$/, '');
    }
}

// Function to format values consistently in chart legends (fixed decimal places)
function formatChartLegendValue(value) {
    // Always show 6 decimal places for consistency in the chart legend
    return value.toFixed(6);
}

// Form submission handler
document.addEventListener('DOMContentLoaded', function() {
    const form = document.getElementById('link-form');
    const errorDiv = document.getElementById('form-error');
    const spinner = document.getElementById('spinner');
    
    if (form) {
        form.addEventListener('submit', async function(e) {
            e.preventDefault();
            
            // Show spinner
            if (spinner) spinner.style.display = 'inline-block';
            
            // Get form data
            const targetUrl = document.getElementById('target-url').value;
            const price = document.getElementById('price-input').value;
            const method = document.getElementById('method-select').value;
            const isTest = Boolean(document.getElementById('is-test-input').checked);
            
            // Create payload
            const payload = {
                target_url: targetUrl,
                price: price,
                method: method,
                is_test: isTest
            };
            
            try {
                // Send request
                const response = await fetch('/links/shrink', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(payload)
                });
                
                // Parse response
                const data = await response.json();
                console.log('Response data:', data); // Log response for debugging
                
                if (response.ok) {
                    // Success - hide error and reset form
                    errorDiv.textContent = '';
                    errorDiv.style.display = 'none';
                    form.reset();
                    window.location.reload();
                } else {
                    // Show error message
                    errorDiv.textContent = data.error || 'An error occurred';
                    errorDiv.style.display = 'block';
                }
            } catch (error) {
                errorDiv.textContent = 'Network error. Please try again.';
                errorDiv.style.display = 'block';
                console.error('Error submitting form:', error);
            } finally {
                // Hide spinner
                if (spinner) spinner.style.display = 'none';
            }
        });
    }
});

// Copy value handler for settings page and other special copy buttons
document.addEventListener('click', async function(e) {
    const copyButton = e.target.closest('.copy-btn');
    if (!copyButton) return;
    
    // Get the URL or value to copy
    const url = copyButton.getAttribute('data-url');
    const value = copyButton.getAttribute('data-value');
    const textToCopy = url || value;
    
    if (!textToCopy) return;
    
    try {
        await navigator.clipboard.writeText(textToCopy);
        
        // Show temporary "Copied!" message near the button
        const existingMsg = copyButton.querySelector('.copy-success');
        if (existingMsg) existingMsg.remove(); // Remove old message if any

        const message = document.createElement('span');
        message.classList.add('copy-success');
        message.textContent = 'Copied!';
        document.body.appendChild(message); // Append to body for positioning
        
        const rect = copyButton.getBoundingClientRect();
        message.style.top = `${window.scrollY + rect.bottom + 5}px`;
        message.style.left = `${window.scrollX + rect.left + rect.width / 2}px`;
        
        // Show, then fade out
        requestAnimationFrame(() => {
            message.style.opacity = '1';
            message.style.transform = 'translate(-50%, 0)'; // Adjust final position
        });

        setTimeout(() => {
            message.style.opacity = '0';
            setTimeout(() => message.remove(), 300);
        }, 1500);

    } catch (error) {
        console.error('Failed to copy:', error);
    }
});

// User menu toggle
document.addEventListener('DOMContentLoaded', function() {
    const trigger = document.getElementById('user-menu-trigger');
    const dropdown = document.getElementById('user-menu-dropdown');
    
    if (trigger && dropdown) {
        trigger.addEventListener('click', function(e) {
            dropdown.classList.toggle('show');
            e.stopPropagation();
        });
        
        document.addEventListener('click', function(e) {
            if (trigger && dropdown && !trigger.contains(e.target) && !dropdown.contains(e.target)) {
                dropdown.classList.remove('show');
            }
        });
    }
});

// Initialize Lucide icons
document.addEventListener('DOMContentLoaded', function() {
    if (window.lucide) {
        lucide.createIcons();
    }
    
    // Enhanced tooltips for long URLs
    initializeEnhancedTooltips();
});

// Function to initialize enhanced tooltips for long URLs
function initializeEnhancedTooltips() {
    // Get all elements with the ellipsis class
    const ellipsisElements = document.querySelectorAll('.ellipsis[data-tooltip]');
    
    ellipsisElements.forEach(element => {
        // Check if the content is being truncated
        if (element.scrollWidth > element.clientWidth) {
            const tooltipText = element.getAttribute('data-tooltip');
            
            element.addEventListener('mouseenter', function(e) {
                // Create tooltip element if it doesn't exist
                if (!document.getElementById('enhanced-tooltip')) {
                    const tooltip = document.createElement('div');
                    tooltip.id = 'enhanced-tooltip';
                    tooltip.textContent = tooltipText;
                    tooltip.style.position = 'absolute';
                    tooltip.style.zIndex = '10000';
                    tooltip.style.backgroundColor = 'rgba(30, 50, 90, 0.95)';
                    tooltip.style.color = '#ffffff';
                    tooltip.style.padding = '8px 12px';
                    tooltip.style.borderRadius = '6px';
                    tooltip.style.fontSize = '0.85rem';
                    tooltip.style.maxWidth = '600px';
                    tooltip.style.wordBreak = 'break-all';
                    tooltip.style.boxShadow = '0 2px 8px rgba(0, 0, 0, 0.3)';
                    tooltip.style.border = '1px solid rgba(120, 150, 200, 0.4)';
                    tooltip.style.pointerEvents = 'none';
                    
                    document.body.appendChild(tooltip);
                    
                    // Position the tooltip
                    const rect = element.getBoundingClientRect();
                    tooltip.style.left = (rect.left) + 'px';
                    tooltip.style.top = (window.scrollY + rect.top - tooltip.offsetHeight - 10) + 'px';
                }
            });
            
            element.addEventListener('mouseleave', function() {
                const tooltip = document.getElementById('enhanced-tooltip');
                if (tooltip) {
                    tooltip.remove();
                }
            });
        }
    });
    
    // Monitor for table updates (e.g., when sorting or filtering)
    const observer = new MutationObserver(function(mutations) {
        mutations.forEach(function(mutation) {
            if (mutation.type === 'childList') {
                initializeEnhancedTooltips();
            }
        });
    });
    
    const tableBody = document.querySelector('#links-table tbody');
    if (tableBody) {
        observer.observe(tableBody, { childList: true });
    }
} 