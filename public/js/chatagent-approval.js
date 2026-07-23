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

  function reconnectDelay(attempt) {
    if (typeof window.flowbotNextReconnectDelay === 'function') {
      return window.flowbotNextReconnectDelay(attempt);
    }
    return Math.min(1000 * Math.pow(2, Math.max(0, attempt)), 8000);
  }

  function setApprovalTitle(active) {
    if (typeof window.flowbotSetPageStatus !== 'function') {
      return;
    }
    if (!active) {
      if (typeof window.flowbotClearPageStatus === 'function') {
        window.flowbotClearPageStatus();
      }
      return;
    }
    var title =
      typeof window.flowbotFormatNeedsApprovalTitle === 'function'
        ? window.flowbotFormatNeedsApprovalTitle()
        : '\u25CF Needs approval';
    window.flowbotSetPageStatus(title);
  }

  function notifyApproval(ev, sessionID) {
    if (typeof window.flowbotNotifyIfHidden !== 'function') {
      return;
    }
    var tool = (ev && ev.tool) || 'tool';
    var summary = (ev && ev.summary) || '';
    window.flowbotNotifyIfHidden({
      title: 'Needs approval',
      body: summary ? tool + ': ' + summary : tool,
      tag: 'flowbot-approval-' + ((ev && ev.id) || sessionID),
    });
  }

  ns.initApproval = function (panel) {
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
    var alwaysHint = panel.querySelector('#chatagent-approve-always-hint');
    var pending = null;
    var submitting = false;
    var toastTimer = null;
    var source = null;
    var reconnectAttempt = 0;
    var reconnectTimer = null;
    var permissionArmed = false;
    // Reload only when the turn fully ends. Timed reloads interrupt multi-tool
    // approval chains and wipe the page before finishStream persists history.
    var reloadOnComplete = false;

    function clearReconnectTimer() {
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
    }

    function armNotifyPermission() {
      if (permissionArmed) {
        return;
      }
      permissionArmed = true;
      if (typeof window.flowbotRequestNotifyPermission === 'function') {
        window.flowbotRequestNotifyPermission();
      }
    }

    function onVisibilityChange() {
      if (!pending || !document.hidden) {
        return;
      }
      notifyApproval(pending, sessionID);
    }

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

    function waitingCopyEl() {
      return threadRoot
        ? threadRoot.querySelector('[data-testid="chatagent-run-waiting"]')
        : null;
    }

    function setWaitingCopy(text) {
      var el = waitingCopyEl();
      if (el && text) {
        el.textContent = text;
      }
    }

    function hidePanel() {
      panel.classList.add('hidden');
      pending = null;
      submitting = false;
      setApprovalTitle(false);
      if (typeof window.flowbotResetNotifyDedupe === 'function') {
        window.flowbotResetNotifyDedupe();
      }
      if (actionsEl) {
        actionsEl.classList.remove('hidden');
      }
    }

    function showConfirm(ev) {
      hideToast();
      pending = ev;
      submitting = false;
      panel.classList.remove('hidden');
      setApprovalTitle(true);
      notifyApproval(ev, sessionID);
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
          alwaysBtn.setAttribute(
            'title',
            'Remember for future matching calls: ' + ev.suggested_pattern,
          );
          if (alwaysHint) {
            alwaysHint.textContent = ev.suggested_pattern;
          }
        } else {
          alwaysBtn.classList.add('hidden');
          alwaysBtn.removeAttribute('title');
          if (alwaysHint) {
            alwaysHint.textContent = '';
          }
        }
      }
      if (actionsEl) {
        actionsEl.classList.remove('hidden');
      }
      setWaitingCopy(
        'Waiting for tool approval. The rest of this turn appears after it finishes.',
      );
    }

    function hydratePendingFromPanel() {
      var id = panel.getAttribute('data-pending-confirm-id') || '';
      if (!id) {
        return;
      }
      reloadOnComplete = true;
      showConfirm({
        type: 'confirm',
        id: id,
        tool: panel.getAttribute('data-pending-tool') || '',
        summary: panel.getAttribute('data-pending-summary') || '',
        permission: panel.getAttribute('data-pending-permission') || '',
        pattern: panel.getAttribute('data-pending-pattern') || '',
        suggested_pattern:
          panel.getAttribute('data-pending-suggested-pattern') || '',
        suggest_always:
          panel.getAttribute('data-pending-suggest-always') === '1',
      });
    }

    function isDetachedObserver() {
      return (
        !!waitingCopyEl() || (threadRoot && !ns.isThreadRunning(threadRoot))
      );
    }

    function markApprovedWaiting() {
      reloadOnComplete = true;
      showStatusToast('Approved — continuing the turn…', 'info');
      setWaitingCopy(
        'Approved. Waiting for the next step (another approval or the final reply)…',
      );
    }

    function resolveConfirmEvent(ev) {
      if (ev.type === 'confirm') {
        // Keep the observer on-page across multi-step approvals; history is
        // refreshed on run_complete once mid-turn tool results are durable.
        reloadOnComplete = true;
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
        if (ev.approved && isDetachedObserver()) {
          markApprovedWaiting();
        } else {
          showStatusToast(
            formatConfirmResolvedLabel(ev),
            ev.approved ? 'success' : 'warning',
          );
          if (!ev.approved && isDetachedObserver()) {
            reloadOnComplete = true;
            setWaitingCopy('Denied. Waiting for the turn to finish…');
          }
        }
        return;
      }
      if (ev.type === 'run_complete') {
        // Only reload when this page was observing an in-flight approval turn.
        // Idle /events subscribers must not full-reload every unrelated finish.
        if (reloadOnComplete) {
          window.location.reload();
        }
        return;
      }
      if (ev.type === 'canceled') {
        pending = null;
        submitting = false;
        hidePanel();
        showStatusToast('Run canceled.', 'warning');
        reloadOnComplete = true;
        window.setTimeout(function () {
          window.location.reload();
        }, 600);
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
      var wasApproved = !!approved;
      flowbotCSRFHeadersAsync({ 'Content-Type': 'application/json' })
        .then(function (headers) {
          return fetch(confirmURL, {
            method: 'POST',
            headers: headers,
            body: JSON.stringify({
              id: pending.id,
              approved: approved,
              mode: mode,
              pattern:
                approved && mode === 'always'
                  ? pending.suggested_pattern || ''
                  : '',
            }),
          });
        })
        .then(function (res) {
          if (res.status === 204) {
            pending = null;
            submitting = false;
            hidePanel();
            if (wasApproved) {
              if (isDetachedObserver()) {
                markApprovedWaiting();
              } else {
                showStatusToast('Approved', 'success');
              }
            } else {
              showStatusToast('Denied', 'warning');
              if (isDetachedObserver()) {
                reloadOnComplete = true;
                setWaitingCopy('Denied. Waiting for the turn to finish…');
              }
            }
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
      armNotifyPermission();
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

    if (threadRoot) {
      threadRoot.addEventListener(
        'click',
        function () {
          armNotifyPermission();
        },
        { once: true },
      );
      threadRoot.addEventListener(
        'keydown',
        function () {
          armNotifyPermission();
        },
        { once: true },
      );
    }
    document.addEventListener('visibilitychange', onVisibilityChange);

    function connect() {
      clearReconnectTimer();
      if (source) {
        source.close();
      }
      source = new EventSource(eventsURL);
      source.addEventListener('open', function () {
        if (reconnectAttempt > 0) {
          showStatusToast('Connected', 'info');
        }
        reconnectAttempt = 0;
      });
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
        showStatusToast('Reconnecting…', 'warning');
        var delay = reconnectDelay(reconnectAttempt);
        reconnectAttempt += 1;
        reconnectTimer = window.setTimeout(connect, delay);
      });
    }

    hydratePendingFromPanel();
    connect();
    return { handleStreamEvent: resolveConfirmEvent };
  };
})();
