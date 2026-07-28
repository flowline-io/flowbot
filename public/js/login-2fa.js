(function () {
  'use strict';

  var submitting = false;

  function findForm(el) {
    return el && el.closest
      ? el.closest('[data-testid="login-2fa-form"]')
      : null;
  }

  function digits(form) {
    return Array.prototype.slice.call(
      form.querySelectorAll('[data-login-2fa-digit]'),
    );
  }

  function valueInput(form) {
    return form.querySelector('[data-login-2fa-value]');
  }

  function syncValue(form) {
    var hidden = valueInput(form);
    if (!hidden) {
      return '';
    }
    var backupPanel = form.querySelector('[data-login-2fa-backup-panel]');
    if (backupPanel && !backupPanel.hidden) {
      var backup = form.querySelector('[data-login-2fa-backup]');
      hidden.value = backup ? String(backup.value || '').trim() : '';
      return hidden.value;
    }
    var code = digits(form)
      .map(function (input) {
        return String(input.value || '')
          .replace(/\D/g, '')
          .slice(0, 1);
      })
      .join('');
    hidden.value = code;
    return code;
  }

  function focusDigit(form, index) {
    var list = digits(form);
    if (!list.length) {
      return;
    }
    var i = Math.max(0, Math.min(index, list.length - 1));
    list[i].focus();
    list[i].select();
  }

  function setMode(form, mode) {
    var otp = form.querySelector('[data-login-2fa-otp-panel]');
    var backup = form.querySelector('[data-login-2fa-backup-panel]');
    if (!otp || !backup) {
      return;
    }
    var useBackup = mode === 'backup';
    otp.hidden = useBackup;
    otp.classList.toggle('hidden', useBackup);
    backup.hidden = !useBackup;
    backup.classList.toggle('hidden', !useBackup);
    if (useBackup) {
      digits(form).forEach(function (input) {
        input.value = '';
      });
      var backupInput = form.querySelector('[data-login-2fa-backup]');
      if (backupInput) {
        backupInput.focus();
      }
    } else {
      var backupInputClear = form.querySelector('[data-login-2fa-backup]');
      if (backupInputClear) {
        backupInputClear.value = '';
      }
      focusDigit(form, 0);
    }
    syncValue(form);
  }

  function tryAutoSubmit(form) {
    if (submitting) {
      return;
    }
    var code = syncValue(form);
    if (code.length !== 6 || !/^\d{6}$/.test(code)) {
      return;
    }
    submitting = true;
    if (typeof form.requestSubmit === 'function') {
      form.requestSubmit();
    } else {
      form.dispatchEvent(
        new Event('submit', { bubbles: true, cancelable: true }),
      );
    }
  }

  function applyDigits(form, text, startIndex) {
    var list = digits(form);
    var nums = String(text || '').replace(/\D/g, '');
    if (!nums || !list.length) {
      return;
    }
    var i = typeof startIndex === 'number' ? startIndex : 0;
    var n = 0;
    while (n < nums.length && i < list.length) {
      list[i].value = nums.charAt(n);
      i += 1;
      n += 1;
    }
    syncValue(form);
    if (i >= list.length) {
      focusDigit(form, list.length - 1);
      tryAutoSubmit(form);
      return;
    }
    focusDigit(form, i);
  }

  function onDigitInput(input) {
    var form = findForm(input);
    if (!form) {
      return;
    }
    var list = digits(form);
    var idx = list.indexOf(input);
    var cleaned = String(input.value || '').replace(/\D/g, '');
    if (cleaned.length > 1) {
      applyDigits(form, cleaned, idx >= 0 ? idx : 0);
      return;
    }
    input.value = cleaned.slice(0, 1);
    syncValue(form);
    if (input.value) {
      if (idx >= 0 && idx < list.length - 1) {
        focusDigit(form, idx + 1);
      }
      tryAutoSubmit(form);
    }
  }

  document.addEventListener('input', function (evt) {
    var target = evt.target;
    if (!target) {
      return;
    }
    if (target.hasAttribute && target.hasAttribute('data-login-2fa-digit')) {
      onDigitInput(target);
      return;
    }
    if (target.hasAttribute && target.hasAttribute('data-login-2fa-backup')) {
      var form = findForm(target);
      if (form) {
        syncValue(form);
      }
    }
  });

  document.addEventListener('keydown', function (evt) {
    var target = evt.target;
    if (
      !target ||
      !target.hasAttribute ||
      !target.hasAttribute('data-login-2fa-digit')
    ) {
      return;
    }
    var form = findForm(target);
    if (!form) {
      return;
    }
    var list = digits(form);
    var idx = list.indexOf(target);
    if (idx < 0) {
      return;
    }
    if (evt.key === 'Backspace') {
      if (target.value) {
        target.value = '';
        syncValue(form);
        return;
      }
      if (idx > 0) {
        evt.preventDefault();
        list[idx - 1].value = '';
        syncValue(form);
        focusDigit(form, idx - 1);
      }
      return;
    }
    if (evt.key === 'ArrowLeft' && idx > 0) {
      evt.preventDefault();
      focusDigit(form, idx - 1);
      return;
    }
    if (evt.key === 'ArrowRight' && idx < list.length - 1) {
      evt.preventDefault();
      focusDigit(form, idx + 1);
    }
  });

  document.addEventListener('paste', function (evt) {
    var target = evt.target;
    if (
      !target ||
      !target.hasAttribute ||
      !target.hasAttribute('data-login-2fa-digit')
    ) {
      return;
    }
    var form = findForm(target);
    if (!form || !evt.clipboardData) {
      return;
    }
    var text = evt.clipboardData.getData('text') || '';
    if (!/\d/.test(text)) {
      return;
    }
    evt.preventDefault();
    var list = digits(form);
    var idx = list.indexOf(target);
    applyDigits(form, text, idx >= 0 ? idx : 0);
  });

  document.addEventListener('click', function (evt) {
    var showBackup =
      evt.target &&
      evt.target.closest &&
      evt.target.closest('[data-login-2fa-show-backup]');
    if (showBackup) {
      var form = findForm(showBackup);
      if (form) {
        setMode(form, 'backup');
      }
      return;
    }
    var showOtp =
      evt.target &&
      evt.target.closest &&
      evt.target.closest('[data-login-2fa-show-otp]');
    if (showOtp) {
      var otpForm = findForm(showOtp);
      if (otpForm) {
        setMode(otpForm, 'otp');
      }
    }
  });

  document.addEventListener('submit', function (evt) {
    var form = findForm(evt.target);
    if (!form) {
      return;
    }
    syncValue(form);
    submitting = true;
  });

  function focusFirst(form) {
    if (!form) {
      return;
    }
    submitting = false;
    var backupPanel = form.querySelector('[data-login-2fa-backup-panel]');
    if (backupPanel && !backupPanel.hidden) {
      var backup = form.querySelector('[data-login-2fa-backup]');
      if (backup) {
        backup.focus();
      }
      return;
    }
    focusDigit(form, 0);
  }

  function init(root) {
    var scope = root || document;
    var form = scope.querySelector
      ? scope.querySelector('[data-testid="login-2fa-form"]')
      : null;
    if (
      !form &&
      scope.getAttribute &&
      scope.getAttribute('data-testid') === 'login-2fa-form'
    ) {
      form = scope;
    }
    focusFirst(form);
  }

  document.addEventListener('DOMContentLoaded', function () {
    init(document);
  });

  document.body.addEventListener('htmx:afterSwap', function (evt) {
    var el = evt.detail && evt.detail.elt;
    if (!el) {
      return;
    }
    if (
      el.getAttribute &&
      el.getAttribute('data-testid') === 'login-2fa-form'
    ) {
      init(el);
      return;
    }
    init(el);
  });
})();
