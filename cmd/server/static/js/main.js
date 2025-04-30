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

// Clipboard copy handler
document.addEventListener('click', async function(e) {
    const copyButton = e.target.closest('.copy-btn');
    if (!copyButton) return;
    
    const url = copyButton.getAttribute('data-url');
    
    try {
        await navigator.clipboard.writeText(url);
        
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
}); 