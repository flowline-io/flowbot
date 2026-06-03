Alpine.data('eventFilters', () => ({
  timeRange: 'custom',
  timeStart: '',
  timeEnd: '',
  search: '',
  pipeline: '',
  source: '',
  eventType: '',
  tab: 'data-events',

  init() {
    const params = new URLSearchParams(window.location.search);
    this.timeStart = params.get('time_start') || '';
    this.timeEnd = params.get('time_end') || '';
    this.search = params.get('search') || '';
    this.pipeline = params.get('pipeline') || '';
    this.source = params.get('source') || '';
    this.eventType = params.get('type') || '';
    this.tab = params.get('tab') || 'data-events';

    if (this.timeStart && this.timeEnd) {
      const now = Date.now();
      const start = new Date(this.timeStart).getTime();
      const end = new Date(this.timeEnd).getTime();
      if (Math.abs(now - end) < 5000) {
        if (Math.abs(now - start - 3600000) < 5000) {
          this.timeRange = '1h';
        } else if (Math.abs(now - start - 86400000) < 10000) {
          this.timeRange = '24h';
        } else if (Math.abs(now - start - 604800000) < 30000) {
          this.timeRange = '7d';
        }
      }
    }
  },

  setTimeRange(range) {
    const now = new Date();
    const durations = { '1h': 3600000, '24h': 86400000, '7d': 604800000 };
    this.timeRange = range;
    if (durations[range]) {
      this.timeEnd = now.toISOString().slice(0, 16);
      this.timeStart = new Date(now - durations[range]).toISOString().slice(0, 16);
    }
    this.submitFilter();
  },

  onDateChange() {
    this.timeRange = 'custom';
    this.submitFilter();
  },

  getFilterParams() {
    const params = new URLSearchParams();
    params.set('tab', this.tab);
    if (this.search) params.set('search', this.search);
    if (this.pipeline) params.set('pipeline', this.pipeline);
    if (this.source) params.set('source', this.source);
    if (this.eventType) params.set('type', this.eventType);
    if (this.timeStart) params.set('time_start', this.timeStart + ':00Z');
    if (this.timeEnd) params.set('time_end', this.timeEnd + ':00Z');
    params.set('page', '1');
    return params.toString();
  },

  submitFilter() {
    const url = '/service/web/events/filtered-events?' + this.getFilterParams();
    htmx.ajax('GET', url, { target: '#events-table-container', swap: 'innerHTML' });
  },

  switchTab(newTab) {
    this.tab = newTab;
    this.submitFilter();
  },

  debounceSearch() {
    clearTimeout(this._searchTimer);
    this._searchTimer = setTimeout(() => this.submitFilter(), 300);
  }
}));
