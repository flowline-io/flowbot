(function () {
  function register() {
    Alpine.data('tableFilter', () => ({
      search: '',

      clearSearch() {
        this.search = '';
      },

      rowMatches(el) {
        const text = el.getAttribute('data-filter-text') || '';
        if (!this.search) {
          return true;
        }
        return text.toLowerCase().includes(this.search.toLowerCase());
      },

      groupVisible(sectionEl) {
        if (!this.search) {
          return true;
        }
        const rows = sectionEl.querySelectorAll('[data-filter-text]');
        for (let i = 0; i < rows.length; i++) {
          if (this.rowMatches(rows[i])) {
            return true;
          }
        }
        return false;
      },
    }));
  }

  if (window.Alpine) {
    register();
  } else {
    document.addEventListener('alpine:init', register);
  }
})();
