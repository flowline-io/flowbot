(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var NEAR_BOTTOM_PX = 96;
  var scrollStateByContainer = new WeakMap();

  ns.showError = function (el, message) {
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
  };

  ns.setRunning = function (running, threadRoot) {
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
  };

  ns.isThreadRunning = function (threadRoot) {
    var cancelBtn = threadRoot
      ? threadRoot.querySelector('#chatagent-cancel-run')
      : null;
    return !!(cancelBtn && !cancelBtn.classList.contains('hidden'));
  };

  ns.isNearBottom = function (container) {
    if (!container) {
      return true;
    }
    return (
      container.scrollHeight - container.scrollTop - container.clientHeight <=
      NEAR_BOTTOM_PX
    );
  };

  function getScrollState(container) {
    return container ? scrollStateByContainer.get(container) : null;
  }

  function jumpButtonFor(container) {
    var state = getScrollState(container);
    if (state && state.jumpBtn) {
      return state.jumpBtn;
    }
    var wrap = container && container.parentElement;
    return wrap ? wrap.querySelector('#chatagent-jump-bottom') : null;
  }

  ns.updateJumpBottomButton = function (container) {
    var btn = jumpButtonFor(container);
    if (!btn || !container) {
      return;
    }
    var state = getScrollState(container);
    var stick = state ? state.stickToBottom : ns.isNearBottom(container);
    btn.classList.toggle('hidden', stick);
  };

  ns.stickMessagesToBottom = function (container) {
    if (!container) {
      return;
    }
    var state = getScrollState(container);
    if (!state) {
      state = { stickToBottom: true };
      scrollStateByContainer.set(container, state);
    } else {
      state.stickToBottom = true;
    }
    container.scrollTop = container.scrollHeight;
    ns.updateJumpBottomButton(container);
  };

  ns.scrollMessages = function (container) {
    if (!container) {
      return;
    }
    var state = getScrollState(container);
    if (!state) {
      container.scrollTop = container.scrollHeight;
      return;
    }
    if (state.stickToBottom) {
      container.scrollTop = container.scrollHeight;
    }
    ns.updateJumpBottomButton(container);
  };

  ns.initMessageScroll = function (container, jumpBtn) {
    if (!container) {
      return;
    }
    var state = {
      stickToBottom: true,
      jumpBtn: jumpBtn || null,
    };
    scrollStateByContainer.set(container, state);
    container.addEventListener(
      'scroll',
      function () {
        state.stickToBottom = ns.isNearBottom(container);
        ns.updateJumpBottomButton(container);
      },
      { passive: true },
    );
    if (jumpBtn) {
      jumpBtn.addEventListener('click', function () {
        ns.stickMessagesToBottom(container);
      });
    }
    ns.stickMessagesToBottom(container);
  };

  ns.formatDuration = function (ms) {
    if (!ms || ms <= 0) {
      return '';
    }
    if (ms < 1000) {
      return ms + 'ms';
    }
    return (ms / 1000).toFixed(1) + 's';
  };

  ns.isToolPayloadText = function (text) {
    var trimmed = (text || '').trim();
    if (!trimmed) {
      return false;
    }
    return (
      trimmed.indexOf('[{"id":') === 0 ||
      trimmed.indexOf('[{"id"') === 0 ||
      trimmed.indexOf('{"id":"call_') === 0
    );
  };

  ns.isRunningToolStatus = function (text) {
    var trimmed = (text || '').trim();
    return (
      trimmed.indexOf('Running tool:') === 0 ||
      trimmed.indexOf('Delegating to subagent:') === 0
    );
  };

  ns.toolCardShouldExpand = function (status) {
    var s = String(status || '')
      .trim()
      .toLowerCase();
    return s === 'error' || s === 'failed' || s === 'needs_approval';
  };
})();
