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
    }));
  }

  if (window.Alpine) {
    register();
  } else {
    document.addEventListener('alpine:init', register);
  }
})();
