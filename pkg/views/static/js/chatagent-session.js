(function () {
  'use strict';

  var panel = document.getElementById('chatagent-approval-panel');
  if (!panel) {
    return;
  }

  var sessionID = panel.getAttribute('data-session-id');
  if (!sessionID) {
    return;
  }

  var summaryEl = document.getElementById('chatagent-approval-summary');
  var metaEl = document.getElementById('chatagent-approval-meta');
  var resolvedEl = document.getElementById('chatagent-approval-resolved');
  var actionsEl = document.getElementById('chatagent-approval-actions');
  var alwaysBtn = panel.querySelector('[data-mode="always"]');
  var pending = null;
  var source = null;

  function hidePanel() {
    panel.classList.add('hidden');
    pending = null;
    if (actionsEl) {
      actionsEl.classList.remove('hidden');
    }
    if (resolvedEl) {
      resolvedEl.classList.add('hidden');
    }
  }

  function showResolved(text) {
    if (!resolvedEl) {
      return;
    }
    resolvedEl.textContent = text;
    resolvedEl.classList.remove('hidden');
    if (actionsEl) {
      actionsEl.classList.add('hidden');
    }
  }

  function showConfirm(ev) {
    pending = ev;
    panel.classList.remove('hidden');
    if (summaryEl) {
      summaryEl.textContent = (ev.tool || 'tool') + ': ' + (ev.summary || '');
    }
    if (metaEl) {
      var parts = [];
      if (ev.permission) {
        parts.push('permission: ' + ev.permission);
      }
      if (ev.pattern) {
        parts.push('pattern: ' + ev.pattern);
      }
      metaEl.textContent = parts.join(' · ');
    }
    if (alwaysBtn) {
      if (ev.suggest_always && ev.suggested_pattern) {
        alwaysBtn.classList.remove('hidden');
      } else {
        alwaysBtn.classList.add('hidden');
      }
    }
    if (resolvedEl) {
      resolvedEl.classList.add('hidden');
    }
    if (actionsEl) {
      actionsEl.classList.remove('hidden');
    }
  }

  function postConfirm(approved, mode) {
    if (!pending || !pending.id) {
      return;
    }
    var body = {
      id: pending.id,
      approved: approved,
      mode: mode,
      pattern: approved && mode === 'always' ? (pending.suggested_pattern || '') : ''
    };
    fetch('/service/web/agent-sessions/' + encodeURIComponent(sessionID) + '/confirm', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    }).catch(function () {
      showResolved('Confirm request failed.');
    });
  }

  panel.addEventListener('click', function (event) {
    var btn = event.target.closest('[data-mode]');
    if (!btn || !pending) {
      return;
    }
    var mode = btn.getAttribute('data-mode');
    if (mode === 'once') {
      postConfirm(true, 'once');
    } else if (mode === 'always') {
      postConfirm(true, 'always');
    } else if (mode === 'reject') {
      postConfirm(false, 'reject');
    }
  });

  function connect() {
    if (source) {
      source.close();
    }
    source = new EventSource('/service/web/agent-sessions/' + encodeURIComponent(sessionID) + '/events');
    source.onmessage = function (msg) {
      if (!msg.data) {
        return;
      }
      var ev;
      try {
        ev = JSON.parse(msg.data);
      } catch (_err) {
        return;
      }
      if (ev.type === 'confirm') {
        showConfirm(ev);
      } else if (ev.type === 'confirm_resolved') {
        var label = ev.approved ? 'Approved' : 'Denied';
        if (ev.reason) {
          label += ' (' + ev.reason + ')';
        }
        showResolved(label);
        window.setTimeout(hidePanel, 2500);
      } else if (ev.type === 'canceled') {
        showResolved('Run canceled.');
        window.setTimeout(hidePanel, 2500);
      }
    };
    source.onerror = function () {
      if (source) {
        source.close();
        source = null;
      }
      window.setTimeout(connect, 3000);
    };
  }

  connect();
})();
