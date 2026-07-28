(function () {
  'use strict';

  function parseJSONAttr(el, name) {
    var raw = el.getAttribute('data-' + name);
    if (!raw) {
      return null;
    }
    try {
      return JSON.parse(raw);
    } catch {
      return null;
    }
  }

  function radarScaleMax(values) {
    var peak = 0;
    for (var i = 0; i < values.length; i++) {
      var v = Number(values[i]);
      if (!isNaN(v) && v > peak) {
        peak = v;
      }
    }
    if (peak < 1) {
      peak = 1;
    }
    // Nice ceiling: at least 2 above peak, rounded up to even step.
    var padded = Math.ceil(peak + 1);
    if (padded % 2 !== 0) {
      padded += 1;
    }
    return Math.max(padded, 2);
  }

  function initLifeRadar(canvas) {
    if (!canvas || canvas.chartInstance || typeof Chart === 'undefined') {
      return;
    }
    var labels = parseJSONAttr(canvas, 'labels');
    var values = parseJSONAttr(canvas, 'values');
    if (
      !Array.isArray(labels) ||
      !Array.isArray(values) ||
      labels.length === 0
    ) {
      return;
    }
    var rootStyles = getComputedStyle(document.documentElement);
    var ink =
      rootStyles.getPropertyValue('--color-base-content').trim() || '#111827';
    var accent =
      rootStyles.getPropertyValue('--color-primary').trim() || '#0f766e';
    var maxR = radarScaleMax(values);
    var step = Math.max(1, Math.ceil(maxR / 5));
    canvas.chartInstance = new Chart(canvas, {
      type: 'radar',
      data: {
        labels: labels,
        datasets: [
          {
            label: 'Stats',
            data: values,
            borderColor: accent,
            backgroundColor: accent + '33',
            borderWidth: 2,
            pointBackgroundColor: accent,
            pointRadius: 3,
            pointHoverRadius: 4,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              label: function (ctx) {
                var v = ctx.parsed.r;
                if (typeof v !== 'number') {
                  return v;
                }
                return v.toFixed(2);
              },
            },
          },
        },
        scales: {
          r: {
            min: 0,
            max: maxR,
            ticks: {
              stepSize: step,
              showLabelBackdrop: false,
              color: ink + '99',
              font: { size: 10 },
            },
            pointLabels: {
              color: ink,
              font: { size: 11 },
            },
            grid: { color: ink + '22' },
            angleLines: { color: ink + '22' },
          },
        },
      },
    });
  }

  function boot() {
    document
      .querySelectorAll('[data-testid="life-radar-chart"]')
      .forEach(initLifeRadar);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
  document.body.addEventListener('htmx:afterSettle', boot);
})();
