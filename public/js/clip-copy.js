(function () {
  'use strict';

  function copyText(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise(function (resolve, reject) {
      var area = document.createElement('textarea');
      area.value = text;
      area.setAttribute('readonly', '');
      area.style.position = 'fixed';
      area.style.left = '-9999px';
      document.body.appendChild(area);
      area.select();
      try {
        if (!document.execCommand('copy')) {
          reject(new Error('copy failed'));
        } else {
          resolve();
        }
      } catch (err) {
        reject(err);
      } finally {
        document.body.removeChild(area);
      }
    });
  }

  document.addEventListener('click', function (event) {
    var btn = event.target.closest('[data-clip-copy]');
    if (!btn) {
      return;
    }
    var md = btn.getAttribute('data-clip-markdown') || '';
    copyText(md)
      .then(function () {
        btn.setAttribute('data-copied', 'true');
        var prev = btn.textContent;
        btn.textContent = 'Copied';
        window.setTimeout(function () {
          btn.removeAttribute('data-copied');
          btn.textContent = prev;
        }, 1500);
      })
      .catch(function () {
        btn.textContent = 'Copy failed';
      });
  });
})();
