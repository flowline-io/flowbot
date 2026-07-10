(function () {
  'use strict';

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

  function showError(el, message) {
    if (!el) {
      return;
    }
    if (!message) {
      el.classList.add('hidden');
      el.textContent = '';
      return;
    }
    el.textContent = message;
    el.classList.remove('hidden');
  }

  function setRunning(running, threadRoot) {
    var cancelBtn = threadRoot
      ? threadRoot.querySelector('#chatagent-cancel-run')
      : null;
    var input = threadRoot
      ? threadRoot.querySelector('#chatagent-followup-input')
      : null;
    if (input) {
      input.disabled = running;
    }
    if (cancelBtn) {
      cancelBtn.classList.toggle('hidden', !running);
    }
  }

  function scrollMessages(container) {
    container.scrollTop = container.scrollHeight;
  }

  function isToolPayloadText(text) {
    var trimmed = (text || '').trim();
    if (!trimmed) {
      return false;
    }
    return (
      trimmed.indexOf('[{"id":') === 0 ||
      trimmed.indexOf('[{"id"') === 0 ||
      trimmed.indexOf('{"id":"call_') === 0
    );
  }

  function isRunningToolStatus(text) {
    var trimmed = (text || '').trim();
    return (
      trimmed.indexOf('Running tool:') === 0 ||
      trimmed.indexOf('Delegating to subagent:') === 0
    );
  }

  function appendUserMessage(container, text) {
    var wrap = document.createElement('div');
    wrap.className = 'chat chat-end';
    wrap.setAttribute('data-role', 'user');
    wrap.setAttribute('data-testid', 'chatagent-message-user');

    var body = document.createElement('div');
    body.className =
      'chat-bubble bg-primary text-primary-content whitespace-pre-wrap text-sm max-w-[92%]';
    body.setAttribute('data-testid', 'chatagent-message-body');
    body.textContent = text;
    wrap.appendChild(body);
    container.appendChild(wrap);
    scrollMessages(container);
  }

  function appendAssistantMessage(container, text, streaming) {
    var wrap = document.createElement('div');
    wrap.className = 'chat chat-start';
    if (streaming) {
      wrap.classList.add('opacity-80');
    }
    wrap.setAttribute('data-role', 'assistant');
    wrap.setAttribute('data-testid', 'chatagent-message-assistant');

    var body = document.createElement('div');
    body.className =
      'chat-bubble bg-base-100 border border-base-300 whitespace-pre-wrap text-sm max-w-[92%]';
    body.setAttribute('data-testid', 'chatagent-message-body');
    body.textContent = text;
    wrap.appendChild(body);
    container.appendChild(wrap);
    scrollMessages(container);
    return body;
  }

  function appendThinkingBlock(container, text) {
    var details = document.createElement('details');
    details.className = 'chatagent-thinking opacity-90';
    details.setAttribute('data-role', 'thinking');
    details.setAttribute('data-testid', 'chatagent-message-thinking');
    details.open = false;

    var summary = document.createElement('summary');
    summary.className =
      'chatagent-thinking-summary cursor-pointer text-xs text-base-content/50 select-none';
    summary.textContent = 'Thinking';
    details.appendChild(summary);

    var body = document.createElement('div');
    body.className = 'chatagent-thinking-body mt-2';
    body.setAttribute('data-testid', 'chatagent-message-body');
    body.textContent = text;
    details.appendChild(body);
    container.appendChild(details);
    scrollMessages(container);
    return body;
  }

  function toolKey(ev) {
    return (ev.subagent || '') + ':' + (ev.name || 'tool');
  }

  function upsertToolCard(container, ev, cards) {
    var key = toolKey(ev);
    var card = cards[key];
    if (!card) {
      var wrap = document.createElement('div');
      wrap.className = 'chat chat-start';
      wrap.setAttribute('data-role', 'tool');
      wrap.setAttribute('data-testid', 'chatagent-message-tool');

      var bubble = document.createElement('div');
      bubble.className =
        'chat-bubble bg-base-200 border border-base-300 max-w-[92%] text-sm';

      var header = document.createElement('div');
      header.className = 'flex items-center gap-2 flex-wrap';

      var badge = document.createElement('span');
      badge.className = 'badge badge-sm badge-outline font-mono';
      badge.setAttribute('data-testid', 'chatagent-tool-name');
      badge.textContent = ev.name || 'tool';

      var status = document.createElement('span');
      status.className = 'text-xs text-base-content/60';
      status.setAttribute('data-testid', 'chatagent-tool-status');
      status.textContent = ev.status || 'running';

      header.appendChild(badge);
      header.appendChild(status);
      bubble.appendChild(header);

      var stdout = document.createElement('pre');
      stdout.className =
        'mt-2 text-xs whitespace-pre-wrap overflow-x-auto max-h-56 bg-base-300/40 rounded p-2 font-mono hidden';
      stdout.setAttribute('data-testid', 'chatagent-tool-stdout');

      var stderr = document.createElement('pre');
      stderr.className =
        'mt-2 text-xs whitespace-pre-wrap overflow-x-auto max-h-32 bg-error/10 text-error rounded p-2 font-mono hidden';
      stderr.setAttribute('data-testid', 'chatagent-tool-stderr');

      bubble.appendChild(stdout);
      bubble.appendChild(stderr);
      wrap.appendChild(bubble);
      container.appendChild(wrap);

      card = { wrap: wrap, status: status, stdout: stdout, stderr: stderr };
      cards[key] = card;
    }

    if (ev.status) {
      card.status.textContent = ev.status;
    }
    if (ev.stdout) {
      card.stdout.textContent = (card.stdout.textContent || '') + ev.stdout;
      card.stdout.classList.remove('hidden');
    }
    if (ev.stderr) {
      card.stderr.textContent = (card.stderr.textContent || '') + ev.stderr;
      card.stderr.classList.remove('hidden');
    }
    scrollMessages(container);
    return card;
  }

  function parseSSEChunk(buffer, onEvent) {
    var parts = buffer.split('\n\n');
    var rest = parts.pop() || '';
    parts.forEach(function (frame) {
      var line = frame.split('\n').find(function (l) {
        return l.indexOf('data: ') === 0;
      });
      if (!line) {
        return;
      }
      try {
        onEvent(JSON.parse(line.slice(6)));
      } catch {
        /* ignore malformed frames */
      }
    });
    return rest;
  }

  function inOpenCodeFence(text) {
    var matches = (text || '').match(/```/g);
    return matches ? matches.length % 2 === 1 : false;
  }

  function markdownRenderDelay(text) {
    var n = (text || '').length;
    if (inOpenCodeFence(text)) {
      if (n < 1000) {
        return 400;
      }
      if (n <= 5000) {
        return 600;
      }
      return 800;
    }
    if (n < 1000) {
      return 120;
    }
    if (n <= 5000) {
      return 250;
    }
    return 500;
  }

  var markdownBubbleClass =
    'chat-bubble bg-base-100 border border-base-300 chatagent-markdown markdown-body text-sm max-w-[92%]';

  var thinkingBodyClass =
    'chatagent-thinking-body chatagent-markdown markdown-body text-sm max-w-[92%]';

  var thinkingPlainClass = 'chatagent-thinking-body text-sm max-w-[92%]';

  function createStreamingMarkdownRenderer(threadRoot, getBodyEl, options) {
    options = options || {};
    var renderedClass = options.renderedClass || markdownBubbleClass;
    var plainClass =
      options.plainClass ||
      markdownBubbleClass.replace(
        ' chatagent-markdown markdown-body',
        ' whitespace-pre-wrap',
      );
    var renderURL = threadRoot.getAttribute('data-render-markdown-url') || '';
    var timer = null;
    var latestSeq = 0;
    var pendingText = '';

    function scroll() {
      var messagesEl = threadRoot.querySelector('#chatagent-messages');
      if (messagesEl) {
        scrollMessages(messagesEl);
      }
    }

    function applyHTML(bodyEl, html) {
      bodyEl.className = renderedClass;
      bodyEl.innerHTML = html;
      bodyEl.dataset.mdRendered = '1';
      scroll();
    }

    function showPlainText(bodyEl, text) {
      bodyEl.className = plainClass;
      delete bodyEl.dataset.mdRendered;
      bodyEl.textContent = text;
      scroll();
    }

    function flush() {
      var bodyEl = getBodyEl();
      if (!bodyEl || !(pendingText || '').trim()) {
        return Promise.resolve();
      }
      if (!renderURL) {
        showPlainText(bodyEl, pendingText);
        return Promise.resolve();
      }
      latestSeq += 1;
      var fetchSeq = latestSeq;
      return fetch(renderURL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: pendingText }),
      })
        .then(function (res) {
          if (!res.ok) {
            return null;
          }
          return res.json();
        })
        .then(function (data) {
          if (fetchSeq !== latestSeq) {
            return;
          }
          var currentBody = getBodyEl();
          if (!currentBody || !data || !data.html) {
            return;
          }
          applyHTML(currentBody, data.html);
        })
        .catch(function () {
          if (fetchSeq !== latestSeq) {
            return;
          }
          var currentBody = getBodyEl();
          if (currentBody && !currentBody.dataset.mdRendered) {
            showPlainText(currentBody, pendingText);
          }
        });
    }

    function cancelTimer() {
      if (timer) {
        clearTimeout(timer);
        timer = null;
      }
    }

    return {
      update: function (text) {
        pendingText = text || '';
        var bodyEl = getBodyEl();
        if (!bodyEl || !pendingText.trim()) {
          return;
        }
        if (!bodyEl.dataset.mdRendered) {
          bodyEl.textContent = pendingText;
          scroll();
        }
        cancelTimer();
        if (!renderURL) {
          return;
        }
        timer = setTimeout(function () {
          timer = null;
          flush();
        }, markdownRenderDelay(pendingText));
      },
      finalize: function (text) {
        pendingText = text || pendingText;
        cancelTimer();
        return flush();
      },
      cancel: function () {
        cancelTimer();
        latestSeq += 1;
      },
    };
  }

  function streamMessage(messagesURL, text, threadRoot, onDone) {
    var messagesEl = threadRoot.querySelector('#chatagent-messages');
    var errorEl = threadRoot.querySelector('#chatagent-thread-error');
    var cancelURL = threadRoot.getAttribute('data-cancel-url') || '';
    var assistantBody = null;
    var assistantText = '';
    var thinkingBody = null;
    var thinkingText = '';
    var toolCards = {};
    var mdRenderer = createStreamingMarkdownRenderer(threadRoot, function () {
      return assistantBody;
    });
    var thinkingRenderer = createStreamingMarkdownRenderer(
      threadRoot,
      function () {
        return thinkingBody;
      },
      {
        renderedClass: thinkingBodyClass,
        plainClass: thinkingPlainClass,
      },
    );

    showError(errorEl, '');
    setRunning(true, threadRoot);
    appendUserMessage(messagesEl, text);

    fetch(messagesURL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      },
      body: JSON.stringify({ text: text }),
    })
      .then(function (res) {
        if (res.status === 409) {
          throw new Error('A run is already in progress.');
        }
        if (!res.ok) {
          return res
            .json()
            .catch(function () {
              return {};
            })
            .then(function (data) {
              throw new Error((data && data.error) || 'Request failed');
            });
        }
        if (!res.body || !res.body.getReader) {
          throw new Error('Streaming is not supported in this browser.');
        }
        var reader = res.body.getReader();
        var decoder = new TextDecoder();
        var buffer = '';

        function pump() {
          return reader.read().then(function (result) {
            if (result.done) {
              return;
            }
            buffer += decoder.decode(result.value, { stream: true });
            buffer = parseSSEChunk(buffer, function (ev) {
              if (ev.type === 'thinking') {
                if (!thinkingBody) {
                  thinkingBody = appendThinkingBlock(messagesEl, '');
                }
                thinkingText += ev.text || '';
                thinkingRenderer.update(thinkingText);
                return;
              }
              if (ev.type === 'tool') {
                upsertToolCard(messagesEl, ev, toolCards);
                return;
              }
              if (ev.type === 'delta') {
                var chunk = ev.text || '';
                if (isToolPayloadText(chunk) || isRunningToolStatus(chunk)) {
                  return;
                }
                if (!assistantBody) {
                  assistantBody = appendAssistantMessage(messagesEl, '', true);
                }
                assistantText += chunk;
                mdRenderer.update(assistantText);
                return;
              }
              if (ev.type === 'done') {
                if (ev.text) {
                  assistantText = ev.text;
                }
                if (assistantBody && assistantText.trim()) {
                  mdRenderer.update(assistantText);
                }
                return;
              }
              if (ev.type === 'error') {
                showError(errorEl, ev.message || 'Run failed');
              } else if (ev.type === 'canceled') {
                showError(errorEl, ev.message || 'Run canceled');
              }
            });
            return pump();
          });
        }
        return pump();
      })
      .catch(function (err) {
        showError(errorEl, err.message || 'Request failed');
      })
      .finally(function () {
        setRunning(false, threadRoot);
        var finalize = Promise.resolve();
        if (thinkingBody && thinkingText.trim()) {
          finalize = thinkingRenderer.finalize(thinkingText);
        }
        if (assistantBody && assistantText.trim()) {
          finalize = finalize.then(function () {
            return mdRenderer.finalize(assistantText);
          });
        } else {
          mdRenderer.cancel();
        }
        if (!thinkingBody || !thinkingText.trim()) {
          thinkingRenderer.cancel();
        }
        finalize.finally(function () {
          if (assistantBody) {
            assistantBody.parentElement.classList.remove('opacity-80');
          }
          if (typeof onDone === 'function') {
            onDone();
          }
        });
      });

    if (cancelURL) {
      var cancelBtn = threadRoot.querySelector('#chatagent-cancel-run');
      if (cancelBtn) {
        cancelBtn.addEventListener('click', function () {
          fetch(cancelURL, { method: 'POST' }).catch(function () {});
        });
      }
    }
  }

  function initApproval(panel) {
    if (!panel) {
      return;
    }
    var sessionID = panel.getAttribute('data-session-id');
    var confirmURL = panel.getAttribute('data-confirm-url');
    var eventsURL = panel.getAttribute('data-events-url');
    if (!sessionID || !confirmURL || !eventsURL) {
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
        showError(errorEl, 'Enter a prompt to start.');
        return;
      }
      showError(errorEl, '');
      startBtn.disabled = true;
      fetch(createURL, { method: 'POST' })
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
          showError(errorEl, err.message || 'Failed to start');
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
    var input = root.querySelector('#chatagent-followup-input');
    if (!sessionID || !messagesURL || !input) {
      return;
    }

    function sendFollowUp() {
      var text = (input.value || '').trim();
      if (!text) {
        return;
      }
      input.value = '';
      streamMessage(messagesURL, text, root);
    }

    input.addEventListener('keydown', function (ev) {
      if (ev.key === 'Enter' && !ev.shiftKey) {
        ev.preventDefault();
        sendFollowUp();
      }
    });

    var pending = consumePendingPrompt(sessionID);
    if (pending && !threadHasHistory(root)) {
      streamMessage(messagesURL, pending, root);
    }
  }

  document
    .querySelectorAll('[data-chatagent-root="composer"]')
    .forEach(initComposer);
  document
    .querySelectorAll('[data-chatagent-root="thread"]')
    .forEach(initThread);
  initApproval(document.getElementById('chatagent-approval-panel'));
})();
