(function () {
  'use strict';

  function renderEnrollQR(form) {
    if (!form || typeof QRCode === 'undefined') {
      return;
    }
    var mount = form.querySelector('[data-testid="enroll-2fa-qr"]');
    if (!mount) {
      return;
    }
    var uri = mount.getAttribute('data-otpauth-uri');
    if (!uri) {
      return;
    }
    mount.innerHTML = '';
    // eslint-disable-next-line no-new
    new QRCode(mount, {
      text: uri,
      width: 216,
      height: 216,
      colorDark: '#000000',
      colorLight: '#ffffff',
      correctLevel: QRCode.CorrectLevel.M,
    });
  }

  function initEnrollQR(root) {
    var scope = root || document;
    var form = scope.querySelector('[data-testid="enroll-2fa-form"]');
    if (form) {
      renderEnrollQR(form);
    }
  }

  document.addEventListener('DOMContentLoaded', function () {
    initEnrollQR(document);
  });

  document.body.addEventListener('htmx:afterSwap', function (evt) {
    var el = evt.detail && evt.detail.elt;
    if (!el) {
      return;
    }
    if (el.getAttribute('data-testid') === 'enroll-2fa-form') {
      renderEnrollQR(el);
      return;
    }
    initEnrollQR(el);
  });
})();
