(function () {
  var panel = document.getElementById('log-panel');
  if (!panel) {
    return;
  }
  var url = panel.getAttribute('data-url');
  if (!url) {
    return;
  }
  var es = new EventSource(url);
  panel.textContent = '';
  es.addEventListener('message', function (e) {
    panel.appendChild(document.createTextNode(e.data + '\n'));
    panel.scrollTop = panel.scrollHeight;
  });
  es.addEventListener('error', function () {
    if (es.readyState === EventSource.CLOSED) {
      panel.appendChild(document.createTextNode('\n-- Log stream ended --'));
      es.close();
    }
  });
})();
