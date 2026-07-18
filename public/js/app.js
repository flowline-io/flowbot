// Alpine.js shared data stores and utilities
document.addEventListener('alpine:init', () => {
  Alpine.store('toasts', []);
});

// Toast notification system - used by pipeline-editor.js and other components
// eslint-disable-next-line no-unused-vars
function showToast(message, type) {
  type = type || 'info';
  var container = document.getElementById('toast-container');
  if (!container) return;

  var item = document.createElement('div');
  item.className = 'toast-item toast-' + type;
  item.textContent = message;
  item.setAttribute('role', 'status');

  container.appendChild(item);

  // Errors often include longer diagnostics; give more reading time.
  var ttl = type === 'error' ? 8000 : 4000;
  setTimeout(function () {
    item.classList.add('toast-removing');
    setTimeout(function () {
      if (item.parentNode) item.parentNode.removeChild(item);
    }, 300);
  }, ttl);
}

// Bridge HTMX HX-Trigger {"showToast": {...}} events to the toast UI.
// Listen on document (not body): app.js loads in <head> before body exists.
document.addEventListener('showToast', function (evt) {
  var d = evt.detail || {};
  showToast(d.message || '', d.type || 'info');
});
