(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var thinkingBodyClass =
    'chatagent-thinking-body chatagent-markdown markdown-body text-sm max-w-[92%]';
  var thinkingPlainClass = 'chatagent-thinking-body text-sm max-w-[92%]';
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
    ns.scrollMessages(container);
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
    ns.scrollMessages(container);
    return body;
  }

  function appendThinkingBlock(container) {
    var details = document.createElement('details');
    details.className = 'chatagent-thinking opacity-90';
    details.setAttribute('data-role', 'thinking');
    details.setAttribute('data-testid', 'chatagent-message-thinking');
    details.open = false;

    var summary = document.createElement('summary');
    summary.className =
      'chatagent-thinking-summary cursor-pointer text-xs text-base-content/50 select-none';
    summary.appendChild(document.createTextNode('Thinking'));

    var durationEl = document.createElement('span');
    durationEl.className = 'chatagent-duration text-base-content/40';
    durationEl.setAttribute('data-testid', 'chatagent-duration');
    summary.appendChild(durationEl);
    details.appendChild(summary);

    var body = document.createElement('div');
    body.className = 'chatagent-thinking-body mt-2';
    body.setAttribute('data-testid', 'chatagent-message-body');
    details.appendChild(body);
    container.appendChild(details);
    ns.scrollMessages(container);

    var startedAt = Date.now();
    var timer = setInterval(function () {
      durationEl.textContent = ' · ' + ns.formatDuration(Date.now() - startedAt);
    }, 100);

    return {
      body: body,
      stopTimer: function () {
        clearInterval(timer);
      },
      setDuration: function (ms) {
        clearInterval(timer);
        if (ms > 0) {
          durationEl.textContent = ' · ' + ns.formatDuration(ms);
        }
      },
    };
  }

  function toolKey(ev) {
    return (ev.subagent || '') + ':' + (ev.name || 'tool');
  }

  function upsertToolCard(container, ev, cards, anchorBody) {
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

      var duration = document.createElement('span');
      duration.className = 'chatagent-duration text-xs text-base-content/50';
      duration.setAttribute('data-testid', 'chatagent-duration');

      header.appendChild(badge);
      header.appendChild(status);
      header.appendChild(duration);
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
      insertStreamNode(container, wrap, anchorBody);

      card = {
        wrap: wrap,
        status: status,
        duration: duration,
        stdout: stdout,
        stderr: stderr,
        startedAt: Date.now(),
        timer: setInterval(function () {
          if (card.status.textContent === 'running' && card.startedAt) {
            card.duration.textContent =
              '· ' + ns.formatDuration(Date.now() - card.startedAt);
          }
        }, 100),
      };
      cards[key] = card;
    }

    if (ev.status) {
      card.status.textContent = ev.status;
    }
    if (ev.duration_ms > 0) {
      if (card.timer) {
        clearInterval(card.timer);
        card.timer = null;
      }
      card.duration.textContent = '· ' + ns.formatDuration(ev.duration_ms);
    } else if (card.status.textContent === 'running' && card.startedAt) {
      card.duration.textContent =
        '· ' + ns.formatDuration(Date.now() - card.startedAt);
    } else if (
      card.status.textContent === 'completed' ||
      card.status.textContent === 'error'
    ) {
      if (card.timer) {
        clearInterval(card.timer);
        card.timer = null;
      }
      if (!card.duration.textContent && card.startedAt) {
        card.duration.textContent =
          '· ' + ns.formatDuration(Date.now() - card.startedAt);
      }
    }
    if (ev.stdout) {
      card.stdout.textContent = (card.stdout.textContent || '') + ev.stdout;
      card.stdout.classList.remove('hidden');
    }
    if (ev.stderr) {
      card.stderr.textContent = (card.stderr.textContent || '') + ev.stderr;
      card.stderr.classList.remove('hidden');
    }
    ns.scrollMessages(container);
    return card;
  }

  function insertStreamNode(container, node, anchorBody) {
    var anchor = anchorBody ? anchorBody.parentElement : null;
    if (anchor && anchor.parentElement === container) {
      container.insertBefore(node, anchor);
      return;
    }
    container.appendChild(node);
  }

  function appendTurnMarker(container, step, durationMs, anchorBody) {
    var wrap = document.createElement('div');
    wrap.className = 'chat chat-start';
    wrap.setAttribute('data-role', 'turn-marker');
    wrap.setAttribute('data-testid', 'chatagent-turn-marker');

    var marker = document.createElement('div');
    marker.className = 'chatagent-turn-marker';
    var label = 'Step ' + (step || 1);
    if (durationMs > 0) {
      label += ' · ' + ns.formatDuration(durationMs);
    }
    marker.textContent = label;
    wrap.appendChild(marker);
    insertStreamNode(container, wrap, anchorBody);
    ns.scrollMessages(container);
  }

  function showRunDuration(messagesEl, durationMs) {
    if (!messagesEl || !durationMs || durationMs <= 0) {
      return;
    }
    var existing = messagesEl.querySelector(
      '[data-testid="chatagent-run-duration"]',
    );
    if (existing) {
      existing.remove();
    }
    var footer = document.createElement('div');
    footer.className =
      'chatagent-run-duration chatagent-duration text-xs text-base-content/50 text-center py-2';
    footer.setAttribute('data-testid', 'chatagent-run-duration');
    footer.textContent = 'Completed in ' + ns.formatDuration(durationMs);
    messagesEl.appendChild(footer);
    ns.scrollMessages(messagesEl);
  }

  function appendAssistantDuration(bodyEl, turnMs, runMs) {
    if (!bodyEl || (turnMs <= 0 && runMs <= 0)) {
      return;
    }
    var existing = bodyEl.querySelector(
      '[data-testid="chatagent-message-duration"]',
    );
    if (existing) {
      existing.remove();
    }
    var footer = document.createElement('div');
    footer.className =
      'mt-2 pt-2 border-t border-base-300/60 text-xs text-base-content/50 chatagent-duration';
    footer.setAttribute('data-testid', 'chatagent-message-duration');
    var parts = [];
    if (turnMs > 0) {
      parts.push('Turn ' + ns.formatDuration(turnMs));
    }
    if (runMs > 0) {
      parts.push('Total ' + ns.formatDuration(runMs));
    }
    footer.textContent = parts.join(' · ');
    bodyEl.appendChild(footer);
  }

  function applyAssistantDuration(bodyEl, turnMs, runMs) {
    if (!bodyEl) {
      return;
    }
    if (turnMs > 0) {
      bodyEl.dataset.turnDurationMs = String(turnMs);
    }
    if (runMs > 0) {
      bodyEl.dataset.runDurationMs = String(runMs);
    }
    appendAssistantDuration(
      bodyEl,
      turnMs > 0 ? turnMs : Number(bodyEl.dataset.turnDurationMs || 0),
      runMs > 0 ? runMs : Number(bodyEl.dataset.runDurationMs || 0),
    );
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
    ns.showError(el, message);
  }

  ns.streamMessage = function (messagesURL, text, threadRoot, onDone, approval) {
    var messagesEl = threadRoot.querySelector('#chatagent-messages');
    var errorEl = threadRoot.querySelector('#chatagent-thread-error');
    var cancelURL = threadRoot.getAttribute('data-cancel-url') || '';
    var assistantBody = null;
    var assistantText = '';
    var thinkingState = null;
    var thinkingText = '';
    var toolCards = {};
    var lastTurnDurationMs = 0;
    var lastRunDurationMs = 0;
    function syncAssistantDuration() {
      applyAssistantDuration(
        assistantBody,
        lastTurnDurationMs,
        lastRunDurationMs,
      );
    }
    var mdRenderer = ns.createStreamingMarkdownRenderer(
      threadRoot,
      function () {
        return assistantBody;
      },
      {
        onAfterRender: function () {
          syncAssistantDuration();
        },
      },
    );
    var ctxCtrl = ns.getContextControl(threadRoot);
    var thinkingRenderer = ns.createStreamingMarkdownRenderer(
      threadRoot,
      function () {
        return thinkingState ? thinkingState.body : null;
      },
      {
        renderedClass: thinkingBodyClass,
        plainClass: thinkingPlainClass,
      },
    );

    showThreadError(errorEl, '');
    ns.setRunning(true, threadRoot);
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

        function handleStreamEvent(ev) {
          if (ev.type === 'thinking') {
            if (!thinkingState) {
              thinkingState = appendThinkingBlock(messagesEl);
            }
            if (ev.status === 'completed') {
              if (thinkingState.setDuration) {
                thinkingState.setDuration(ev.duration_ms || 0);
              }
              return;
            }
            thinkingText += ev.text || '';
            thinkingRenderer.update(thinkingText);
            return;
          }
          if (ev.type === 'tool') {
            upsertToolCard(messagesEl, ev, toolCards, assistantBody);
            return;
          }
          if (ev.type === 'turn') {
            if (ev.duration_ms > 0) {
              lastTurnDurationMs = ev.duration_ms;
              syncAssistantDuration();
            }
            appendTurnMarker(
              messagesEl,
              ev.step,
              ev.duration_ms || 0,
              assistantBody,
            );
            return;
          }
          if (ev.type === 'delta') {
            var chunk = ev.text || '';
            if (ns.isToolPayloadText(chunk) || ns.isRunningToolStatus(chunk)) {
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
            if (!assistantBody && assistantText.trim()) {
              assistantBody = appendAssistantMessage(messagesEl, '', true);
            }
            if (assistantBody && assistantText.trim()) {
              mdRenderer.update(assistantText);
            }
            if (assistantBody && assistantBody.parentElement) {
              messagesEl.appendChild(assistantBody.parentElement);
            }
            if (ev.duration_ms > 0) {
              lastRunDurationMs = ev.duration_ms;
              showRunDuration(messagesEl, ev.duration_ms);
            }
            syncAssistantDuration();
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
        }

        function pump() {
          return reader.read().then(function (result) {
            if (result.value) {
              buffer += decoder.decode(result.value, { stream: true });
            }
            buffer = ns.flushSSEBuffer(buffer, handleStreamEvent);
            if (!result.done) {
              return pump();
            }
            if (buffer.trim()) {
              ns.flushSSEBuffer(buffer + '\n\n', handleStreamEvent);
            }
          });
        }
        return pump();
      })
      .then(function () {
        var finalize = Promise.resolve();
        if (thinkingState && thinkingText.trim()) {
          finalize = thinkingRenderer.finalize(thinkingText);
        }
        if (assistantBody && assistantText.trim()) {
          finalize = finalize.then(function () {
            return mdRenderer.finalize(assistantText);
          });
        } else {
          mdRenderer.cancel();
        }
        if (!thinkingState || !thinkingText.trim()) {
          thinkingRenderer.cancel();
        }
        if (thinkingState && thinkingState.stopTimer) {
          thinkingState.stopTimer();
        }
        return finalize.then(function () {
          syncAssistantDuration();
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
      })
      .catch(function (err) {
        showThreadError(errorEl, err.message || 'Request failed');
      })
      .finally(function () {
        ns.setRunning(false, threadRoot);
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
})();
