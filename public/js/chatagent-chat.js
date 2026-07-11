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

  var RING_CIRCUMFERENCE = 62.832;

  var CONTEXT_CATEGORY_LABELS = {
    system_prompt: 'System prompt',
    system_tools: 'Tool definitions',
    skills: 'Skills',
    messages: 'Conversation',
    autocompact_buffer: 'Autocompact buffer',
    free_space: 'Free space',
  };

  var CONTEXT_CATEGORY_COLORS = {
    system_prompt: '#9ca3af',
    system_tools: '#a78bfa',
    skills: '#fb923c',
    messages: '#b4534a',
    autocompact_buffer: '#64748b',
    free_space: '#1f2937',
  };

  var CONTEXT_BAR_ORDER = [
    'system_prompt',
    'system_tools',
    'skills',
    'messages',
    'autocompact_buffer',
    'free_space',
  ];

  var contextControls = new WeakMap();

  function formatTokenCount(n) {
    var value = Number(n) || 0;
    if (value >= 1000) {
      var scaled = value / 1000;
      if (scaled >= 100) {
        return Math.round(scaled) + 'K';
      }
      return scaled.toFixed(1).replace(/\.0$/, '') + 'K';
    }
    return String(Math.round(value));
  }

  function formatTokenRange(used, windowSize) {
    return (
      '~' +
      formatTokenCount(used) +
      ' / ' +
      formatTokenCount(windowSize) +
      ' Tokens'
    );
  }

  function contextUsagePercent(total, windowSize, reported) {
    if (windowSize > 0 && total > 0) {
      return (total / windowSize) * 100;
    }
    if (typeof reported === 'number' && !Number.isNaN(reported)) {
      return reported;
    }
    return 0;
  }

  function updateContextRing(ringWrap, percent) {
    if (!ringWrap) {
      return;
    }
    var progress = ringWrap.querySelector('.chatagent-context-ring-progress');
    if (!progress) {
      return;
    }
    var clamped = Math.max(0, Math.min(100, percent || 0));
    progress.setAttribute(
      'stroke-dashoffset',
      String(RING_CIRCUMFERENCE * (1 - clamped / 100)),
    );
  }

  function updateContextSummary(popover, report) {
    if (!popover || !report) {
      return;
    }
    var percentEl = popover.querySelector('#chatagent-context-percent');
    var tokensEl = popover.querySelector('#chatagent-context-tokens');
    var pct = Math.round(report.total_percent || 0);
    if (percentEl) {
      percentEl.textContent = pct + '% Full';
    }
    if (tokensEl) {
      tokensEl.textContent = formatTokenRange(
        report.total_tokens || 0,
        report.context_window || 0,
      );
    }
  }

  function renderContextUsage(popover, report) {
    if (!popover || !report) {
      return;
    }
    updateContextSummary(popover, report);

    var barEl = popover.querySelector('#chatagent-context-bar');
    var legendEl = popover.querySelector('#chatagent-context-legend');
    if (!barEl || !legendEl) {
      return;
    }

    var byID = {};
    (report.categories || []).forEach(function (cat) {
      byID[cat.id] = cat;
    });

    barEl.textContent = '';
    CONTEXT_BAR_ORDER.forEach(function (id) {
      var cat = byID[id];
      if (!cat || cat.percent <= 0) {
        return;
      }
      var seg = document.createElement('div');
      seg.className = 'chatagent-context-bar-segment';
      seg.setAttribute('data-category', id);
      seg.style.width = cat.percent + '%';
      seg.style.backgroundColor = CONTEXT_CATEGORY_COLORS[id] || '#6b7280';
      barEl.appendChild(seg);
    });

    legendEl.textContent = '';
    CONTEXT_BAR_ORDER.forEach(function (id) {
      if (id === 'free_space') {
        return;
      }
      var cat = byID[id];
      if (!cat) {
        return;
      }
      var row = document.createElement('div');
      row.className = 'chatagent-context-legend-row';

      var labelWrap = document.createElement('div');
      labelWrap.className = 'chatagent-context-legend-label';

      var swatch = document.createElement('span');
      swatch.className = 'chatagent-context-swatch';
      swatch.style.backgroundColor = CONTEXT_CATEGORY_COLORS[id] || '#6b7280';

      var label = document.createElement('span');
      label.textContent = CONTEXT_CATEGORY_LABELS[id] || cat.label || id;

      labelWrap.appendChild(swatch);
      labelWrap.appendChild(label);

      var tokens = document.createElement('span');
      tokens.className = 'chatagent-context-legend-tokens';
      tokens.textContent = formatTokenCount(cat.tokens);

      row.appendChild(labelWrap);
      row.appendChild(tokens);
      legendEl.appendChild(row);

      if (id === 'skills' && report.skills && report.skills.length > 0) {
        report.skills.forEach(function (skill) {
          var skillRow = document.createElement('div');
          skillRow.className =
            'chatagent-context-legend-row chatagent-context-skill-row';

          var skillLabelWrap = document.createElement('div');
          skillLabelWrap.className = 'chatagent-context-legend-label';

          var skillLabel = document.createElement('span');
          skillLabel.textContent = skill.name || 'skill';
          skillLabelWrap.appendChild(skillLabel);

          var skillTokens = document.createElement('span');
          skillTokens.className = 'chatagent-context-legend-tokens';
          skillTokens.textContent = formatTokenCount(skill.tokens);

          skillRow.appendChild(skillLabelWrap);
          skillRow.appendChild(skillTokens);
          legendEl.appendChild(skillRow);
        });
      }
    });
  }

  function initContextControl(threadRoot) {
    var contextURL = threadRoot.getAttribute('data-context-url') || '';
    var ringWrap = threadRoot.querySelector('.chatagent-context-ring-wrap');
    var popover = threadRoot.querySelector('.chatagent-context-popover');
    if (!contextURL || !ringWrap || !popover) {
      return null;
    }

    var ringBtn = ringWrap.querySelector('.chatagent-context-ring');
    var closeBtn = popover.querySelector('.chatagent-context-close');
    var errorEl = popover.querySelector('#chatagent-context-error');
    var cachedReport = null;
    var open = false;
    var outsideListener = null;

    function showPopoverError(message) {
      showError(errorEl, message);
    }

    function setOpen(next) {
      open = next;
      if (!popover) {
        return;
      }
      popover.classList.toggle('hidden', !open);
      if (outsideListener) {
        document.removeEventListener('click', outsideListener);
        outsideListener = null;
      }
      if (open) {
        outsideListener = function (event) {
          if (
            ringWrap.contains(event.target) ||
            popover.contains(event.target)
          ) {
            return;
          }
          setOpen(false);
        };
        window.setTimeout(function () {
          if (open && outsideListener) {
            document.addEventListener('click', outsideListener);
          }
        }, 0);
      }
    }

    function applyReport(report) {
      cachedReport = report;
      updateContextRing(ringWrap, report.total_percent || 0);
      if (open) {
        renderContextUsage(popover, report);
      }
    }

    function fetchContextReport(forceRender) {
      return fetch(contextURL, { headers: { Accept: 'application/json' } })
        .then(function (res) {
          if (!res.ok) {
            return res
              .json()
              .catch(function () {
                return {};
              })
              .then(function (data) {
                throw new Error(
                  (data && data.error) || 'Failed to load context usage',
                );
              });
          }
          return res.json();
        })
        .then(function (report) {
          showPopoverError('');
          applyReport(report);
          if (forceRender && open) {
            renderContextUsage(popover, report);
          }
          return report;
        })
        .catch(function (err) {
          if (open) {
            showPopoverError(err.message || 'Failed to load context usage');
          }
          return null;
        });
    }

    function togglePopover() {
      if (open) {
        setOpen(false);
        return;
      }
      setOpen(true);
      showPopoverError('');
      if (cachedReport) {
        renderContextUsage(popover, cachedReport);
        return;
      }
      fetchContextReport(true);
    }

    if (ringBtn) {
      ringBtn.addEventListener('click', function (event) {
        event.stopPropagation();
        togglePopover();
      });
    }
    if (closeBtn) {
      closeBtn.addEventListener('click', function (event) {
        event.stopPropagation();
        setOpen(false);
      });
    }

    fetchContextReport(false);

    var control = {
      handleUsage: function (ev) {
        var pct = contextUsagePercent(
          ev.total_tokens,
          ev.context_window,
          ev.context_percent,
        );
        updateContextRing(ringWrap, pct);
        if (open && popover) {
          var summary = cachedReport
            ? Object.assign({}, cachedReport)
            : {
                total_tokens: ev.total_tokens || 0,
                context_window: ev.context_window || 0,
                total_percent: pct,
                categories: [],
                skills: [],
              };
          summary.total_tokens = ev.total_tokens || summary.total_tokens;
          summary.context_window = ev.context_window || summary.context_window;
          summary.total_percent = pct;
          updateContextSummary(popover, summary);
        }
      },
      onRunComplete: function () {
        if (open) {
          fetchContextReport(true);
        } else {
          fetchContextReport(false);
        }
      },
    };

    contextControls.set(threadRoot, control);
    return control;
  }

  function getContextControl(threadRoot) {
    return contextControls.get(threadRoot) || null;
  }

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

  function isApprovalStatusMessage(message) {
    var trimmed = (message || '').trim();
    if (!trimmed) {
      return false;
    }
    return /^(Approved|Denied|Timed out)/i.test(trimmed);
  }

  function showThreadError(el, message) {
    if (isApprovalStatusMessage(message)) {
      return;
    }
    showError(el, message);
  }

  function streamMessage(messagesURL, text, threadRoot, onDone, approval) {
    var messagesEl = threadRoot.querySelector('#chatagent-messages');
    var errorEl = threadRoot.querySelector('#chatagent-thread-error');
    var cancelURL = threadRoot.getAttribute('data-cancel-url') || '';
    var assistantBody = null;
    var assistantText = '';
    var thinkingBody = null;
    var thinkingText = '';
    var toolCards = {};
    var ctxCtrl = getContextControl(threadRoot);
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

    showThreadError(errorEl, '');
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
              if (ev.type === 'usage') {
                if (ctxCtrl) {
                  ctxCtrl.handleUsage(ev);
                }
                return;
              }
              if (
                approval &&
                (ev.type === 'confirm' ||
                  ev.type === 'confirm_resolved' ||
                  ev.type === 'canceled')
              ) {
                approval.handleStreamEvent(ev);
                return;
              }
              if (ev.type === 'error') {
                showThreadError(errorEl, ev.message || 'Run failed');
              } else if (ev.type === 'canceled') {
                showThreadError(errorEl, ev.message || 'Run canceled');
              }
            });
            return pump();
          });
        }
        return pump();
      })
      .catch(function (err) {
        showThreadError(errorEl, err.message || 'Request failed');
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
          if (ctxCtrl) {
            ctxCtrl.onRunComplete();
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

  function initApproval(panel) {
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
    var approval = initApproval(
      root.querySelector('#chatagent-approval-panel'),
    );

    function sendFollowUp() {
      var text = (input.value || '').trim();
      if (!text) {
        return;
      }
      input.value = '';
      streamMessage(messagesURL, text, root, null, approval);
    }

    input.addEventListener('keydown', function (ev) {
      if (ev.key === 'Enter' && !ev.shiftKey) {
        ev.preventDefault();
        sendFollowUp();
      }
    });

    var pending = consumePendingPrompt(sessionID);
    initContextControl(root);
    if (pending && !threadHasHistory(root)) {
      streamMessage(messagesURL, pending, root, null, approval);
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
      initApproval(panel);
    });
})();
