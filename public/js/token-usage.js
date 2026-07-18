(function () {
  'use strict';

  var palette = [
    '#0f766e',
    '#0d9488',
    '#5eead4',
    '#64748b',
    '#f59e0b',
    '#ef4444',
  ];

  function formatAxis(value) {
    if (value >= 1e9) {
      return (value / 1e9).toFixed(1) + 'B';
    }
    if (value >= 1e6) {
      return (value / 1e6).toFixed(1) + 'M';
    }
    if (value >= 1e3) {
      return (value / 1e3).toFixed(1) + 'K';
    }
    return String(value);
  }

  function initChart(canvas) {
    if (!canvas || canvas.chartInstance) {
      return;
    }
    if (!canvas.dataset.stats) {
      return;
    }

    var stats;
    try {
      stats = JSON.parse(canvas.dataset.stats);
    } catch {
      return;
    }

    var series = stats.series || [];
    if (series.length === 0) {
      return;
    }

    var labels = series[0].points.map(function (p) {
      return p.date;
    });

    var datasets = series.map(function (s, idx) {
      var color = palette[idx % palette.length];
      return {
        label: s.label,
        data: s.points.map(function (p) {
          return p.cumulative;
        }),
        daily: s.points.map(function (p) {
          return p.daily;
        }),
        borderColor: color,
        backgroundColor: color + '33',
        fill: true,
        tension: 0.2,
        pointRadius: 2,
        pointHoverRadius: 4,
      };
    });

    canvas.chartInstance = new Chart(canvas, {
      type: 'line',
      data: { labels: labels, datasets: datasets },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: 'index', intersect: false },
        scales: {
          x: {
            ticks: { maxRotation: 45, minRotation: 0 },
          },
          y: {
            stacked: false,
            beginAtZero: true,
            title: { display: true, text: 'Cumulative Tokens' },
            ticks: {
              callback: function (v) {
                return formatAxis(v);
              },
            },
          },
        },
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              label: function (ctx) {
                var daily = ctx.dataset.daily[ctx.dataIndex];
                return ctx.dataset.label + ': ' + formatAxis(daily);
              },
            },
          },
        },
      },
    });

    positionTodayMarker(canvas, stats, labels);
  }

  function positionTodayMarker(canvas, stats, labels) {
    var wrap = canvas.closest('[data-testid="token-usage-chart-wrap"]');
    if (!wrap) {
      return;
    }
    var marker = wrap.querySelector('#token-usage-today-marker');
    var label = wrap.querySelector('#token-usage-today-label');
    if (!marker || !label || !stats.today) {
      return;
    }

    var todayIdx = labels.indexOf(stats.today);
    if (todayIdx < 0) {
      marker.classList.add('hidden');
      label.classList.add('hidden');
      return;
    }

    var chart = canvas.chartInstance;
    if (!chart) {
      return;
    }

    chart.update();
    var meta = chart.getDatasetMeta(0);
    if (!meta || !meta.data[todayIdx]) {
      return;
    }

    var x = meta.data[todayIdx].x;
    marker.style.left = x + 'px';
    marker.classList.remove('hidden');
    label.style.left = x + 4 + 'px';
    label.style.top = '4px';
    label.classList.remove('hidden');
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
    document
      .querySelectorAll('#token-usage-container canvas[data-chart-type]')
      .forEach(initChart);
  }

  document.addEventListener('htmx:beforeSwap', function (evt) {
    if (evt.detail.target.id === 'token-usage-container') {
      destroyCharts(evt.detail.target);
    }
  });

  document.addEventListener('htmx:afterSettle', function () {
    var container = document.getElementById('token-usage-container');
    if (container) {
      initAll();
    }
  });

  document.addEventListener('DOMContentLoaded', initAll);
})();
