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

// CSRF double-submit: cookie csrfToken + X-CSRF-Token header / form field.
window.flowbotCSRFCache = window.flowbotCSRFCache || '';

function flowbotGetCookie(name) {
  var cookieSource = document['cookie'] || '';
  var parts = cookieSource.split(';');
  for (var i = 0; i < parts.length; i++) {
    var p = parts[i].trim();
    if (p.indexOf(name + '=') === 0) {
      return decodeURIComponent(p.substring(name.length + 1));
    }
  }
  return '';
}

function flowbotCSRFToken() {
  return flowbotGetCookie('csrfToken') || window.flowbotCSRFCache || '';
}

function flowbotRefreshCSRF() {
  return fetch('/service/web/csrf-token', {
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
  })
    .then(function (res) {
      if (!res.ok) {
        throw new Error('csrf refresh failed');
      }
      return res.json();
    })
    .then(function (data) {
      window.flowbotCSRFCache = (data && data.token) || '';
      return window.flowbotCSRFCache;
    });
}

// Merge CSRF into fetch headers for cookie-authenticated mutations.
// eslint-disable-next-line no-unused-vars
function flowbotCSRFHeaders(extra) {
  var headers = {};
  if (extra) {
    Object.keys(extra).forEach(function (k) {
      headers[k] = extra[k];
    });
  }
  var tok = flowbotCSRFToken();
  if (tok) {
    headers['X-CSRF-Token'] = tok;
  }
  return headers;
}

// Ensure a CSRF token is available, then return headers for fetch mutations.
// eslint-disable-next-line no-unused-vars
function flowbotCSRFHeadersAsync(extra) {
  var tok = flowbotCSRFToken();
  if (tok) {
    return Promise.resolve(flowbotCSRFHeaders(extra));
  }
  return flowbotRefreshCSRF().then(function () {
    return flowbotCSRFHeaders(extra);
  });
}

document.addEventListener('DOMContentLoaded', function () {
  if (!flowbotCSRFToken()) {
    flowbotRefreshCSRF().catch(function () {
      /* non-fatal: mutations call flowbotCSRFHeadersAsync */
    });
  } else {
    window.flowbotCSRFCache = flowbotCSRFToken();
  }
});

document.addEventListener('htmx:configRequest', function (evt) {
  var tok = flowbotCSRFToken();
  if (tok) {
    evt.detail.headers['X-CSRF-Token'] = tok;
  }
});

document.addEventListener('submit', function (evt) {
  var form = evt.target;
  if (!form || form.tagName !== 'FORM') return;
  var tok = flowbotCSRFToken();
  if (!tok) return;
  var existing = form.querySelector('input[name="csrf_token"]');
  if (existing) {
    existing.value = tok;
    return;
  }
  var input = document.createElement('input');
  input.type = 'hidden';
  input.name = 'csrf_token';
  input.value = tok;
  form.appendChild(input);
});
