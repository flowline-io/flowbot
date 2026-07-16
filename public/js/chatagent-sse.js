(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  ns.flushSSEBuffer = function(buffer, onEvent) {
    if (!buffer) {
      return '';
    }
    return ns.parseSSEChunk(buffer, onEvent);
  }

  ns.parseSSEChunk = function(buffer, onEvent) {
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
})();
