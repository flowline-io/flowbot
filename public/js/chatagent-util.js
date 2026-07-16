(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

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

  ns.scrollMessages = function (container) {
    container.scrollTop = container.scrollHeight;
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
})();
