(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  function formatConfirmResolvedLabel(ev) {
    if (ev.reason === 'timeout') {
      return 'Timed out — tool was denied automatically.';
    }
    var label = ev.approved ? 'Approved' : 'Denied';
    if (ev.reason && ev.reason !== 'approved' && ev.reason !== 'denied') {
      label += ' (' + ev.reason + ')';
    }
    return label;
  }

  ns.initApproval = function(panel) {
    if (!panel) {
      return null;
    }
    var sessionID = panel.getAttribute('data-session-id');
    var confirmURL = panel.getAttribute('data-confirm-url');
    var eventsURL = panel.getAttribute('data-events-url');
    if (!sessionID || !confirmURL || !eventsURL) {
      return null;
    }

    var threadRoot = panel.closest('[data-chatagent-root="thread"]');
    var toastEl = threadRoot
      ? threadRoot.querySelector('#chatagent-status-toast')
      : document.getElementById('chatagent-status-toast');
    var summaryEl = panel.querySelector('#chatagent-approval-summary');
    var metaEl = panel.querySelector('#chatagent-approval-meta');
    var actionsEl = panel.querySelector('#chatagent-approval-actions');
    var alwaysBtn = panel.querySelector('[data-mode="always"]');
    var pending = null;
    var submitting = false;
    var toastTimer = null;
    var source = null;

    function clearToastTimer() {
      if (toastTimer) {
        clearTimeout(toastTimer);
        toastTimer = null;
      }
    }

    function hideToast() {
      clearToastTimer();
      if (!toastEl) {
        return;
      }
      toastEl.classList.add('hidden');
      toastEl.textContent = '';
      toastEl.classList.remove(
        'alert-success',
        'alert-warning',
        'alert-error',
        'alert-info',
      );
    }

    function showStatusToast(text, tone) {
      if (!toastEl) {
        return;
      }
      clearToastTimer();
      toastEl.textContent = text;
      toastEl.classList.remove(
        'hidden',
        'alert-success',
        'alert-warning',
        'alert-error',
        'alert-info',
      );
      if (tone === 'success') {
        toastEl.classList.add('alert-success');
      } else if (tone === 'error') {
        toastEl.classList.add('alert-error');
      } else if (tone === 'warning') {
        toastEl.classList.add('alert-warning');
      } else {
        toastEl.classList.add('alert-info');
      }
      toastTimer = window.setTimeout(hideToast, 2500);
    }

    function hidePanel() {
      panel.classList.add('hidden');
      pending = null;
      submitting = false;
      if (actionsEl) {
        actionsEl.classList.remove('hidden');
      }
    }

    function showConfirm(ev) {
      hideToast();
      pending = ev;
      submitting = false;
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
      if (actionsEl) {
        actionsEl.classList.remove('hidden');
      }
    }

    function resolveConfirmEvent(ev) {
      if (ev.type === 'confirm') {
        showConfirm(ev);
        return;
      }
      if (ev.type === 'confirm_resolved') {
        if (pending && ev.id && pending.id !== ev.id) {
          return;
        }
        pending = null;
        submitting = false;
        hidePanel();
        showStatusToast(
          formatConfirmResolvedLabel(ev),
          ev.approved ? 'success' : 'warning',
        );
        return;
      }
      if (ev.type === 'canceled') {
        pending = null;
        submitting = false;
        hidePanel();
        showStatusToast('Run canceled.', 'warning');
      }
    }

    function showConfirmExpired(message) {
      pending = null;
      submitting = false;
      hidePanel();
      showStatusToast(message || 'Approval request expired.', 'error');
    }

    function postConfirm(approved, mode) {
      if (!pending || !pending.id || submitting) {
        return;
      }
      submitting = true;
      fetch(confirmURL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: pending.id,
          approved: approved,
          mode: mode,
          pattern:
            approved && mode === 'always'
              ? pending.suggested_pattern || ''
              : '',
        }),
      })
        .then(function (res) {
          if (res.status === 204) {
            submitting = false;
            return;
          }
          if (res.status === 404 || res.status === 409) {
            showConfirmExpired('Approval request expired or already resolved.');
            return;
          }
          submitting = false;
          return res
            .json()
            .catch(function () {
              return {};
            })
            .then(function (data) {
              showStatusToast(
                (data && data.error) || 'Confirm request failed.',
                'error',
              );
            });
        })
        .catch(function () {
          submitting = false;
          showStatusToast('Confirm request failed.', 'error');
        });
    }

    panel.addEventListener('click', function (event) {
      var btn = event.target.closest('[data-mode]');
      if (!btn || !pending || submitting) {
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
      source = new EventSource(eventsURL);
      source.addEventListener('message', function (msg) {
        if (!msg.data) {
          return;
        }
        var ev;
        try {
          ev = JSON.parse(msg.data);
        } catch {
          return;
        }
        resolveConfirmEvent(ev);
      });
      source.addEventListener('error', function () {
        if (source) {
          source.close();
          source = null;
        }
        window.setTimeout(connect, 3000);
      });
    }

    connect();
    return { handleStreamEvent: resolveConfirmEvent };
  }
})();
