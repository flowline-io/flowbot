// Unified confirmation modal controller.
// Works with HTMX (data-confirm attribute), Alpine.js, and vanilla JS.
(function () {
  var modal, titleEl, bodyEl, confirmBtn;
  var currentCallback = null;
  var currentCancelCallback = null;
  var inFlightElements = new WeakSet();

  function getModal() {
    if (!modal) {
      modal = document.getElementById('confirm-modal');
      titleEl = document.getElementById('confirm-modal-title');
      bodyEl = document.getElementById('confirm-modal-body');
      confirmBtn = document.getElementById('confirm-modal-confirm');
    }
    return modal;
  }

  function setupButtons() {
    var cancelEl = document.getElementById('confirm-modal-cancel');
    var confirmEl = document.getElementById('confirm-modal-confirm');
    if (cancelEl) {
      cancelEl.addEventListener('click', function () { closeModal(false); });
    }
    if (confirmEl) {
      confirmEl.addEventListener('click', function () { closeModal(true); });
    }
  }

  function openModal(title, message, confirmText, confirmClass, onConfirm, onCancel) {
    var m = getModal();
    if (!m) return;
    titleEl.textContent = title;
    bodyEl.textContent = message;
    confirmBtn.textContent = confirmText || 'Confirm';
    confirmBtn.className = 'btn ' + (confirmClass || 'btn-error');
    currentCallback = onConfirm;
    currentCancelCallback = onCancel;
    setupButtons();
    m.showModal();
  }

  function closeModal(confirmed) {
    var m = getModal();
    if (!m) return;
    m.close();
    if (confirmed && currentCallback) {
      currentCallback();
    } else if (!confirmed && currentCancelCallback) {
      currentCancelCallback();
    }
    currentCallback = null;
    currentCancelCallback = null;
  }

  // Expose global API for programmatic use (pipeline-editor.js, Alpine, etc.)
  window.showConfirmModal = function (opts) {
    openModal(
      opts.title || 'Confirm',
      opts.message || '',
      opts.confirmText,
      opts.confirmClass,
      opts.onConfirm,
      opts.onCancel
    );
  };

  // HTMX integration: intercept clicks on elements with data-confirm attribute.
  // Uses capture phase to intercept before HTMX processes the click.
  document.addEventListener(
    'click',
    function (e) {
      var el = e.target.closest('[data-confirm]');
      if (!el) return;
      if (inFlightElements.has(el)) return;

      e.preventDefault();
      e.stopPropagation();

      var msg = el.getAttribute('data-confirm');
      var title = el.getAttribute('data-confirm-title') || 'Confirm Action';
      var btn = el.getAttribute('data-confirm-btn') || 'Confirm';
      var cls = el.getAttribute('data-confirm-class') || 'btn-error';

      openModal(title, msg, btn, cls, function () {
        inFlightElements.add(el);
        el.click();
        setTimeout(function () {
          inFlightElements.delete(el);
        }, 200);
      });
    },
    true
  );
})();
