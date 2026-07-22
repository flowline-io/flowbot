(function () {
  var STORAGE_KEY = 'flowbot-event-filters';

  function readPresets() {
    try {
      var raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) {
        return [];
      }
      var parsed = JSON.parse(raw);
      if (!Array.isArray(parsed)) {
        return [];
      }
      return parsed.filter(function (p) {
        return p && typeof p.name === 'string' && p.name.trim() !== '';
      });
    } catch {
      return [];
    }
  }

  function writePresets(presets) {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(presets));
    } catch {
      // intentionally silent: private mode / quota
    }
  }

  function register() {
    Alpine.data('eventFilters', () => ({
      timeRange: 'custom',
      timeStart: '',
      timeEnd: '',
      search: '',
      pipeline: '',
      source: '',
      eventType: '',
      tab: 'data-events',
      savedPresets: [],
      selectedPreset: '',

      init() {
        const params = new URLSearchParams(window.location.search);
        this.timeStart = params.get('time_start') || '';
        this.timeEnd = params.get('time_end') || '';
        this.search = params.get('search') || '';
        this.pipeline = params.get('pipeline') || '';
        this.source = params.get('source') || '';
        this.eventType = params.get('type') || '';
        this.tab = params.get('tab') || 'data-events';
        this.savedPresets = readPresets();
        this.$nextTick(() => {
          this.syncPresetOptions();
          this.submitFilter();
        });

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
          this.timeStart = new Date(now - durations[range])
            .toISOString()
            .slice(0, 16);
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

      syncURL() {
        const qs = this.getFilterParams();
        const next = '/service/web/events?' + qs;
        if (window.location.pathname + window.location.search !== next) {
          history.replaceState(null, '', next);
        }
      },

      submitFilter() {
        const qs = this.getFilterParams();
        this.syncURL();
        const url = '/service/web/events/filtered-events?' + qs;
        htmx.ajax('GET', url, {
          target: '#events-table-container',
          swap: 'innerHTML',
        });
      },

      switchTab(newTab) {
        this.tab = newTab;
        this.submitFilter();
      },

      debounceSearch() {
        clearTimeout(this.searchTimer);
        this.searchTimer = setTimeout(() => this.submitFilter(), 300);
      },

      resetFilters() {
        clearTimeout(this.searchTimer);
        this.timeRange = 'custom';
        this.timeStart = '';
        this.timeEnd = '';
        this.search = '';
        this.pipeline = '';
        this.source = '';
        this.eventType = '';
        this.selectedPreset = '';
        this.submitFilter();
      },

      syncPresetOptions() {
        var sel = this.$refs.savedFiltersSelect;
        if (!sel) {
          return;
        }
        var current = this.selectedPreset || '';
        while (sel.options.length > 1) {
          sel.remove(1);
        }
        for (var i = 0; i < this.savedPresets.length; i++) {
          var opt = document.createElement('option');
          opt.value = this.savedPresets[i].name;
          opt.textContent = this.savedPresets[i].name;
          sel.appendChild(opt);
        }
        sel.value = current;
        if (sel.value !== current) {
          this.selectedPreset = '';
        }
      },

      saveCurrentFilter() {
        var name = window.prompt('Filter name');
        if (!name) {
          return;
        }
        name = name.trim();
        if (!name) {
          return;
        }
        var preset = {
          name: name,
          source: this.source || '',
          type: this.eventType || '',
          pipeline: this.pipeline || '',
          search: this.search || '',
          timeRange: this.timeRange || 'custom',
          timeStart: this.timeStart || '',
          timeEnd: this.timeEnd || '',
          tab: this.tab || 'data-events',
        };
        var next = this.savedPresets.filter(function (p) {
          return p.name !== name;
        });
        next.push(preset);
        this.savedPresets = next;
        writePresets(next);
        this.selectedPreset = name;
        this.$nextTick(() => this.syncPresetOptions());
      },

      applySelectedPreset() {
        var name = this.selectedPreset;
        if (!name) {
          return;
        }
        var preset = null;
        for (var i = 0; i < this.savedPresets.length; i++) {
          if (this.savedPresets[i].name === name) {
            preset = this.savedPresets[i];
            break;
          }
        }
        if (!preset) {
          return;
        }
        this.source = preset.source || '';
        this.eventType = preset.type || '';
        this.pipeline = preset.pipeline || '';
        this.search = preset.search || '';
        this.tab = preset.tab || 'data-events';
        this.timeRange = preset.timeRange || 'custom';
        if (
          preset.timeRange === '1h' ||
          preset.timeRange === '24h' ||
          preset.timeRange === '7d'
        ) {
          this.setTimeRange(preset.timeRange);
          return;
        }
        this.timeStart = preset.timeStart || '';
        this.timeEnd = preset.timeEnd || '';
        this.submitFilter();
      },

      deleteSelectedPreset() {
        var name = this.selectedPreset;
        if (!name) {
          return;
        }
        var next = this.savedPresets.filter(function (p) {
          return p.name !== name;
        });
        this.savedPresets = next;
        writePresets(next);
        this.selectedPreset = '';
        this.$nextTick(() => this.syncPresetOptions());
      },
    }));
  }

  if (window.Alpine) {
    register();
  } else {
    document.addEventListener('alpine:init', register);
  }
})();
