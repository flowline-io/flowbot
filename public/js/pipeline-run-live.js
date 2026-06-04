'use strict';

(function () {
  function register() {
    Alpine.data('pipelineRunLive', function () {
      var el = document.getElementById('initial-data');
      var initial = el
        ? JSON.parse(el.textContent)
        : { steps: [], runStatus: 'done' };

      return {
        runID: initial.runID,
        pipelineName: initial.pipelineName,
        trigger: initial.trigger,
        totalSteps: initial.totalSteps,
        steps: initial.steps || [],
        selectedIndex: -1,
        totalElapsed: 0,
        completed: 0,
        failedSteps: 0,
        runStatus: initial.runStatus || 'pending',
        eventSource: null,

        init: function () {
          this.recalc();

          var idx = this.steps.findIndex(function (s) {
            return s.status === 'running' || s.status === 'pending';
          });
          this.selectedIndex = idx >= 0 ? idx : this.steps.length - 1;

          if (this.runStatus === 'running') {
            var self = this;
            var watchURL = window.location.pathname.replace(
              /\/live$/,
              '/live/watch',
            );
            this.eventSource = new EventSource(watchURL);
            this.eventSource.addEventListener('message', function (e) {
              var evt = JSON.parse(e.data);
              self.applyEvent(evt);
            });
            this.eventSource.addEventListener('error', function () {
              if (self.runStatus === 'done' || self.runStatus === 'failed') {
                self.eventSource.close();
              }
            });
          }
        },

        recalc: function () {
          this.completed = this.steps.filter(function (s) {
            return s.status === 'done';
          }).length;
          this.failedSteps = this.steps.filter(function (s) {
            return s.status === 'error';
          }).length;
          this.totalElapsed = this.steps.reduce(function (acc, s) {
            return acc + (s.elapsed_ms || 0);
          }, 0);
        },

        applyEvent: function (evt) {
          if (evt.step_index === -1) {
            if (evt.status === 'start') this.runStatus = 'running';
            if (evt.status === 'complete') this.runStatus = 'done';
            if (evt.status === 'failed') this.runStatus = 'failed';
            if (evt.elapsed_ms) this.totalElapsed = evt.elapsed_ms;
            return;
          }
          var step = this.steps[evt.step_index];
          if (!step) return;
          step.status = evt.status;
          if (evt.status === 'done') {
            step.output = evt.output;
            step.elapsed_ms = evt.elapsed_ms;
          }
          if (evt.status === 'error') {
            step.error = evt.error;
            step.elapsed_ms = evt.elapsed_ms;
          }
          if (evt.status === 'running') {
            step.input = evt.input;
            this.selectedIndex = evt.step_index;
          }
          this.recalc();
        },

        selectStep: function (idx) {
          this.selectedIndex = idx;
        },

        get selectedStep() {
          return this.steps[this.selectedIndex] || null;
        },

        get formattedElapsed() {
          var ms = this.totalElapsed;
          if (ms < 1000) return ms + 'ms';
          return (ms / 1000).toFixed(1) + 's';
        },

        stepStatusIndicator: function (status) {
          return (
            {
              pending: 'text-base-content/30',
              running: 'text-info animate-pulse',
              done: 'text-success',
              error: 'text-error',
            }[status] || ''
          );
        },

        stepStatusIcon: function (status) {
          return (
            {
              pending: '\u25CB',
              running: '\u25C9',
              done: '\u2713',
              error: '\u2717',
            }[status] || '?'
          );
        },

        runStatusClass: function () {
          return (
            {
              pending: 'badge-ghost',
              running: 'badge-info',
              done: 'badge-success',
              failed: 'badge-error',
            }[this.runStatus] || 'badge-ghost'
          );
        },
      };
    });
  }

  if (window.Alpine) {
    register();
  } else {
    document.addEventListener('alpine:init', register);
  }
})();
