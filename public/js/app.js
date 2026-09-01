// Client i18n strings injected by layout via #flowbot-i18n.
function flowbotI18n(key, fallback) {
  var el = document.getElementById('flowbot-i18n');
  if (!el || !el.textContent) {
    return fallback || key;
  }
  try {
    var dict = JSON.parse(el.textContent);
    if (dict && dict[key]) {
      return dict[key];
    }
  } catch {
    /* ignore parse failures */
  }
  return fallback || key;
}

// Alpine.js shared data stores and utilities
(function () {
  function registerAppAlpineData() {
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

    Alpine.data('lifeGoalCompose', () => ({
      cat: 'Project',
      get showArea() {
        return this.cat === 'Project' || this.cat === 'Resource';
      },
    }));

    Alpine.data('agentKnowledgeForm', () => ({
      content: '',
      init() {
        var ta = this.$el.querySelector(
          '[data-testid="agent-knowledge-content"]',
        );
        if (ta) {
          this.content = ta.value || '';
        }
      },
      get canGenerate() {
        return String(this.content || '').trim().length > 0;
      },
    }));
  }

  document.addEventListener('alpine:init', registerAppAlpineData);
})();

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

// Exponential backoff for SSE reconnects: 1s → 2s → 4s → 8s (cap).
function flowbotNextReconnectDelay(attempt) {
  var n = Math.max(0, attempt | 0);
  var ms = 1000 * Math.pow(2, n);
  return Math.min(ms, 8000);
}
window.flowbotNextReconnectDelay = flowbotNextReconnectDelay;

// Bridge HTMX HX-Trigger {"showToast": {...}} events to the toast UI.
// Listen on document (not body): app.js loads in <head> before body exists.
document.addEventListener('showToast', function (evt) {
  var d = evt.detail || {};
  showToast(d.message || '', d.type || 'info');
});

// CSRF double-submit: cookie csrf_ or __Host-csrf_ + X-Csrf-Token header / form field.
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
  return (
    flowbotGetCookie('__Host-csrf_') ||
    flowbotGetCookie('csrf_') ||
    window.flowbotCSRFCache ||
    ''
  );
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
    headers['X-Csrf-Token'] = tok;
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
    var field = null;
    var form =
      el.tagName === 'FORM' ? el : el.closest ? el.closest('form') : null;
    if (form) {
      field = form.querySelector('input[name="csrf_token"]');
    }
    if ((!field || !field.value) && document.querySelector) {
      field = document.querySelector('input[name="csrf_token"]');
    }
    if (field && field.value) {
      tok = field.value;
      window.flowbotCSRFCache = tok;
    }
  }
  if (tok) {
    evt.detail.headers['X-Csrf-Token'] = tok;
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
  showToast(
    flowbotI18n(
      'error.network',
      'Network error. Check your connection and try again.',
    ),
    'error',
  );
});

document.addEventListener('htmx:timeout', function () {
  showToast(
    flowbotI18n('error.timeout', 'Request timed out. Please try again.'),
    'error',
  );
});

// Preserve scroll position for containers marked data-preserve-scroll across HTMX swaps.
(function () {
  var saved = null;

  function scrollRoot(elt) {
    if (!elt) {
      return null;
    }
    if (elt.hasAttribute && elt.hasAttribute('data-preserve-scroll')) {
      return elt;
    }
    if (elt.closest) {
      return elt.closest('[data-preserve-scroll]');
    }
    return null;
  }

  document.addEventListener('htmx:beforeSwap', function (evt) {
    var target = evt.detail && evt.detail.target;
    var root = scrollRoot(target);
    if (!root) {
      saved = null;
      return;
    }
    saved = {
      id: root.id || '',
      top: root.scrollTop,
      left: root.scrollLeft,
      winX: window.scrollX,
      winY: window.scrollY,
    };
  });

  document.addEventListener('htmx:afterSwap', function (evt) {
    if (!saved) {
      return;
    }
    var target = evt.detail && evt.detail.target;
    var root = scrollRoot(target);
    if (root && (!saved.id || root.id === saved.id)) {
      root.scrollTop = saved.top;
      root.scrollLeft = saved.left;
    }
    window.scrollTo(saved.winX, saved.winY);
    saved = null;
  });
})();

