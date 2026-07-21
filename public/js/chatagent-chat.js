(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var PENDING_PREFIX = 'flowbot-chatagent-pending:';
  var COMPOSER_MODEL_KEY = 'flowbot-chatagent-composer:model';
  var COMPOSER_THINKING_KEY = 'flowbot-chatagent-composer:thinking';

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

  function lsGet(key) {
    try {
      return localStorage.getItem(key) || '';
    } catch {
      return '';
    }
  }

  function lsSet(key, value) {
    try {
      if (value) {
        localStorage.setItem(key, value);
      } else {
        localStorage.removeItem(key);
      }
    } catch {
      /* storage may be unavailable */
    }
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
    var defaultModel = root.getAttribute('data-default-model') || '';
    var input = root.querySelector('#chatagent-composer-input');
    var startBtn = root.querySelector('#chatagent-composer-start');
    var modelSel = root.querySelector('#chatagent-composer-model');
    var thinkingSel = root.querySelector('#chatagent-composer-thinking');
    var errorEl = root.querySelector('#chatagent-composer-error');
    if (!createURL || !detailTemplate || !input || !startBtn) {
      return;
    }

    // Restore last-used values from localStorage.
    if (modelSel) {
      var savedModel = lsGet(COMPOSER_MODEL_KEY) || defaultModel;
      if (savedModel) {
        modelSel.value = savedModel;
        // Fall back to first option if saved value is no longer available.
        if (modelSel.value !== savedModel) {
          modelSel.selectedIndex = 0;
        }
      }
      modelSel.addEventListener('change', function () {
        lsSet(COMPOSER_MODEL_KEY, modelSel.value);
      });
    }
    if (thinkingSel) {
      var savedThinking = lsGet(COMPOSER_THINKING_KEY) || 'default';
      thinkingSel.value = savedThinking;
      if (thinkingSel.value !== savedThinking) {
        thinkingSel.value = 'default';
      }
      thinkingSel.addEventListener('change', function () {
        lsSet(COMPOSER_THINKING_KEY, thinkingSel.value);
      });
    }

    function start() {
      var text = (input.value || '').trim();
      if (!text) {
        ns.showError(errorEl, 'Enter a prompt to start.');
        return;
      }
      ns.showError(errorEl, '');
      startBtn.disabled = true;

      var body = {
        model: modelSel && modelSel.value ? modelSel.value : defaultModel || '',
        thinking_level:
          thinkingSel && thinkingSel.value ? thinkingSel.value : 'default',
      };

      flowbotCSRFHeadersAsync()
        .then(function (headers) {
          headers['Content-Type'] = 'application/json';
          return fetch(createURL, {
            method: 'POST',
            headers: headers,
            body: JSON.stringify(body),
          });
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

  function thinkingLabel(level) {
    var normalized = (level || 'default').toLowerCase();
    if (normalized === 'off') {
      return 'Off';
    }
    if (normalized === 'low') {
      return 'Low';
    }
    if (normalized === 'medium') {
      return 'Medium';
    }
    if (normalized === 'high') {
      return 'High';
    }
    return 'Default';
  }

  function settingsHeaderLabel(model, thinking, defaultModel) {
    var m = (model || '').trim() || (defaultModel || '').trim();
    var tl = thinkingLabel(thinking);
    if (m && tl) {
      return m + ' \u00b7 Thinking: ' + tl;
    }
    if (m) {
      return m;
    }
    if (tl) {
      return 'Thinking: ' + tl;
    }
    return '';
  }

  function initThreadSettings(root) {
    var settingsURL = root.getAttribute('data-settings-url');
    if (!settingsURL) {
      return;
    }
    var defaultModel = root.getAttribute('data-default-model') || '';
    var modelSel = root.querySelector('#chatagent-thread-model');
    var thinkingSel = root.querySelector('#chatagent-thread-thinking');
    var modelLabel = root.querySelector('#chatagent-session-model-label');
    var errorEl = root.querySelector('#chatagent-thread-error');

    var currentModel = '';
    var currentThinking = 'default';
    if (modelSel) {
      currentModel =
        modelSel.getAttribute('data-session-model') ||
        modelSel.value ||
        defaultModel ||
        '';
      if (!modelSel.getAttribute('data-session-model') && defaultModel) {
        modelSel.value = defaultModel;
        if (modelSel.value !== defaultModel && modelSel.options.length) {
          // keep server-rendered selection if default is not in list
        }
      } else if (currentModel) {
        modelSel.value = currentModel;
      }
      currentModel = modelSel.value || currentModel;
    } else {
      // No model picker: preserve the stored override (may be empty = yaml default).
      currentModel = root.getAttribute('data-session-model') || '';
    }
    if (thinkingSel) {
      currentThinking =
        thinkingSel.getAttribute('data-session-thinking') ||
        thinkingSel.value ||
        'default';
      thinkingSel.value = currentThinking || 'default';
      currentThinking = thinkingSel.value || 'default';
    }

    function putSettings(nextModel, nextThinking) {
      var body = {
        model: nextModel || '',
        thinking_level: nextThinking || 'default',
      };
      flowbotCSRFHeadersAsync()
        .then(function (headers) {
          headers['Content-Type'] = 'application/json';
          return fetch(settingsURL, {
            method: 'PUT',
            headers: headers,
            body: JSON.stringify(body),
          });
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
                  (data && data.error) || 'Failed to save settings',
                );
              });
          }
          return res.json();
        })
        .then(function (data) {
          currentModel = (data && data.model) || '';
          currentThinking = (data && data.thinking_level) || 'default';
          if (modelLabel) {
            modelLabel.textContent = settingsHeaderLabel(
              currentModel,
              currentThinking,
              defaultModel,
            );
          }
          ns.showError(errorEl, '');
        })
        .catch(function (err) {
          var msg = (err && err.message) || 'Failed to save settings';
          if (typeof showToast === 'function') {
            showToast(msg, 'error');
          }
          ns.showError(errorEl, msg);
        });
    }

    if (modelSel) {
      modelSel.addEventListener('change', function () {
        var thinking = thinkingSel ? thinkingSel.value : currentThinking;
        putSettings(modelSel.value, thinking);
      });
    }
    if (thinkingSel) {
      thinkingSel.addEventListener('change', function () {
        var model = modelSel ? modelSel.value : currentModel;
        putSettings(model, thinkingSel.value);
      });
    }
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

    initThreadSettings(root);

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
