(function () {
  function register() {
    Alpine.data('homelabRegistry', () => ({
      search: '',
      filterCapability: '',

      appMatches(el) {
        const appName = el.getAttribute('data-app-name') || '';
        const caps = el.getAttribute('data-app-caps') || '';
        const searchMatch =
          !this.search ||
          appName.toLowerCase().includes(this.search.toLowerCase());
        const capMatch =
          !this.filterCapability ||
          caps.split(',').includes(this.filterCapability);
        return searchMatch && capMatch;
      },
    }));
  }

  if (window.Alpine) {
    register();
  } else {
    document.addEventListener('alpine:init', register);
  }
})();
