(function () {
  var panel = document.getElementById('log-panel');
  if (!panel) {
    return;
  }
  var url = panel.getAttribute('data-url');
  if (!url) {
    return;
  }

  var es = null;

  function appendLine(text) {
    panel.appendChild(document.createTextNode(text + '\n'));
    panel.scrollTop = panel.scrollHeight;
  }

  function connect() {
    if (es) {
      es.close();
      es = null;
    }
    es = new EventSource(url);
    es.addEventListener('open', function () {
      if (typeof showToast === 'function') {
        showToast('Log stream connected', 'info');
      }
    });
    es.addEventListener('message', function (e) {
      appendLine(e.data);
    });
    es.addEventListener('error', function () {
      if (!es || es.readyState !== EventSource.CLOSED) {
        return;
      }
      es.close();
      es = null;
      appendLine('-- Log stream ended --');
      if (typeof showToast === 'function') {
        showToast(
          'Log stream ended. Refresh the page to reconnect.',
          'warning',
        );
      }
    });
  }

  panel.textContent = '';
  connect();
})();
