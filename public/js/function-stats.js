(function () {
  'use strict';

  var colors = {
    primary:
      getComputedStyle(document.documentElement).getPropertyValue('--p') ||
      '#0f766e',
    success: '#22c55e',
    error: '#ef4444',
    warning: '#f59e0b',
    info: '#06b6d4',
  };

  function initChart(canvas) {
    var type = canvas.dataset.chartType;
    if (!type || canvas.chartInstance) return;

    var statsEl = document.getElementById('function-chart-success-rate');
    if (!statsEl || !statsEl.dataset.stats) return;

    var stats;
    try {
      stats = JSON.parse(statsEl.dataset.stats);
    } catch {
      return;
    }

    if (type === 'line') {
      var trend = stats.success_rate_trend || [];
      canvas.chartInstance = new Chart(canvas, {
        type: 'line',
        data: {
          labels: trend.map(function (p) {
            return p.date;
          }),
          datasets: [
            {
              label: 'Success Rate',
              data: trend.map(function (p) {
                return +(p.rate * 100).toFixed(1);
              }),
              borderColor: colors.success,
              backgroundColor: colors.success + '20',
              fill: true,
              tension: 0.2,
              pointRadius: 3,
            },
          ],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          scales: {
            y: {
              min: 0,
              max: 100,
              ticks: {
                callback: function (v) {
                  return v + '%';
                },
              },
            },
          },
          plugins: { legend: { display: false } },
        },
      });
    } else if (type === 'bar') {
      var buckets = stats.duration_distribution || [];
      canvas.chartInstance = new Chart(canvas, {
        type: 'bar',
        data: {
          labels: buckets.map(function (b) {
            return b.bucket;
          }),
          datasets: [
            {
              label: 'Function Runs',
              data: buckets.map(function (b) {
                return b.count;
              }),
              backgroundColor: colors.primary,
            },
          ],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: { legend: { display: false } },
          scales: { y: { beginAtZero: true, ticks: { stepSize: 1 } } },
        },
      });
    } else if (type === 'doughnut') {
      var pie = stats.version_pie || [];
      canvas.chartInstance = new Chart(canvas, {
        type: 'doughnut',
        data: {
          labels: pie.map(function (s) {
            return s.version;
          }),
          datasets: [
            {
              data: pie.map(function (s) {
                return s.count;
              }),
              backgroundColor: [
                colors.primary,
                colors.success,
                colors.warning,
                colors.info,
              ],
            },
          ],
        },
        options: { responsive: true, maintainAspectRatio: false },
      });
    }
  }

  function destroyCharts(container) {
    container.querySelectorAll('canvas').forEach(function (c) {
      if (c.chartInstance) {
        c.chartInstance.destroy();
        c.chartInstance = null;
      }
    });
  }

  function initAll() {
    if (typeof Chart === 'undefined') {
      return;
    }
    document
      .querySelectorAll('#function-stats-container canvas[data-chart-type]')
      .forEach(initChart);
  }

  document.addEventListener('htmx:beforeSwap', function (evt) {
    if (evt.detail.target.id === 'function-stats-container') {
      destroyCharts(evt.detail.target);
    }
  });

  document.addEventListener('htmx:afterSettle', function (_evt) {
    var container = document.getElementById('function-stats-container');
    if (container) initAll();
  });

  document.addEventListener('DOMContentLoaded', initAll);
})();
