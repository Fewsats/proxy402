document.addEventListener('DOMContentLoaded', function () {
  const form = document.getElementById('debug-form');
  const output = document.getElementById('response-output');
  form.addEventListener('submit', async function (e) {
    e.preventDefault();
    const url = document.getElementById('target-url').value;
    const method = document.getElementById('method-select').value;
    const payment = document.getElementById('payment-header').value.trim();
    output.textContent = 'Loading...';
    try {
      const headers = {};
      if (payment) headers['X-Payment'] = payment;
      const res = await fetch(url, { method: method, headers: headers });
      const text = await res.text();
      try {
        output.textContent = JSON.stringify(JSON.parse(text), null, 2);
      } catch {
        output.textContent = text;
      }
    } catch (err) {
      output.textContent = 'Request failed: ' + err;
    }
  });
});
