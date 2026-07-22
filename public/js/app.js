// Alpine.js shared data stores and utilities
document.addEventListener('alpine:init', () => {
  Alpine.store('toasts', []);

  Alpine.data('themePicker', () => ({
    theme: 'light',
    open: false,
    setTheme(name) {
      document.documentElement.setAttribute('data-theme', name);
      localStorage.setItem('flowbot-theme', name);
      this.theme = name;
      this.open = false;
    },
    toggleLightDark() {
      this.setTheme(this.theme === 'light' ? 'dark' : 'light');
    },
    init() {
      this.theme =
        document.documentElement.getAttribute('data-theme') || 'light';
    },
  }));
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
  // Prefer server-rendered form field when document.cookie is unavailable (proxies / Secure mismatch).
  if (!tok && evt.detail && evt.detail.elt) {
    var el = evt.detail.elt;
    var form =
      el.tagName === 'FORM' ? el : el.closest ? el.closest('form') : null;
    if (form) {
      var field = form.querySelector('input[name="csrf_token"]');
      if (field && field.value) {
        tok = field.value;
        window.flowbotCSRFCache = tok;
      }
    }
  }
  if (tok) {
    evt.detail.headers['X-CSRF-Token'] = tok;
  }
});

// Expand/collapse run rows without hx-on / fragile onclick return false.
// When detail content is already loaded, clear it and cancel the HTMX fetch.
document.addEventListener('htmx:beforeRequest', function (evt) {
  var elt = evt.detail && evt.detail.elt;
  if (!elt || !elt.hasAttribute('data-run-expand')) {
    return;
  }
  var rid = elt.getAttribute('data-run-id');
  if (!rid) {
    return;
  }
  var detail =
    document.getElementById('workflow-steps-' + rid) ||
    document.getElementById('steps-' + rid);
  if (!detail) {
    return;
  }
  var td = detail.querySelector('td');
  if (!td || !td.innerHTML.trim()) {
    return;
  }
  td.innerHTML = '';
  var chevron = elt.querySelector('.chevron');
  if (chevron) {
    chevron.classList.remove('rotate-90');
  }
  evt.preventDefault();
});

// Rotate expand chevrons without hx-on (hx-on uses new Function → CSP unsafe-eval).
document.addEventListener('htmx:afterRequest', function (evt) {
  var detail = evt.detail;
  if (!detail || !detail.elt || !detail.elt.hasAttribute('data-run-expand')) {
    return;
  }
  if (detail.successful === false) {
    return;
  }
  var chevron = detail.elt.querySelector('.chevron');
  if (chevron) {
    chevron.classList.add('rotate-90');
  }
});

// Pause Run History polling while a run is expanded so Output/Error details stay open.
document.addEventListener('htmx:beforeRequest', function (evt) {
  var elt = evt.detail && evt.detail.elt;
  if (!elt || elt.id !== 'workflow-runs-panel') {
    return;
  }
  if (elt.querySelector('.chevron.rotate-90')) {
    evt.preventDefault();
    return;
  }
  var detailCells = elt.querySelectorAll('.run-detail-row td');
  for (var i = 0; i < detailCells.length; i++) {
    if (detailCells[i].innerHTML.trim()) {
      evt.preventDefault();
      return;
    }
  }
});

// Toggle step-run detail rows without inline onclick (CSP-friendly).
document.addEventListener('click', function (evt) {
  var row = evt.target && evt.target.closest('[data-step-toggle]');
  if (!row) {
    return;
  }
  evt.stopPropagation();
  var chevron = row.querySelector('.step-chevron');
  if (chevron) {
    chevron.classList.toggle('rotate-90');
  }
  var detail = row.nextElementSibling;
  if (detail && detail.classList.contains('step-detail-row')) {
    detail.classList.toggle('hidden');
  }
});

document.addEventListener('keydown', function (evt) {
  if (evt.key !== 'Enter' && evt.key !== ' ') {
    return;
  }
  var row = evt.target && evt.target.closest('[data-step-toggle]');
  if (!row || evt.target !== row) {
    return;
  }
  evt.preventDefault();
  row.click();
});

