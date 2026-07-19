// Apply stored DaisyUI theme before paint (no inline script).
(function () {
  try {
    var t = localStorage.getItem('flowbot-theme');
    if (t) {
      document.documentElement.setAttribute('data-theme', t);
    }
  } catch {
    /* ignore storage access failures */
  }
})();
