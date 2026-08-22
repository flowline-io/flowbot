(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var thinkingBodyClass =
    'chatagent-thinking-body chatagent-markdown markdown-body text-sm';
  var thinkingPlainClass = 'chatagent-thinking-body text-sm';
  var assistantBodyClass =
    'chatagent-assistant-body chatagent-markdown markdown-body text-sm';
  var assistantPlainClass =
    'chatagent-assistant-body whitespace-pre-wrap text-sm';
  var copyMarkdownIconSVG =
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" class="w-3 h-3" aria-hidden="true"><path d="M5 6.5A1.5 1.5 0 0 1 6.5 5h6A1.5 1.5 0 0 1 14 6.5v6a1.5 1.5 0 0 1-1.5 1.5h-6A1.5 1.5 0 0 1 5 12.5v-6Z"></path><path d="M3.5 2A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11V6.5a3 3 0 0 1 3-3H11A1.5 1.5 0 0 0 9.5 2h-6Z"></path></svg>';
  var thinkIconSVG =
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" class="chatagent-step-icon" aria-hidden="true"><circle cx="8" cy="8" r="5" stroke="currentColor" stroke-width="1"></circle><circle cx="8" cy="8" r="1.15" fill="currentColor"></circle></svg>';
  var toolIconSVG =
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" class="chatagent-step-icon" aria-hidden="true"><rect x="3.5" y="3.5" width="9" height="9" rx="1.25" stroke="currentColor" stroke-width="1"></rect><path d="M6.25 8h3.5M8 6.25v3.5" stroke="currentColor" stroke-width="1" stroke-linecap="round"></path></svg>';

  function oneLinePreview(text, limit) {
    limit = limit || 72;
    text = String(text || '').replace(/^\s+|\s+$/g, '');
    var nl = text.search(/[\n\r]/);
    if (nl >= 0) {
      text = text.slice(0, nl).replace(/^\s+|\s+$/g, '');
    }
    text = text.replace(/\s+/g, ' ');
    if (!text) {
      return '';
    }
    var chars = Array.from(text);
    if (chars.length <= limit) {
      return text;
    }
    if (limit === 1) {
      return '…';
    }
    return chars.slice(0, limit - 1).join('') + '…';
  }

  function setStepPreview(el, text) {
    if (!el) {
      return;
    }
    var preview = oneLinePreview(text);
    el.textContent = preview;
    el.hidden = !preview;
    var dot = el.previousElementSibling;
    if (dot && dot.classList.contains('chatagent-step-dot')) {
      dot.hidden = !preview;
    }
  }

  function chatAgentToolPreview(text, stdout) {
    var preview = oneLinePreview(text);
    if (preview) {
      return preview;
    }
    return oneLinePreview(stdout);
  }

  function createStepSummary(opts) {
    var summary = document.createElement('summary');
    summary.className = opts.summaryClass;
    summary.innerHTML = opts.iconHTML;

    var label = document.createElement('span');
    label.className = opts.labelClass || 'chatagent-step-label';
    if (opts.labelTestId) {
      label.setAttribute('data-testid', opts.labelTestId);
    }
    label.textContent = opts.label || '';
    summary.appendChild(label);

    var previewDot = document.createElement('span');
    previewDot.className = 'chatagent-step-dot';
    previewDot.setAttribute('aria-hidden', 'true');
    previewDot.textContent = '·';
    previewDot.hidden = true;
    summary.appendChild(previewDot);

    var previewEl = document.createElement('span');
    previewEl.className = 'chatagent-step-preview';
    previewEl.setAttribute('data-testid', 'chatagent-step-preview');
    previewEl.hidden = true;
    summary.appendChild(previewEl);

    var durationEl = document.createElement('span');
    durationEl.className =
      opts.durationClass || 'chatagent-duration text-xs text-base-content/50';
    durationEl.setAttribute('data-testid', 'chatagent-duration');
    summary.appendChild(durationEl);

    return {
      summary: summary,
      previewEl: previewEl,
      durationEl: durationEl,
    };
  }

  function createCopyButton(opts) {
    var btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'btn btn-ghost btn-xs btn-square ' + opts.cssClass;
    btn.title = opts.title;
    btn.setAttribute('aria-label', opts.title);
    btn.setAttribute('data-testid', opts.testId);
    btn.setAttribute('data-clip-copy', '');
    if (opts.plainText) {
      btn.setAttribute('data-clip-text', opts.text || '');
    } else {
      btn.setAttribute('data-clip-markdown', opts.text || '');
    }
    btn.innerHTML = copyMarkdownIconSVG;
    return btn;
  }

  function setToolStatus(card, status) {
    if (!card) {
      return;
    }
    card.status = status || '';
    if (!card.details) {
      return;
    }
    if (card.status) {
      card.details.setAttribute('data-tool-status', card.status);
    } else {
      card.details.removeAttribute('data-tool-status');
    }
  }
  function isImageAttachment(item) {
    if (!item) {
      return false;
    }
    if (item.kind === 'image') {
      return true;
    }
    var mime = item.mime_type || (item.file && item.file.type) || '';
    return mime.indexOf('image/') === 0;
  }

  function mediaPreviewURL(mediaURL, fileID) {
    if (!mediaURL || !fileID) {
      return '';
    }
    return mediaURL.replace(/\/$/, '') + '/' + encodeURIComponent(fileID);
  }

  function appendUserMessage(container, text, attachments, mediaURL) {
    var wrap = document.createElement('div');
    wrap.className = 'chat chat-end';
    wrap.setAttribute('data-role', 'user');
    wrap.setAttribute('data-testid', 'chatagent-message-user');

    var body = document.createElement('div');
    body.className = 'chatagent-user-bubble whitespace-pre-wrap text-sm';
    body.setAttribute('data-testid', 'chatagent-message-body');

    var atts = Array.isArray(attachments) ? attachments : [];
    if (atts.length) {
      var gallery = document.createElement('div');
      gallery.className = 'chatagent-message-attachments';
      gallery.setAttribute('data-testid', 'chatagent-message-attachments');
      atts.forEach(function (item) {
        if (isImageAttachment(item)) {
          var src = '';
          if (item.previewURL) {
            src = item.previewURL;
          } else if (item.file && isImageAttachment(item)) {
            src = URL.createObjectURL(item.file);
          } else if (item.file_id) {
            src = mediaPreviewURL(mediaURL, item.file_id);
          }
          if (src) {
            var safeSrc = ns.safeMediaSrc(src);
            if (!safeSrc) {
              return;
            }
            var img = document.createElement('img');
            img.src = encodeURI(safeSrc.replace(/[<>"']/g, ''));
            img.alt = '';
            img.className = 'chatagent-message-attach-img';
            img.setAttribute('data-testid', 'chatagent-message-attach-img');
            gallery.appendChild(img);
            return;
          }
        }
        var chip = document.createElement('span');
        chip.className = 'chatagent-message-attach-file';
        chip.textContent =
          '[' +
          (item.kind || 'media') +
          '] ' +
          (item.file_id || item.name || 'attachment');
        gallery.appendChild(chip);
      });
      body.appendChild(gallery);
    }

    if (text) {
      var textEl = document.createElement('div');
      textEl.className = 'chatagent-message-text';
      textEl.textContent = text;
      body.appendChild(textEl);
    } else if (!atts.length) {
      body.textContent = '';
    }

    wrap.appendChild(body);
    if (text) {
      wrap.appendChild(
        createCopyButton({
          cssClass: 'chatagent-copy-user',
          title: 'Copy',
          testId: 'chatagent-copy-user',
          text: text,
          plainText: true,
        }),
      );
    }
    container.appendChild(wrap);
    if (ns.stickMessagesToBottom) {
      ns.stickMessagesToBottom(container);
    } else {
      ns.scrollMessages(container);
    }
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
    body.className = assistantPlainClass;
    body.setAttribute('data-testid', 'chatagent-message-body');
    body.textContent = text;
    wrap.appendChild(body);
    container.appendChild(wrap);
    ns.scrollMessages(container);
    return body;
  }

  function ensureMessageMeta(bodyEl) {
    var wrap = bodyEl && bodyEl.parentElement;
    if (!wrap) {
      return null;
    }
    var meta = wrap.querySelector(
      ':scope > [data-testid="chatagent-message-meta"]',
    );
    if (!meta) {
      meta = document.createElement('div');
      meta.className = 'chatagent-message-meta';
      meta.setAttribute('data-testid', 'chatagent-message-meta');
      wrap.appendChild(meta);
    }
    return meta;
  }

  function ensureCopyMarkdownButton(bodyEl, markdown) {
    if (!bodyEl || !(markdown || '').trim()) {
      return;
    }
    var meta = ensureMessageMeta(bodyEl);
    if (!meta) {
      return;
    }
    var btn = meta.querySelector('[data-testid="chatagent-copy-md"]');
    if (!btn) {
      btn = createCopyButton({
        cssClass: 'chatagent-copy-md',
        title: flowbotI18n('client.chatagent.copy_md', 'Copy markdown'),
        testId: 'chatagent-copy-md',
        text: markdown,
      });
      meta.appendChild(btn);
    }
    btn.setAttribute('data-clip-markdown', markdown);
  }

  function appendThinkingBlock(container) {
    var details = document.createElement('details');
    details.className = 'chatagent-thinking chatagent-step';
    details.setAttribute('data-role', 'thinking');
    details.setAttribute('data-testid', 'chatagent-message-thinking');
    details.open = false;

    var step = createStepSummary({
      summaryClass:
        'chatagent-thinking-summary chatagent-step-summary cursor-pointer text-xs text-base-content/50 select-none',
      iconHTML: thinkIconSVG,
      label: 'Think',
      durationClass: 'chatagent-duration text-base-content/40',
    });
    var summary = step.summary;
    var previewEl = step.previewEl;
    var durationEl = step.durationEl;
    details.appendChild(summary);

    var body = document.createElement('div');
    body.className = 'chatagent-thinking-body';
    body.setAttribute('data-testid', 'chatagent-message-body');
    details.appendChild(body);
    container.appendChild(details);
    ns.scrollMessages(container);

    var startedAt = Date.now();
    var timer = setInterval(function () {
      durationEl.textContent = ns.formatDuration(Date.now() - startedAt);
    }, 100);

    return {
      body: body,
      preview: previewEl,
      stopTimer: function () {
        clearInterval(timer);
      },
      setDuration: function (ms) {
        clearInterval(timer);
        if (ms > 0) {
          durationEl.textContent = ns.formatDuration(ms);
        }
      },
      setPreview: function (text) {
        setStepPreview(previewEl, text);
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

      var details = document.createElement('details');
      details.className = 'chatagent-tool chatagent-step';
      details.setAttribute('data-testid', 'chatagent-tool-details');
      details.open = false;

      var step = createStepSummary({
        summaryClass:
          'chatagent-tool-summary chatagent-step-summary cursor-pointer select-none list-none',
        iconHTML: toolIconSVG,
        label: ev.name || 'tool',
        labelClass: 'chatagent-step-label font-mono',
        labelTestId: 'chatagent-tool-name',
      });
      var summary = step.summary;
      var previewEl = step.previewEl;
      var duration = step.durationEl;

      var body = document.createElement('div');
      body.className = 'chatagent-tool-body';

      var stdout = document.createElement('pre');
      stdout.className = 'chatagent-tool-output hidden';
      stdout.setAttribute('data-testid', 'chatagent-tool-stdout');

      var stderr = document.createElement('pre');
      stderr.className = 'chatagent-tool-output chatagent-tool-stderr hidden';
      stderr.setAttribute('data-testid', 'chatagent-tool-stderr');

      body.appendChild(stdout);
      body.appendChild(stderr);
      details.appendChild(summary);
      details.appendChild(body);
      wrap.appendChild(details);
      insertStreamNode(container, wrap, anchorBody);

      card = {
        wrap: wrap,
        details: details,
        status: '',
        duration: duration,
        preview: previewEl,
        stdout: stdout,
        stderr: stderr,
        startedAt: Date.now(),
        timer: setInterval(function () {
          if (card.status === 'running' && card.startedAt) {
            card.duration.textContent = ns.formatDuration(
              Date.now() - card.startedAt,
            );
          }
        }, 100),
      };
      setToolStatus(card, ev.status || 'running');
      cards[key] = card;
    }

    if (ev.status) {
      setToolStatus(card, ev.status);
    }
    if (ev.duration_ms > 0) {
      if (card.timer) {
        clearInterval(card.timer);
        card.timer = null;
      }
      card.duration.textContent = ns.formatDuration(ev.duration_ms);
    } else if (card.status === 'running' && card.startedAt) {
      card.duration.textContent = ns.formatDuration(
        Date.now() - card.startedAt,
      );
    } else if (
      card.status === 'completed' ||
      card.status === 'error' ||
      card.status === 'needs_approval'
    ) {
      if (card.timer) {
        clearInterval(card.timer);
        card.timer = null;
      }
      if (!card.duration.textContent && card.startedAt) {
        card.duration.textContent = ns.formatDuration(
          Date.now() - card.startedAt,
        );
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
    var preview = chatAgentToolPreview(ev.text, card.stdout.textContent);
    setStepPreview(card.preview, preview);
    if (card.details && ns.toolCardShouldExpand(card.status)) {
      card.details.open = true;
    }
    ns.scrollMessages(container);
    return card;
  }

  function expandRunningToolCards(toolCards) {
    Object.keys(toolCards || {}).forEach(function (key) {
      var card = toolCards[key];
      if (!card || !card.details) {
        return;
      }
      if (card.status === 'running') {
        setToolStatus(card, 'needs_approval');
        card.details.open = true;
      }
    });
  }

  function insertStreamNode(container, node, anchorBody) {
    var anchor = anchorBody ? anchorBody.parentElement : null;
    if (anchor && anchor.parentElement === container) {
      container.insertBefore(node, anchor);
      return;
    }
    container.appendChild(node);
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
    var meta = ensureMessageMeta(bodyEl);
    if (!meta) {
      return;
    }
    var existing = meta.querySelector(
      '[data-testid="chatagent-message-duration"]',
    );
    if (existing) {
      existing.remove();
    }
    var footer = document.createElement('div');
    footer.className = 'chatagent-duration text-xs text-base-content/50';
    footer.setAttribute('data-testid', 'chatagent-message-duration');
    var parts = [];
    if (turnMs > 0) {
      parts.push('Turn ' + ns.formatDuration(turnMs));
    }
    if (runMs > 0) {
      parts.push('Total ' + ns.formatDuration(runMs));
    }
    footer.textContent = parts.join(' · ');
    var copyBtn = meta.querySelector('[data-testid="chatagent-copy-md"]');
    if (copyBtn) {
      meta.insertBefore(footer, copyBtn);
    } else {
      meta.appendChild(footer);
    }
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

  function showReloadMessagesPrompt(el, message) {
    if (!el) {
      return;
    }
    if (isApprovalStatusMessage(message)) {
      return;
    }
    el.classList.remove('hidden');
    el.textContent = '';
    el.appendChild(document.createTextNode(message + ' '));
    var btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'btn btn-ghost btn-xs text-primary align-baseline';
    btn.textContent = 'Reload messages';
    btn.setAttribute('data-testid', 'chatagent-reload-messages');
    btn.addEventListener('click', function () {
      window.location.reload();
    });
    el.appendChild(btn);
  }

  ns.streamMessage = function (
    messagesURL,
    text,
    threadRoot,
    onDone,
    approval,
    attachments,
  ) {
    var messagesEl = threadRoot.querySelector('#chatagent-messages');
    var errorEl = threadRoot.querySelector('#chatagent-thread-error');
    var cancelURL = threadRoot.getAttribute('data-cancel-url') || '';
    var mediaURL = threadRoot.getAttribute('data-media-url') || '';
    var pending = Array.isArray(attachments) ? attachments.slice() : [];
    var assistantBody = null;
    var assistantText = '';
    var thinkingState = null;
    var thinkingText = '';
    var toolCards = {};
    var lastTurnDurationMs = 0;
    var lastRunDurationMs = 0;
    var sawDone = false;
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
        renderedClass: assistantBodyClass,
        plainClass: assistantPlainClass,
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
    appendUserMessage(messagesEl, text, pending, mediaURL);

    flowbotCSRFHeadersAsync({
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    })
      .then(function (headers) {
        var upload = Promise.resolve(pending);
        if (pending.length && mediaURL) {
          upload = Promise.all(
            pending.map(function (item) {
              if (item.file_id) {
                return {
                  file_id: item.file_id,
                  mime_type: item.mime_type,
                  kind: item.kind,
                };
              }
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
        return upload.then(function (refs) {
          return fetch(messagesURL, {
            method: 'POST',
            headers: headers,
            body: JSON.stringify({ text: text || '', attachments: refs || [] }),
          });
        });
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
          if (ns.handleTrajectoryStreamEvent) {
            ns.handleTrajectoryStreamEvent(ev, threadRoot);
          }
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
            if (thinkingState.setPreview) {
              thinkingState.setPreview(thinkingText);
            }
            thinkingRenderer.update(thinkingText);
            return;
          }
          if (ev.type === 'tool') {
            var card = upsertToolCard(messagesEl, ev, toolCards, assistantBody);
            if (ns.handleTodoToolEvent) {
              ns.handleTodoToolEvent(ev, card, threadRoot);
            }
            return;
          }
          if (ev.type === 'turn') {
            if (ev.duration_ms > 0) {
              lastTurnDurationMs = ev.duration_ms;
              syncAssistantDuration();
            }
            return;
          }
          if (ev.type === 'delta') {
            var chunk = ev.text || '';
            if (ns.isToolPayloadText(chunk) || ns.isRunningToolStatus(chunk)) {
              return;
            }
            if (!assistantBody) {
              assistantBody = appendAssistantMessage(messagesEl, '', true);
              syncAssistantDuration();
            }
            assistantText += chunk;
            mdRenderer.update(assistantText);
            return;
          }
          if (ev.type === 'done') {
            sawDone = true;
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
            if (ns.refreshTodosFromServer) {
              ns.refreshTodosFromServer(threadRoot);
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
            if (ev.type === 'confirm') {
              expandRunningToolCards(toolCards);
            }
            approval.handleStreamEvent(ev);
            return;
          }
          if (ev.type === 'error') {
            sawDone = true;
            showThreadError(errorEl, ev.message || 'Run failed');
          } else if (ev.type === 'canceled') {
            sawDone = true;
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
            ensureCopyMarkdownButton(assistantBody, assistantText);
          }
          if (ctxCtrl) {
            ctxCtrl.onRunComplete();
          }
          if (typeof onDone === 'function') {
            onDone();
          }
          // Stream ended cleanly but never delivered Done (e.g. mid-turn SSE
          // detach while waiting for tool approval). Reload persisted history.
          if (!sawDone) {
            showReloadMessagesPrompt(
              errorEl,
              'Run finished without a live stream.',
            );
          }
        });
      })
      .catch(function (err) {
        var msg = (err && err.message) || 'Request failed';
        // Incomplete chunked SSE (e.g. server write timeout) often surfaces as a
        // TypeError/network error after the turn already persisted server-side.
        var networkLost =
          !sawDone &&
          (err.name === 'TypeError' ||
            /network|fetch|load failed|incomplete/i.test(msg));
        if (networkLost) {
          showReloadMessagesPrompt(errorEl, 'Connection lost while streaming.');
          return;
        }
        showThreadError(errorEl, msg);
      })
      .finally(function () {
        ns.setRunning(false, threadRoot);
      });

    if (cancelURL) {
      var cancelBtn = threadRoot.querySelector('#chatagent-cancel-run');
      if (cancelBtn) {
        cancelBtn.addEventListener('click', function () {
          flowbotCSRFHeadersAsync().then(function (headers) {
            fetch(cancelURL, { method: 'POST', headers: headers }).catch(
              function () {},
            );
          });
        });
      }
    }
  };
})();
