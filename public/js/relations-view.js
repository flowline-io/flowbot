(function () {
  'use strict';

  var selectedNode = '';

  function pane(name) {
    return document.querySelector('[data-relations-pane="' + name + '"]');
  }

  function setTab(name) {
    document.querySelectorAll('[data-relations-tab]').forEach(function (tab) {
      var active = tab.getAttribute('data-relations-tab') === name;
      tab.classList.toggle('tab-active', active);
      tab.setAttribute('aria-selected', active ? 'true' : 'false');
    });
    var tree = pane('tree');
    var graph = pane('graph');
    if (tree) tree.classList.toggle('hidden', name !== 'tree');
    if (graph) graph.classList.toggle('hidden', name !== 'graph');
    if (name === 'graph') {
      loadGraph();
    }
  }

  function currentFilters() {
    var pipeline = document.querySelector('[data-testid="relations-pipeline-filter"]');
    var since = document.querySelector('[data-testid="relations-since-filter"]');
    return {
      pipeline: pipeline ? pipeline.value : '',
      since: since ? since.value : '',
    };
  }

  function loadGraph() {
    var root = pane('graph');
    if (!root) return;
    var empty = document.getElementById('relations-graph-empty');
    var chart = root.querySelector('[data-flow-chart]');
    var dataEl = document.getElementById('flow-graph-data');
    if (!selectedNode) {
      if (empty) empty.classList.remove('hidden');
      if (chart) chart.classList.add('hidden');
      return;
    }
    var f = currentFilters();
    var url =
      '/service/web/relations/graph?node=' +
      encodeURIComponent(selectedNode) +
      '&pipeline=' +
      encodeURIComponent(f.pipeline || '') +
      '&since=' +
      encodeURIComponent(f.since || '');
    fetch(url, { credentials: 'same-origin', headers: { Accept: 'application/json' } })
      .then(function (r) {
        if (!r.ok) throw new Error('graph fetch failed');
        return r.json();
      })
      .then(function (graph) {
        if (dataEl) dataEl.textContent = JSON.stringify(graph || { nodes: [], edges: [] });
        var has =
          graph &&
          ((graph.nodes && graph.nodes.length) || (graph.edges && graph.edges.length));
        if (empty) empty.classList.toggle('hidden', !!has);
        if (chart) chart.classList.toggle('hidden', !has);
        root.setAttribute('data-flow-kind', 'topology');
        if (window.FlowbotFlowGraph) {
          window.FlowbotFlowGraph.renderRoot(root);
        }
      })
      .catch(function () {
        if (empty) empty.classList.remove('hidden');
        if (chart) chart.classList.add('hidden');
      });
  }

  function rememberNodeFromURL(url) {
    try {
      var u = new URL(url, window.location.origin);
      var node = u.searchParams.get('node');
      if (node) selectedNode = node;
    } catch {
      /* intentionally silent */
    }
  }

  document.addEventListener('click', function (evt) {
    var tab = evt.target.closest('[data-relations-tab]');
    if (tab) {
      setTab(tab.getAttribute('data-relations-tab') || 'tree');
    }
  });

  document.body.addEventListener('htmx:beforeRequest', function (evt) {
    var path = evt.detail && evt.detail.pathInfo && evt.detail.pathInfo.requestPath;
    if (path && path.indexOf('/relations/tree') !== -1) {
      rememberNodeFromURL(path);
    }
  });

  document.body.addEventListener('htmx:afterSwap', function (evt) {
    if (evt.target && (evt.target.id === 'relations-tree' || evt.target.querySelector && evt.target.querySelector('#relations-tree, [data-testid="relations-tree"]'))) {
      var hidden = document.querySelector('#relations-tree [name="node"], [data-testid="relations-tree"] [name="node"]');
      if (hidden && hidden.value) selectedNode = hidden.value;
      var graphTab = document.querySelector('[data-relations-tab="graph"].tab-active');
      if (graphTab) loadGraph();
    }
  });
})();
