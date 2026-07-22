'use strict';

(function () {
  function pickScrollIndex(steps, selectedIndex) {
    if (!steps || !steps.length) return -1;
    var running = -1;
    for (var i = 0; i < steps.length; i++) {
      if (steps[i] && steps[i].status === 'running') {
        running = i;
        break;
      }
    }
    if (running >= 0) return running;
    if (selectedIndex >= 0 && selectedIndex < steps.length)
      return selectedIndex;
    return steps.length - 1;
  }

  function shouldFlash(prevStatus, nextStatus) {
    if (!nextStatus) return false;
    return prevStatus !== nextStatus;
  }

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
        flashIndex: -1,
        flashTimer: null,

        init: function () {
          this.recalc();

          var idx = this.steps.findIndex(function (s) {
            return s.status === 'running' || s.status === 'pending';
          });
          this.selectedIndex = idx >= 0 ? idx : this.steps.length - 1;
          this.$nextTick(
            function () {
              this.scrollToStep(
                pickScrollIndex(this.steps, this.selectedIndex),
              );
            }.bind(this),
          );

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

        flashStep: function (idx) {
          var self = this;
          this.flashIndex = idx;
          if (this.flashTimer) {
            clearTimeout(this.flashTimer);
          }
          this.flashTimer = setTimeout(function () {
            if (self.flashIndex === idx) {
              self.flashIndex = -1;
            }
          }, 900);
        },

        scrollToStep: function (idx) {
          if (idx < 0) return;
          var root = this.$el;
          if (!root || !root.querySelector) return;
          var row = root.querySelector('[data-step-index="' + idx + '"]');
          if (row && typeof row.scrollIntoView === 'function') {
            row.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
          }
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
          var prevStatus = step.status;
          if (shouldFlash(prevStatus, evt.status)) {
            this.flashStep(evt.step_index);
          }
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
            var self = this;
            this.$nextTick(function () {
              self.scrollToStep(evt.step_index);
            });
          }
          this.recalc();
        },

        selectStep: function (idx) {
          this.selectedIndex = idx;
        },

        stepRowClass: function (idx) {
          var classes = '';
          if (this.selectedIndex === idx) {
            classes += 'bg-base-300 ';
          }
          if (this.flashIndex === idx) {
            classes += 'run-step-flash ';
          }
          return classes.trim();
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

  // Expose pure helpers for unit tests when running under Node.
  if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
      pickScrollIndex: pickScrollIndex,
      shouldFlash: shouldFlash,
    };
  } else {
    window.FlowbotPipelineRunLive = {
      pickScrollIndex: pickScrollIndex,
      shouldFlash: shouldFlash,
    };
  }

  if (typeof Alpine !== 'undefined') {
    register();
  } else if (typeof document !== 'undefined') {
    document.addEventListener('alpine:init', register);
  }
})();
