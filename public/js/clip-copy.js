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

  function showCopyFeedback(btn, message, restore) {
    btn.setAttribute('data-copied', 'true');
    if (btn.querySelector('svg')) {
      var prevTitle = btn.getAttribute('title') || '';
      var prevAria = btn.getAttribute('aria-label') || '';
      btn.setAttribute('title', message);
      btn.setAttribute('aria-label', message);
      if (!restore) {
        return;
      }
      window.setTimeout(function () {
        btn.removeAttribute('data-copied');
        btn.setAttribute('title', prevTitle);
        btn.setAttribute('aria-label', prevAria);
      }, 1500);
      return;
    }
    var prev = btn.textContent;
    btn.textContent = message;
    if (!restore) {
      return;
    }
    window.setTimeout(function () {
      btn.removeAttribute('data-copied');
      btn.textContent = prev;
    }, 1500);
  }

  document.addEventListener('click', function (event) {
    var btn = event.target.closest('[data-clip-copy]');
    if (!btn) {
      return;
    }
    var md = btn.getAttribute('data-clip-markdown') || '';
    copyText(md)
      .then(function () {
        showCopyFeedback(btn, 'Copied', true);
      })
      .catch(function () {
        showCopyFeedback(btn, 'Copy failed', true);
      });
  });
})();
