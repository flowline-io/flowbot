(function () {
  'use strict';

  var TEAL = '#0f766e';
  var GREY = '#94a3b8';
  var FAIL = '#ef4444';

  function parseGraph(el) {
    if (!el) return null;
    var raw = el.textContent || el.dataset.graph || '';
    if (!raw) return null;
    try {
      return JSON.parse(raw);
    } catch {
      return null;
    }
  }

  function edgeFailed(edge) {
    return (edge.count_failed || 0) > 0;
  }

  function edgeWeight(edge) {
    return Math.max(1, (edge.count_ok || 0) + (edge.count_failed || 0));
  }

  function edgeColor(edge) {
    if (edgeFailed(edge)) return FAIL;
    if (edge.static) return GREY;
    return TEAL;
  }

  function runListURL(edge) {
    var via = edge.via || {};
    var kind = via.kind || '';
    var name = via.name || '';
    if (!name) return '';
    if (kind === 'pipeline') return '/service/web/pipelines/' + encodeURIComponent(name) + '/runs';
    if (kind === 'workflow') return '/service/web/workflows/' + encodeURIComponent(name) + '/runs';
    if (kind === 'function') return '/service/web/functions/' + encodeURIComponent(name) + '/runs';
    return '';
  }

  function toGraphSeries(graph) {
    var nodes = (graph.nodes || []).map(function (n) {
      return {
        id: n.id,
        name: n.id,
        symbolSize: 36,
        category: 0,
      };
    });
    var links = (graph.edges || []).map(function (e) {
      var w = edgeWeight(e);
      return {
        source: e.source,
        target: e.target,
        value: w,
        lineStyle: {
          color: edgeColor(e),
          width: Math.min(8, 1 + Math.log2(w + 1)),
          type: e.static ? 'dashed' : 'solid',
          curveness: 0.15,
        },
        label: {
          show: w > 1,
          formatter: String(w),
        },
        edge: e,
      };
    });
    return { nodes: nodes, links: links };
  }

  function toSankeySeries(graph) {
    var nodeIds = {};
    (graph.nodes || []).forEach(function (n) {
      nodeIds[n.id] = true;
    });
    var nodes = Object.keys(nodeIds).map(function (id) {
      return { name: id };
    });
    var links = (graph.edges || []).map(function (e) {
      return {
        source: e.source,
        target: e.target,
        value: edgeWeight(e),
        lineStyle: {
          color: edgeColor(e),
          opacity: 0.55,
        },
        edge: e,
      };
    });
    return { nodes: nodes, links: links };
  }

  function toTreeSeries(graph, rootId) {
    var childrenOf = {};
    (graph.edges || []).forEach(function (e) {
      if (!childrenOf[e.source]) childrenOf[e.source] = [];
      childrenOf[e.source].push({
        name: e.target,
        value: edgeWeight(e),
        itemStyle: { color: edgeFailed(e) ? FAIL : TEAL },
        edge: e,
      });
    });
    function build(id, depth) {
      if (depth > 6) return { name: id };
      var kids = (childrenOf[id] || []).map(function (c) {
        var node = build(c.name, depth + 1);
        node.value = c.value;
        node.itemStyle = c.itemStyle;
        node.edge = c.edge;
        return node;
      });
      return { name: id, children: kids };
    }
    var root = rootId || ((graph.nodes || [])[0] && graph.nodes[0].id) || 'root';
    return build(root, 0);
  }

  function mountChart(host) {
    if (!host || !window.echarts) return null;
    if (host.flowbotChart) {
      host.flowbotChart.dispose();
      host.flowbotChart = null;
    }
    var chart = window.echarts.init(host, null, { renderer: 'canvas' });
    host.flowbotChart = chart;
    return chart;
  }

  function renderTopology(host, graph) {
    var chart = mountChart(host);
    if (!chart) return;
    var series = toGraphSeries(graph);
    chart.setOption({
      tooltip: {
        formatter: function (p) {
          if (p.dataType === 'edge' || (p.data && p.data.edge)) {
            var e = (p.data && p.data.edge) || {};
            var via = e.via || {};
            return (
              (e.source || '') +
              ' → ' +
              (e.target || '') +
              '<br/>ok: ' +
              (e.count_ok || 0) +
              ' / failed: ' +
              (e.count_failed || 0) +
              '<br/>via: ' +
              (via.kind || '') +
              ' ' +
              (via.name || '')
            );
          }
          return p.name || '';
        },
      },
      legend: { show: false },
      series: [
        {
          type: 'graph',
          layout: 'force',
          roam: true,
          draggable: true,
          force: { repulsion: 180, edgeLength: [60, 140] },
          label: { show: true, position: 'bottom' },
          edgeSymbol: ['none', 'arrow'],
          edgeSymbolSize: [0, 10],
          data: series.nodes,
          links: series.links,
          lineStyle: { opacity: 0.85 },
          emphasis: { focus: 'adjacency' },
        },
      ],
    });
    chart.off('click');
    chart.on('click', function (params) {
      var edge = params.data && params.data.edge;
      if (!edge) return;
      var url = runListURL(edge);
      if (url) window.location.href = url;
    });
  }

  function renderSankey(host, graph) {
    var chart = mountChart(host);
    if (!chart) return;
    var series = toSankeySeries(graph);
    chart.setOption({
      tooltip: { trigger: 'item' },
      series: [
        {
          type: 'sankey',
          emphasis: { focus: 'adjacency' },
          data: series.nodes,
          links: series.links,
          lineStyle: { color: 'gradient', curveness: 0.5 },
          label: { color: 'inherit' },
        },
      ],
    });
    chart.off('click');
    chart.on('click', function (params) {
      var edge = params.data && params.data.edge;
      if (!edge) return;
      var url = runListURL(edge);
      if (url) window.location.href = url;
    });
  }

  function renderTree(host, graph, rootId) {
    var chart = mountChart(host);
    if (!chart) return;
    chart.setOption({
      tooltip: { trigger: 'item' },
      series: [
        {
          type: 'tree',
          orient: 'LR',
          expandAndCollapse: true,
          initialTreeDepth: 3,
          label: { position: 'left', verticalAlign: 'middle' },
          leaves: { label: { position: 'right' } },
          data: [toTreeSeries(graph, rootId)],
        },
      ],
    });
  }

  function activeMode(root) {
    var btn = root.querySelector('[data-flow-mode].btn-active');
    return (btn && btn.getAttribute('data-flow-mode')) || 'topology';
  }

  function renderRoot(root) {
    if (!root || root.getAttribute('data-flow-ready') === 'pending') return;
    var dataEl = root.querySelector('[data-flow-graph-json], #flow-graph-data');
    var host = root.querySelector('[data-flow-chart]');
    if (!host) return;
    var graph = parseGraph(dataEl);
    if (!graph || (!(graph.nodes || []).length && !(graph.edges || []).length)) {
      host.innerHTML = '';
      host.setAttribute('data-empty', '1');
      return;
    }
    host.removeAttribute('data-empty');
    var mode = root.getAttribute('data-flow-kind') || activeMode(root);
    if (mode === 'sankey') {
      renderSankey(host, graph);
    } else if (mode === 'tree') {
      renderTree(host, graph, root.getAttribute('data-flow-root') || '');
    } else {
      renderTopology(host, graph);
    }
  }

  function bindModeButtons(root) {
    root.querySelectorAll('[data-flow-mode]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        root.querySelectorAll('[data-flow-mode]').forEach(function (b) {
          b.classList.remove('btn-active');
        });
        btn.classList.add('btn-active');
        root.setAttribute('data-flow-kind', btn.getAttribute('data-flow-mode') || 'topology');
        renderRoot(root);
      });
    });
  }

  function initAll(scope) {
    var roots = (scope || document).querySelectorAll('[data-flow-graph]');
    roots.forEach(function (root) {
      if (root.flowbotBound) {
        renderRoot(root);
        return;
      }
      root.flowbotBound = true;
      bindModeButtons(root);
      renderRoot(root);
    });
  }

  function onReady() {
    initAll(document);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', onReady);
  } else {
    onReady();
  }

  document.body.addEventListener('htmx:afterSwap', function (evt) {
    initAll(evt.target);
  });

  window.addEventListener('resize', function () {
    document.querySelectorAll('[data-flow-chart]').forEach(function (host) {
      if (host.flowbotChart) host.flowbotChart.resize();
    });
  });

  window.FlowbotFlowGraph = {
    initAll: initAll,
    renderRoot: renderRoot,
  };
})();