// Capture phase so the hidden field exists before HTMX serializes the form.
document.addEventListener(
  'submit',
  function (evt) {
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
  },
  true,
);

// Dual-channel HTMX errors: swap HTML / HX-Retarget fragments inline (no toast);
// toast only for network failures and non-HTML error bodies.
function flowbotLoginURL() {
  var next = window.location.pathname + window.location.search;
  return '/service/web/login?next=' + encodeURIComponent(next);
}

function flowbotRedirectToLogin() {
  window.location.href = flowbotLoginURL();
}

function flowbotXHRHasHTMLBody(xhr) {
  if (!xhr) return false;
  var ct = (xhr.getResponseHeader('Content-Type') || '').toLowerCase();
  return ct.indexOf('text/html') !== -1;
}

function flowbotXHRHasRetarget(xhr) {
  if (!xhr) return false;
  return !!(xhr.getResponseHeader('HX-Retarget') || '');
}

document.addEventListener('htmx:beforeSwap', function (evt) {
  var xhr = evt.detail.xhr;
  if (!xhr) return;
  var status = xhr.status;
  if (status >= 200 && status < 400) return;
  if (status === 401) return;
  if (flowbotXHRHasRetarget(xhr) || flowbotXHRHasHTMLBody(xhr)) {
    evt.detail.shouldSwap = true;
    evt.detail.isError = false;
  }
});

document.addEventListener('htmx:responseError', function (evt) {
  var xhr = evt.detail.xhr;
  var status = xhr ? xhr.status : 0;
  if (status === 401) {
    flowbotRedirectToLogin();
    return;
  }
  // Inline FormError / retargeted fragments are handled via beforeSwap (isError=false).
  if (flowbotXHRHasRetarget(xhr) || flowbotXHRHasHTMLBody(xhr)) {
    return;
  }
  var body =
    xhr && typeof xhr.responseText === 'string' ? xhr.responseText : '';
  showToast(flowbotHTMXErrorMessage(status, body), 'error');
});

document.addEventListener('htmx:sendError', function () {
  showToast('Network error. Check your connection and try again.', 'error');
});

document.addEventListener('htmx:timeout', function () {
  showToast('Request timed out. Please try again.', 'error');
});

// Keep in sync with htmxResponseErrorMessage in internal/modules/web/utils.go.
function flowbotHTMXErrorMessage(status, body) {
  body = (body || '').trim();
  if (body && body.length < 240 && body.indexOf('<') === -1) {
    return body;
  }
  if (status === 403) {
    return 'Permission denied. You do not have access to perform this action.';
  }
  if (status === 400 || status === 422) {
    return 'Validation error. Check your input and try again.';
  }
  if (status === 404) {
    return 'Not found. The requested resource no longer exists.';
  }
  if (status === 408 || status === 504) {
    return 'Request timed out. Please try again.';
  }
  if (status >= 500) {
    return 'Server error (' + status + '). Please try again.';
  }
  if (status) {
    return 'Request failed (' + status + '). Please try again.';
  }
  return 'Request failed. Please try again.';
}

// Global top progress: do NOT put hx-indicator on <body> — that replaces the
// requesting element's htmx-request class and hides button HtmxIndicator spinners.
(function () {
  var active = 0;
  function progressEl() {
    return document.getElementById('flowbot-htmx-progress');
  }
  function bump(delta) {
    active = Math.max(0, active + delta);
    var el = progressEl();
    if (!el) {
      return;
    }
    if (active > 0) {
      el.classList.add('htmx-request');
    } else {
      el.classList.remove('htmx-request');
    }
  }
  document.addEventListener('htmx:beforeRequest', function () {
    bump(1);
  });
  document.addEventListener('htmx:afterRequest', function () {
    bump(-1);
  });
})();

// Scroll History deep-links into view after Channels/Rules table settle.
document.addEventListener('htmx:afterSettle', function (evt) {
  var root = evt.target;
  if (!root || !root.querySelector) {
    return;
  }
  var el = root.querySelector('[data-notify-highlight]');
  if (!el) {
    return;
  }
  el.scrollIntoView({ block: 'center', behavior: 'smooth' });
});
