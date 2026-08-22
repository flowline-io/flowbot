(function () {
  'use strict';

  // Teal + greys only (web-ui chart palette); no semantic colors for series.
  var palette = {
    primary: '#0f766e',
    primarySoft: '#0d9488',
    tealLight: '#5eead4',
    grey: '#64748b',
    greySoft: '#94a3b8',
  };

  function parseStats(container) {
    if (!container || !container.dataset.stats) {
      return null;
    }
    try {
      return JSON.parse(container.dataset.stats);
    } catch {
      return null;
    }
  }

  function shortLabels(labels) {
    return (labels || []).map(function (d) {
      if (typeof d !== 'string' || d.length < 10) {
        return d;
      }
      return d.slice(5);
    });
  }

  function baseOptions() {
    return {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          labels: {
            boxWidth: 12,
            color: palette.grey,
          },
        },
      },
    };
  }

  function initChart(canvas, stats) {
    var type = canvas.dataset.chartType;
    if (!type || canvas.chartInstance || !stats) {
      return;
    }
    var labels = shortLabels(stats.day_labels);
    var opts = baseOptions();

    if (type === 'activity') {
      canvas.chartInstance = new Chart(canvas, {
        type: 'line',
        data: {
          labels: labels,
          datasets: [
            {
              label: flowbotI18n('client.stats.completions', 'Completions'),
              data: stats.activity_counts || [],
              borderColor: palette.primary,
              backgroundColor: palette.primary + '33',
              fill: true,
              tension: 0.25,
              pointRadius: 2,
              yAxisID: 'y',
            },
            {
              label: flowbotI18n('client.stats.exp', 'EXP'),
              data: stats.activity_exp || [],
              borderColor: palette.grey,
              backgroundColor: 'transparent',
              fill: false,
              tension: 0.25,
              pointRadius: 2,
              yAxisID: 'y1',
            },
          ],
        },
        options: Object.assign({}, opts, {
          interaction: { mode: 'index', intersect: false },
          scales: {
            y: { beginAtZero: true, ticks: { stepSize: 1 }, position: 'left' },
            y1: {
              beginAtZero: true,
              position: 'right',
              grid: { drawOnChartArea: false },
            },
          },
        }),
      });
      return;
    }

    if (type === 'growth') {
      canvas.chartInstance = new Chart(canvas, {
        type: 'bar',
        data: {
          labels: stats.growth_labels || [],
          datasets: [
            {
              label: flowbotI18n('client.stats.growth_exp', 'EXP'),
              data: stats.growth_values || [],
              backgroundColor: palette.primary,
            },
          ],
        },
        options: Object.assign({}, opts, {
          indexAxis: 'y',
          plugins: { legend: { display: false } },
          scales: { x: { beginAtZero: true } },
        }),
      });
      return;
    }

    if (type === 'quests') {
      canvas.chartInstance = new Chart(canvas, {
        type: 'bar',
        data: {
          labels: stats.quest_type_labels || [],
          datasets: [
            {
              label: flowbotI18n('client.stats.quest_completed', 'Completed'),
              data: stats.quest_type_values || [],
              backgroundColor: [
                palette.primary,
                palette.primarySoft,
                palette.grey,
              ],
            },
          ],
        },
        options: Object.assign({}, opts, {
          plugins: { legend: { display: false } },
          scales: { y: { beginAtZero: true, ticks: { stepSize: 1 } } },
        }),
      });
      return;
    }

    if (type === 'economy') {
      canvas.chartInstance = new Chart(canvas, {
        type: 'line',
        data: {
          labels: labels,
          datasets: [
            {
              label: flowbotI18n('client.stats.gold_in', 'Gold in'),
              data: stats.gold_in || [],
              borderColor: palette.primary,
              backgroundColor: palette.primary + '22',
              fill: true,
              tension: 0.25,
              pointRadius: 2,
            },
            {
              label: flowbotI18n('client.stats.gold_out', 'Gold out'),
              data: stats.gold_out || [],
              borderColor: palette.grey,
              backgroundColor: palette.grey + '22',
              fill: true,
              tension: 0.25,
              pointRadius: 2,
            },
          ],
        },
        options: Object.assign({}, opts, {
          scales: { y: { beginAtZero: true } },
        }),
      });
      return;
    }

    if (type === 'drops') {
      canvas.chartInstance = new Chart(canvas, {
        type: 'bar',
        data: {
          labels: labels,
          datasets: [
            {
              label: flowbotI18n('client.stats.drops', 'Drops'),
              data: stats.drops || [],
              backgroundColor: palette.primary,
            },
          ],
        },
        options: Object.assign({}, opts, {
          plugins: { legend: { display: false } },
          scales: { y: { beginAtZero: true, ticks: { stepSize: 1 } } },
        }),
      });
    }
  }

  function destroyCharts(container) {
    if (!container) {
      return;
    }
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
    var container = document.getElementById('life-stats-container');
    if (!container || !container.dataset.stats) {
      return;
    }
    var stats = parseStats(container);
    container.querySelectorAll('canvas[data-chart-type]').forEach(function (c) {
      initChart(c, stats);
    });
  }

  function loadPanel() {
    var el = document.querySelector('[data-testid="life-stats-loader"]');
    if (!el || typeof htmx === 'undefined') {
      return;
    }
    if (el.dataset.stats) {
      return;
    }
    var tz = 'UTC';
    try {
      tz = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
    } catch {
      tz = 'UTC';
    }
    var url = '/service/web/life/stats/panel?tz=' + encodeURIComponent(tz);
    htmx.ajax('GET', url, { target: el, swap: 'outerHTML' });
  }

  document.addEventListener('htmx:beforeSwap', function (evt) {
    if (evt.detail.target && evt.detail.target.id === 'life-stats-container') {
      destroyCharts(evt.detail.target);
    }
  });

  document.addEventListener('htmx:afterSettle', function () {
    initAll();
  });

  document.addEventListener('DOMContentLoaded', function () {
    loadPanel();
    initAll();
  });
})();
