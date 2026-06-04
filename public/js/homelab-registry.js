document.addEventListener('alpine:init', function () {
  Alpine.data('homelabRegistry', () => ({
    search: '',
    filterCapability: '',

    appMatches(el) {
      const name = el.getAttribute('data-app-name') || '';
      const caps = el.getAttribute('data-app-caps') || '';
      const searchMatch =
        !this.search || name.toLowerCase().includes(this.search.toLowerCase());
      const capMatch =
        !this.filterCapability ||
        caps.split(',').includes(this.filterCapability);
      return searchMatch && capMatch;
    },
  }));
});
