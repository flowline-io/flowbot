(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var PENDING_PREFIX = 'flowbot-chatagent-pending:';

  function pendingKey(sessionID) {
    return PENDING_PREFIX + sessionID;
  }

  function consumePendingPrompt(sessionID) {
    var text = '';
    var params = new URLSearchParams(window.location.search);
    if (params.has('prompt')) {
      text = params.get('prompt') || '';
      params.delete('prompt');
      var suffix = params.toString();
      var cleanURL = window.location.pathname + (suffix ? '?' + suffix : '');
      window.history.replaceState({}, '', cleanURL);
    }
    try {
      var key = pendingKey(sessionID);
      if (!text.trim()) {
        text = sessionStorage.getItem(key) || '';
      }
      // Always clear: create flow writes both ?prompt= and sessionStorage. If we
      // only clear storage when the URL is empty, revisiting the session from the
      // list re-sends the first prompt.
      sessionStorage.removeItem(key);
    } catch {
      /* storage may be unavailable */
    }
    return text.trim();
  }

  function threadHasHistory(threadRoot) {
    var messagesEl = threadRoot.querySelector('#chatagent-messages');
    if (!messagesEl) {
      return false;
    }
    return !!messagesEl.querySelector(
      '[data-role="user"], [data-role="assistant"], [data-role="tool"], [data-role="thinking"]',
    );
  }
  function initComposer(root) {
    var createURL = root.getAttribute('data-create-url');
    var detailTemplate = root.getAttribute('data-detail-url-template');
    var input = root.querySelector('#chatagent-composer-input');
    var startBtn = root.querySelector('#chatagent-composer-start');
    var errorEl = root.querySelector('#chatagent-composer-error');
    if (!createURL || !detailTemplate || !input || !startBtn) {
      return;
    }

    function start() {
      var text = (input.value || '').trim();
      if (!text) {
        ns.showError(errorEl, 'Enter a prompt to start.');
        return;
      }
      ns.showError(errorEl, '');
      startBtn.disabled = true;
      flowbotCSRFHeadersAsync()
        .then(function (headers) {
          return fetch(createURL, { method: 'POST', headers: headers });
        })
        .then(function (res) {
          if (!res.ok) {
            return res
              .json()
              .catch(function () {
                return {};
              })
              .then(function (data) {
                throw new Error(
                  (data && data.error) || 'Failed to create session',
                );
              });
          }
          return res.json();
        })
        .then(function (data) {
          var sessionID = data.session_id;
          if (!sessionID) {
            throw new Error('Missing session id');
          }
          try {
            sessionStorage.setItem(pendingKey(sessionID), text);
          } catch {
            /* storage may be unavailable */
          }
          var detailURL =
            detailTemplate.replace('{id}', sessionID) +
            '?prompt=' +
            encodeURIComponent(text);
          window.location.href = detailURL;
        })
        .catch(function (err) {
          ns.showError(errorEl, err.message || 'Failed to start');
          startBtn.disabled = false;
        });
    }

    startBtn.addEventListener('click', start);
    input.addEventListener('keydown', function (ev) {
      if (ev.key === 'Enter' && !ev.shiftKey) {
        ev.preventDefault();
        start();
      }
    });
  }

  function initThread(root) {
    var sessionID = root.getAttribute('data-session-id');
    var messagesURL = root.getAttribute('data-messages-url');
    var closeURL = root.getAttribute('data-close-url');
    var input = root.querySelector('#chatagent-followup-input');
    var errorEl = root.querySelector('#chatagent-thread-error');
    if (!sessionID || !messagesURL || !input) {
      return;
    }
    var approval = ns.initApproval(
      root.querySelector('#chatagent-approval-panel'),
    );

    function closeSession() {
      if (!closeURL) {
        return;
      }
      var closeBtn = root.querySelector('#chatagent-close-session');
      function doClose() {
        if (closeBtn) {
          closeBtn.disabled = true;
        }
        flowbotCSRFHeadersAsync()
          .then(function (headers) {
            return fetch(closeURL, { method: 'DELETE', headers: headers });
          })
          .then(function (res) {
            if (!res.ok) {
              return res
                .json()
                .catch(function () {
                  return {};
                })
                .then(function (data) {
                  throw new Error(
                    (data && data.error) || 'Failed to close session',
                  );
                });
            }
            window.location.href = '/service/web/agents';
          })
          .catch(function (err) {
            ns.showError(errorEl, err.message || 'Failed to close session');
            if (closeBtn) {
              closeBtn.disabled = false;
            }
          });
      }
      if (window.showConfirmModal) {
        window.showConfirmModal({
          title: 'Close session',
          message:
            'Close this session? You will not be able to send more messages.',
          confirmText: 'Close',
          confirmClass: 'btn-error',
          onConfirm: doClose,
        });
        return;
      }
      if (window.confirm('Close this session?')) {
        doClose();
      }
    }

    var closeBtn = root.querySelector('#chatagent-close-session');
    if (closeBtn) {
      closeBtn.addEventListener('click', closeSession);
    }

    function sendFollowUp() {
      var text = (input.value || '').trim();
      if (!text) {
        return;
      }
      input.value = '';
      ns.streamMessage(messagesURL, text, root, null, approval);
    }

    input.addEventListener('keydown', function (ev) {
      if (ev.key === 'Enter' && !ev.shiftKey) {
        ev.preventDefault();
        sendFollowUp();
      }
    });

    var pending = consumePendingPrompt(sessionID);
    ns.initContextControl(root);
    if (ns.hydrateTodosFromToolCards) {
      ns.hydrateTodosFromToolCards(root);
    }
    if (ns.refreshTodosFromServer) {
      ns.refreshTodosFromServer(root);
    }
    if (pending && !threadHasHistory(root)) {
      ns.streamMessage(messagesURL, pending, root, null, approval);
    }
  }

  document
    .querySelectorAll('[data-chatagent-root="composer"]')
    .forEach(initComposer);
  document
    .querySelectorAll('[data-chatagent-root="thread"]')
    .forEach(initThread);
  document
    .querySelectorAll('#chatagent-approval-panel')
    .forEach(function (panel) {
      if (panel.closest('[data-chatagent-root="thread"]')) {
        return;
      }
      ns.initApproval(panel);
    });
})();