// Keep in sync with htmxResponseErrorMessage in internal/modules/web/utils.go.
function flowbotHTMXErrorMessage(status, body) {
  body = (body || '').trim();
  if (body && body.length < 240 && body.indexOf('<') === -1) {
    return body;
  }
  if (status === 403) {
    return flowbotI18n(
      'error.permission_denied',
      'Permission denied. You do not have access to perform this action.',
    );
  }
  if (status === 400 || status === 422) {
    return flowbotI18n(
      'error.validation',
      'Validation error. Check your input and try again.',
    );
  }
  if (status === 404) {
    return flowbotI18n(
      'error.not_found',
      'Not found. The requested resource no longer exists.',
    );
  }
  if (status === 408 || status === 504) {
    return flowbotI18n('error.timeout', 'Request timed out. Please try again.');
  }
  if (status >= 500) {
    return (
      flowbotI18n('error.server', 'Server error') +
      ' (' +
      status +
      '). ' +
      flowbotI18n('error.try_again', 'Please try again.')
    );
  }
  if (status) {
    return (
      flowbotI18n('error.request_failed', 'Request failed') +
      ' (' +
      status +
      '). ' +
      flowbotI18n('error.try_again', 'Please try again.')
    );
  }
  return (
    flowbotI18n('error.request_failed', 'Request failed') +
    '. ' +
    flowbotI18n('error.try_again', 'Please try again.')
  );
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

window.flowbotAbortInFlightHTMX = function () {
  if (typeof htmx === 'undefined' || typeof htmx.trigger !== 'function') {
    return;
  }
  document.querySelectorAll('.htmx-request').forEach(function (el) {
    htmx.trigger(el, 'htmx:abort');
  });
};

window.flowbotIsPrimarySameTabNav = function (event) {
  if (!event || event.defaultPrevented || event.button) {
    return false;
  }
  if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
    return false;
  }
  var a =
    event.target && event.target.closest
      ? event.target.closest('a[href]')
      : null;
  if (!a) {
    return false;
  }
  if (a.closest && a.closest('[data-confirm]')) {
    return false;
  }
  if (
    a.getAttribute('hx-get') ||
    a.getAttribute('hx-post') ||
    a.getAttribute('hx-put') ||
    a.getAttribute('hx-patch') ||
    a.getAttribute('hx-delete')
  ) {
    return false;
  }
  var href = a.getAttribute('href');
  if (!href) {
    return false;
  }
  var scheme;
  try {
    scheme = decodeURI(href).trim().toLowerCase();
  } catch {
    scheme = href.trim().toLowerCase();
  }
  if (
    scheme.charAt(0) === '#' ||
    scheme.startsWith('javascript:') ||
    scheme.startsWith('data:') ||
    scheme.startsWith('vbscript:')
  ) {
    return false;
  }
  var target = a.getAttribute('target');
  if (target && target !== '_self') {
    return false;
  }
  return !a.hasAttribute('download');
};

// Bubble phase so data-confirm capture can preventDefault first.
// Intentionally silent: free HTTP/1.1 sockets before the next document GET.
document.addEventListener('click', function (event) {
  if (!window.flowbotIsPrimarySameTabNav(event)) {
    return;
  }
  window.flowbotAbortInFlightHTMX();
});

window.addEventListener('pagehide', function () {
  window.flowbotAbortInFlightHTMX();
});

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

// Page tab title + desktop Notification helpers (approval / live run status).
(function () {
  var baseTitle = '';
  var lastNotifyKey = '';

  function flowbotCaptureBaseTitle() {
    if (!baseTitle) {
      baseTitle = document.title || 'Flowbot';
    }
    return baseTitle;
  }

  // Status strings replace the tab title entirely so background tabs stay readable.
  function flowbotSetPageStatus(status) {
    var base = flowbotCaptureBaseTitle();
    if (!status) {
      document.title = base;
      return base;
    }
    document.title = String(status);
    return document.title;
  }

  function flowbotClearPageStatus() {
    return flowbotSetPageStatus('');
  }

  function flowbotFormatNeedsApprovalTitle() {
    return (
      '\u25CF ' + flowbotI18n('client.app.needs_approval', 'Needs approval')
    );
  }

  function flowbotFormatLiveFinishedTitle(pipelineName, failed) {
    var name = String(pipelineName || '').trim();
    var prefix = failed
      ? flowbotI18n('client.pipeline_run.failed', 'Live failed')
      : flowbotI18n('client.pipeline_run.finished', 'Live finished');
    return name ? prefix + ': ' + name : prefix;
  }

  function flowbotRequestNotifyPermission() {
    if (typeof Notification === 'undefined') {
      return Promise.resolve('denied');
    }
    if (Notification.permission !== 'default') {
      return Promise.resolve(Notification.permission);
    }
    try {
      return Notification.requestPermission();
    } catch {
      return Promise.resolve('denied');
    }
  }

  // Desktop notify only when the tab is hidden (user already sees the in-page panel).
  function flowbotNotifyIfHidden(opts) {
    opts = opts || {};
    if (typeof Notification === 'undefined') {
      return null;
    }
    if (typeof document.hidden === 'boolean' && !document.hidden) {
      return null;
    }
    if (Notification.permission !== 'granted') {
      return null;
    }
    var key = opts.tag || opts.title || '';
    if (key && key === lastNotifyKey) {
      return null;
    }
    lastNotifyKey = key;
    try {
      var n = new Notification(opts.title || 'Flowbot', {
        body: opts.body || '',
        tag: opts.tag || 'flowbot-status',
      });
      n.addEventListener('click', function () {
        try {
          window.focus();
        } catch {
          /* ignore */
        }
        n.close();
      });
      return n;
    } catch {
      return null;
    }
  }

  function flowbotResetNotifyDedupe() {
    lastNotifyKey = '';
  }

  window.flowbotCaptureBaseTitle = flowbotCaptureBaseTitle;
  window.flowbotSetPageStatus = flowbotSetPageStatus;
  window.flowbotClearPageStatus = flowbotClearPageStatus;
  window.flowbotFormatNeedsApprovalTitle = flowbotFormatNeedsApprovalTitle;
  window.flowbotFormatLiveFinishedTitle = flowbotFormatLiveFinishedTitle;
  window.flowbotRequestNotifyPermission = flowbotRequestNotifyPermission;
  window.flowbotNotifyIfHidden = flowbotNotifyIfHidden;
  window.flowbotResetNotifyDedupe = flowbotResetNotifyDedupe;
})();
