(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

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
  ns.createStreamingMarkdownRenderer = function(threadRoot, getBodyEl, options) {
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
        ns.scrollMessages(messagesEl);
      }
    }

    function applyHTML(bodyEl, html) {
      bodyEl.className = renderedClass;
      bodyEl.innerHTML = html;
      bodyEl.dataset.mdRendered = '1';
      if (options.onAfterRender) {
        options.onAfterRender(bodyEl);
      }
      scroll();
    }

    function showPlainText(bodyEl, text) {
      bodyEl.className = plainClass;
      delete bodyEl.dataset.mdRendered;
      bodyEl.textContent = text;
      if (options.onAfterRender) {
        options.onAfterRender(bodyEl);
      }
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
})();
