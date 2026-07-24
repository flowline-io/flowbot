(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var PENDING_PREFIX = 'flowbot-chatagent-pending:';
  var COMPOSER_MODEL_KEY = 'flowbot-chatagent-composer:model';
  var COMPOSER_THINKING_KEY = 'flowbot-chatagent-composer:thinking';
  var MAX_ATTACHMENTS = 8;

  function pendingKey(sessionID) {
    return PENDING_PREFIX + sessionID;
  }

  function parsePendingPayload(raw) {
    if (!raw) {
      return { text: '', attachments: [] };
    }
    try {
      var parsed = JSON.parse(raw);
      if (
        parsed &&
        typeof parsed === 'object' &&
        !Array.isArray(parsed) &&
        ('text' in parsed || 'attachments' in parsed)
      ) {
        return {
          text: typeof parsed.text === 'string' ? parsed.text : '',
          attachments: Array.isArray(parsed.attachments)
            ? parsed.attachments
            : [],
        };
      }
    } catch {
      /* legacy plain-text pending prompt */
    }
    return { text: String(raw), attachments: [] };
  }

  function consumePendingPrompt(sessionID) {
    var text = '';
    var attachments = [];
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
      var stored = parsePendingPayload(sessionStorage.getItem(key) || '');
      // Always clear: create flow writes both ?prompt= and sessionStorage. If we
      // only clear storage when the URL is empty, revisiting the session from the
      // list re-sends the first prompt.
      sessionStorage.removeItem(key);
      if (!text.trim()) {
        text = stored.text || '';
      }
      attachments = stored.attachments || [];
    } catch {
      /* storage may be unavailable */
    }
    return { text: text.trim(), attachments: attachments };
  }

  function storePendingPrompt(sessionID, text, attachments) {
    try {
      sessionStorage.setItem(
        pendingKey(sessionID),
        JSON.stringify({
          text: text || '',
          attachments: attachments || [],
        }),
      );
    } catch {
      /* storage may be unavailable */
    }
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

  function isImageMime(mime) {
    return !!(mime && mime.indexOf('image/') === 0);
  }

  function revokePreviewURL(item) {
    if (item && item.previewURL) {
      URL.revokeObjectURL(item.previewURL);
      item.previewURL = '';
    }
  }

  function clearPendingAttachments(list) {
    (list || []).forEach(revokePreviewURL);
    list.length = 0;
  }

  function optionMultimodal(opt) {
    if (!opt) {
      return false;
    }
    return opt.getAttribute('data-multimodal') === 'true';
  }

  function selectedModelMultimodal(modelSel, settingsEl) {
    if (modelSel && modelSel.options && modelSel.selectedIndex >= 0) {
      return optionMultimodal(modelSel.options[modelSel.selectedIndex]);
    }
    if (settingsEl) {
      return settingsEl.getAttribute('data-selected-multimodal') === 'true';
    }
    return false;
  }

  function syncAttachVisibility(modelSel, attachBtn, settingsEl, onHide) {
    var multimodal = selectedModelMultimodal(modelSel, settingsEl);
    if (settingsEl) {
      settingsEl.setAttribute(
        'data-selected-multimodal',
        multimodal ? 'true' : 'false',
      );
    }
    if (attachBtn) {
      attachBtn.hidden = !multimodal;
    }
    if (!multimodal && typeof onHide === 'function') {
      onHide();
    }
    return multimodal;
  }

  function createAttachmentQueue(opts) {
    var pendingAttachments = [];
    var pendingEl = opts.pendingEl;
    var mediaInput = opts.mediaInput;
    var attachBtn = opts.attachBtn;
    var modelSel = opts.modelSel;
    var settingsEl = opts.settingsEl;
    var inputEl = opts.inputEl;
    var errorEl = opts.errorEl;

    function renderPendingAttachments() {
      if (!pendingEl) {
        return;
      }
      pendingEl.textContent = '';
      pendingAttachments.forEach(function (item) {
        var rm = document.createElement('button');
        rm.type = 'button';
        rm.className = 'chatagent-pending-remove';
        rm.setAttribute('aria-label', 'Remove attachment');
        rm.textContent = '\u00d7';
        rm.addEventListener('click', function () {
          var i = pendingAttachments.indexOf(item);
          if (i < 0) {
            return;
          }
          revokePreviewURL(pendingAttachments[i]);
          pendingAttachments.splice(i, 1);
          renderPendingAttachments();
        });

        if (item.previewURL && isImageMime(item.mime_type)) {
          var thumb = document.createElement('div');
          thumb.className = 'chatagent-pending-thumb';
          var img = document.createElement('img');
          img.src = item.previewURL;
          img.alt = item.name || 'Attached image';
          thumb.appendChild(img);
          thumb.appendChild(rm);
          pendingEl.appendChild(thumb);
          return;
        }

        var chip = document.createElement('span');
        chip.className = 'chatagent-pending-file';
        var name = document.createElement('span');
        name.className = 'chatagent-pending-file-name';
        name.textContent = item.name || item.kind || 'media';
        chip.appendChild(name);
        chip.appendChild(rm);
        pendingEl.appendChild(chip);
      });
    }

    function clearPending() {
      clearPendingAttachments(pendingAttachments);
      renderPendingAttachments();
    }

    function queueFile(file) {
      if (!file) {
        return;
      }
      if (!selectedModelMultimodal(modelSel, settingsEl)) {
        ns.showError(errorEl, 'Selected model does not support media input');
        return;
      }
      if (pendingAttachments.length >= MAX_ATTACHMENTS) {
        ns.showError(errorEl, 'At most 8 attachments per message');
        return;
      }
      var item = {
        file: file,
        name: file.name,
        mime_type: file.type,
        previewURL: '',
      };
      if (isImageMime(file.type)) {
        item.previewURL = URL.createObjectURL(file);
      }
      pendingAttachments.push(item);
      renderPendingAttachments();
      ns.showError(errorEl, '');
    }

    function syncVisibility() {
      syncAttachVisibility(modelSel, attachBtn, settingsEl, function () {
        if (pendingAttachments.length) {
          clearPending();
        }
      });
    }

    syncVisibility();

    if (modelSel) {
      modelSel.addEventListener('change', syncVisibility);
    }

    if (attachBtn && mediaInput) {
      attachBtn.addEventListener('click', function () {
        if (attachBtn.hidden || attachBtn.disabled) {
          return;
        }
        mediaInput.click();
      });
      mediaInput.addEventListener('change', function () {
        var files = mediaInput.files || [];
        for (var i = 0; i < files.length; i++) {
          queueFile(files[i]);
        }
        mediaInput.value = '';
      });
    }

    if (inputEl) {
      inputEl.addEventListener('paste', function (ev) {
        if (!selectedModelMultimodal(modelSel, settingsEl)) {
          return;
        }
        var items = (ev.clipboardData && ev.clipboardData.items) || [];
        for (var i = 0; i < items.length; i++) {
          if (items[i].type && items[i].type.indexOf('image/') === 0) {
            var f = items[i].getAsFile();
            if (f) {
              queueFile(f);
              ev.preventDefault();
            }
          }
        }
      });
    }

    return {
      list: pendingAttachments,
      render: renderPendingAttachments,
      take: function () {
        var atts = pendingAttachments.slice();
        pendingAttachments.length = 0;
        renderPendingAttachments();
        return atts;
      },
      clear: clearPending,
      syncVisibility: syncVisibility,
    };
  }

  function uploadComposerAttachments(mediaURL, files) {
    return Promise.all(
      (files || []).map(function (item) {
        if (!item.file) {
          return Promise.reject(new Error('missing attachment file'));
        }
        var fd = new FormData();
        fd.append(
          'file',
          item.file,
          item.name || item.file.name || 'upload.bin',
        );
        return flowbotCSRFHeadersAsync({}).then(function (upHeaders) {
          return fetch(mediaURL, {
            method: 'POST',
            headers: upHeaders,
            body: fd,
          }).then(function (res) {
            return res.json().then(function (body) {
              if (!res.ok) {
                throw new Error((body && body.error) || 'upload failed');
              }
              return {
                file_id: body.file_id,
                mime_type: body.mime_type,
                kind: body.kind,
              };
            });
          });
        });
      }),
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
    var settingsEl = root.querySelector('[data-testid="chatagent-settings"]');
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

    var queue = createAttachmentQueue({
      pendingEl: root.querySelector('#chatagent-composer-pending'),
      mediaInput: root.querySelector('#chatagent-composer-media-input'),
      attachBtn: root.querySelector('#chatagent-composer-attach'),
      modelSel: modelSel,
      settingsEl: settingsEl,
      inputEl: input,
      errorEl: errorEl,
    });

    function start() {
      var text = (input.value || '').trim();
      if (!text && queue.list.length === 0) {
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
      var localAtts = queue.list.slice();

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
          var mediaURL = detailTemplate.replace('{id}', sessionID) + '/media';
          var upload =
            localAtts.length > 0
              ? uploadComposerAttachments(mediaURL, localAtts)
              : Promise.resolve([]);
          return upload.then(function (refs) {
            clearPendingAttachments(localAtts);
            queue.clear();
            storePendingPrompt(sessionID, text, refs);
            var detailURL = detailTemplate.replace('{id}', sessionID);
            if (text) {
              detailURL += '?prompt=' + encodeURIComponent(text);
            }
            window.location.href = detailURL;
          });
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

  function initThreadSettings(root, onModelChange) {
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
          if (typeof onModelChange === 'function') {
            onModelChange();
          }
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

    var modelSel = root.querySelector('#chatagent-thread-model');
    var settingsEl = root.querySelector('[data-testid="chatagent-settings"]');
    var attachBtn = root.querySelector('#chatagent-attach-media');

    var queue = createAttachmentQueue({
      pendingEl: root.querySelector('#chatagent-pending-attachments'),
      mediaInput: root.querySelector('#chatagent-media-input'),
      attachBtn: attachBtn,
      modelSel: modelSel,
      settingsEl: settingsEl,
      inputEl: input,
      errorEl: errorEl,
    });

    initThreadSettings(root, function () {
      queue.syncVisibility();
    });
    // Re-sync after settings restore session model selection.
    queue.syncVisibility();

    function closeSession() {
      if (!closeURL) {
        return;
      }
      var closeBtns = root.querySelectorAll('[data-chatagent-close]');
      function setCloseDisabled(disabled) {
        for (var i = 0; i < closeBtns.length; i++) {
          closeBtns[i].disabled = disabled;
        }
      }
      function doClose() {
        setCloseDisabled(true);
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
            setCloseDisabled(false);
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

    var closeBtns = root.querySelectorAll('[data-chatagent-close]');
    for (var ci = 0; ci < closeBtns.length; ci++) {
      closeBtns[ci].addEventListener('click', closeSession);
    }

    function sendFollowUp() {
      var text = (input.value || '').trim();
      if (!text && queue.list.length === 0) {
        return;
      }
      var atts = queue.take();
      input.value = '';
      ns.streamMessage(messagesURL, text, root, null, approval, atts);
    }

    input.addEventListener('keydown', function (ev) {
      if (ev.key === 'Enter' && !ev.shiftKey) {
        ev.preventDefault();
        sendFollowUp();
      }
    });

    var pending = consumePendingPrompt(sessionID);
    ns.initContextControl(root);
    var messagesEl = root.querySelector('#chatagent-messages');
    var jumpBtn = root.querySelector('#chatagent-jump-bottom');
    if (ns.initMessageScroll) {
      ns.initMessageScroll(messagesEl, jumpBtn);
    }
    if (ns.enhanceCodeBlocks) {
      ns.enhanceCodeBlocks(messagesEl);
    }
    if (ns.hydrateTodosFromToolCards) {
      ns.hydrateTodosFromToolCards(root);
    }
    if (ns.refreshTodosFromServer) {
      ns.refreshTodosFromServer(root);
    }
    if (
      (pending.text || (pending.attachments && pending.attachments.length)) &&
      !threadHasHistory(root)
    ) {
      ns.streamMessage(
        messagesURL,
        pending.text,
        root,
        null,
        approval,
        pending.attachments || [],
      );
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
