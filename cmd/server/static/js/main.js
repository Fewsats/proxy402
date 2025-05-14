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

// Initialize tab switching and file upload functionality
document.addEventListener('DOMContentLoaded', function() {
    // Tab switching functionality
    initTabs();
    
    // File upload functionality
    initFileUpload();
});

// Initialize tab switching
function initTabs() {
    const tabs = document.querySelectorAll('.tab');
    if (!tabs.length) return;
    
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            // Remove active class from all tabs
            tabs.forEach(t => t.classList.remove('active'));
            
            // Add active class to clicked tab
            tab.classList.add('active');
            
            // Get target section
            const targetId = tab.getAttribute('data-target');
            
            // Hide all sections
            document.querySelectorAll('#url-section, #file-section').forEach(section => {
                section.style.display = 'none';
            });
            
            // Show target section
            const targetSection = document.getElementById(targetId);
            if (targetSection) {
                targetSection.style.display = 'block';
            }
            
            // Set method to GET for file uploads and disable the dropdown
            const methodSelect = document.getElementById('method-select');
            const methodTooltip = document.getElementById('method-tooltip');
            if (methodSelect && methodTooltip) {
                if (targetId === 'file-section') {
                    methodSelect.value = 'GET';
                    methodSelect.disabled = true;
                    methodTooltip.style.display = 'inline-block';
                } else {
                    methodSelect.disabled = false;
                    methodTooltip.style.display = 'none';
                }
            }
            
            // Update submit button text
            const submitBtn = document.getElementById('submit-btn');
            if (submitBtn) {
                submitBtn.textContent = targetId === 'url-section' ? 'Add Link ' : 'Upload File ';
                // Add the spinner back
                const spinner = document.createElement('span');
                spinner.id = 'spinner';
                spinner.className = 'htmx-indicator';
                spinner.textContent = '↻';
                submitBtn.appendChild(spinner);
            }
            
            // Update required fields
            const urlInput = document.getElementById('target-url');
            const fileInput = document.getElementById('file-input');
            if (urlInput && fileInput) {
                urlInput.required = targetId === 'url-section';
                fileInput.required = targetId === 'file-section';
            }
        });
    });
}

// Initialize file upload functionality
function initFileUpload() {
    const dropzone = document.getElementById('file-dropzone');
    const fileInput = document.getElementById('file-input');
    const filePreview = document.getElementById('file-preview');
    
    if (!dropzone || !fileInput || !filePreview) return;
    
    // Handle file selection
    function handleFile(file) {
        if (!file) return;
        
        const fileName = filePreview.querySelector('.file-name');
        const fileSize = filePreview.querySelector('.file-size');
        
        if (fileName && fileSize) {
            // Display file info
            fileName.textContent = file.name;
            fileSize.textContent = formatFileSize(file.size);
        }
        
        // Show preview, hide dropzone
        filePreview.style.display = 'flex';
        dropzone.style.display = 'none';
    }
    
    // Prevent default browser drag behavior
    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        dropzone.addEventListener(eventName, e => {
            e.preventDefault();
            e.stopPropagation();
        });
    });
    
    // Highlight effect
    ['dragenter', 'dragover'].forEach(eventName => {
        dropzone.addEventListener(eventName, () => dropzone.classList.add('dragover'));
    });
    
    ['dragleave', 'drop'].forEach(eventName => {
        dropzone.addEventListener(eventName, () => dropzone.classList.remove('dragover'));
    });
    
    // Handle dropped files
    dropzone.addEventListener('drop', e => {
        if (e.dataTransfer.files.length) {
            fileInput.files = e.dataTransfer.files;
            handleFile(e.dataTransfer.files[0]);
        }
    });
    
    // Handle file selection via input
    fileInput.addEventListener('change', () => {
        if (fileInput.files.length) {
            handleFile(fileInput.files[0]);
        }
    });
    
    // Open file dialog when clicking on dropzone
    dropzone.addEventListener('click', () => fileInput.click());
    
    // Remove file button
    const removeButton = filePreview.querySelector('.remove-file');
    if (removeButton) {
        removeButton.addEventListener('click', () => {
            fileInput.value = '';
            filePreview.style.display = 'none';
            dropzone.style.display = 'block';
        });
    }
}

// Format file size
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
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
            
            // Get active tab
            const activeTab = document.querySelector('.tab.active');
            const isFileUpload = activeTab?.getAttribute('data-target') === 'file-section';
            
            try {
                if (isFileUpload) {
                    await handleFileUpload();
                } else {
                    await handleUrlSubmission();
                }
            } catch (error) {
                errorDiv.textContent = error.message || 'Network error. Please try again.';
                errorDiv.style.display = 'block';
                console.error('Error submitting form:', error);
            } finally {
                if (spinner) spinner.style.display = 'none';
            }
        });
    }
    
    async function handleFileUpload() {
        const fileInput = document.getElementById('file-input');
        if (!fileInput.files.length) {
            throw new Error('Please select a file to upload.');
        }
        
        const file = fileInput.files[0];
        
        // Step 1: Create the route and get the signed upload URL
        const routePayload = {
            original_filename: file.name,
            price: document.getElementById('price-input').value,
            is_test: document.getElementById('is-test-input').checked,
            type: document.getElementById('type-select').value,
            credits: parseInt(document.getElementById('credits-input').value) || 0
        };
        
        // Send request to create the route and get the signed URL
        const routeResponse = await fetch('/files/upload', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(routePayload)
        });
        
        if (!routeResponse.ok) {
            const errorData = await routeResponse.json();
            throw new Error(errorData.error || 'Failed to create file route');
        }
        
        const routeData = await routeResponse.json();
        
        // Step 2: Upload the file to the signed URL
        const uploadResponse = await fetch(routeData.upload_url, {
            method: 'PUT',
            body: file,
            headers: {
                'Content-Type': file.type || 'application/octet-stream'
            }
        });
        
        if (!uploadResponse.ok) {
            throw new Error('Failed to upload file to storage. Please try again.');
        }
        
        // Success, reload the page to show the new file route
        errorDiv.style.display = 'none';
        form.reset();
        window.location.reload();
    }
    
    async function handleUrlSubmission() {
        // Create payload
        const payload = {
            target_url: document.getElementById('target-url').value,
            price: document.getElementById('price-input').value,
            method: document.getElementById('method-select').value,
            is_test: document.getElementById('is-test-input').checked,
            type: document.getElementById('type-select').value,
            credits: parseInt(document.getElementById('credits-input').value) || 0

        };
        
        // Send request
        const response = await fetch('/links/shrink', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(payload)
        });
        
        const data = await response.json();
        
        if (response.ok) {
            errorDiv.style.display = 'none';
            form.reset();
            window.location.reload();
        } else {
            throw new Error(data.error || 'An error occurred');
        }
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

// Initialize Lucide icons and fix .01 price
// Initialize Lucide icons, tooltips, and fix leading-dot price inputs
document.addEventListener('DOMContentLoaded', function() {
    // 1) Lucide
    if (window.lucide) {
        lucide.createIcons();
    }
    
    // 2) Enhanced tooltips
    initializeEnhancedTooltips();

    // 3) Auto-prefix leading "x" with "0.x" on price input
    const priceInput = document.getElementById('price-input');
    if (priceInput) {
        priceInput.addEventListener('blur', () => {
            const v = priceInput.value.trim();
            if (v.startsWith('.') && /^\.\d+$/.test(v)) {
                priceInput.value = '0' + v;
            }
        });
    }
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
