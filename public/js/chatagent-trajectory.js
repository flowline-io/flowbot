(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var states = new WeakMap();

  function stateFor(root) {
    var st = states.get(root);
    if (st) {
      return st;
    }
    st = {
      view: 'chat',
      rows: [],
      selectedId: '',
      inspectorTab: 'preview',
      loaded: false,
      loading: false,
    };
    states.set(root, st);
    return st;
  }

  function setView(root, view, pushUrl) {
    var st = stateFor(root);
    st.view = view === 'trajectory' ? 'trajectory' : 'chat';
    root.setAttribute('data-session-view', st.view);
    var messages = root.querySelector('#chatagent-messages');
    var traj = root.querySelector('#chatagent-trajectory');
    if (messages) {
      messages.classList.toggle('hidden', st.view === 'trajectory');
      messages.hidden = st.view === 'trajectory';
    }
    if (traj) {
      traj.classList.toggle('hidden', st.view !== 'trajectory');
      traj.hidden = st.view !== 'trajectory';
    }
    root.querySelectorAll('[data-chatagent-view]').forEach(function (btn) {
      var active = btn.getAttribute('data-chatagent-view') === st.view;
      btn.classList.toggle('is-active', active);
      btn.setAttribute('aria-selected', active ? 'true' : 'false');
    });
    if (pushUrl) {
      try {
        var url = new URL(window.location.href);
        if (st.view === 'trajectory') {
          url.searchParams.set('view', 'trajectory');
        } else {
          url.searchParams.delete('view');
        }
        window.history.replaceState({}, '', url);
      } catch {
        /* ignore */
      }
    }
    if (st.view === 'trajectory') {
      loadTrajectory(root);
    } else if (ns.updateJumpBottomButton && messages) {
      ns.updateJumpBottomButton(messages);
    }
  }

  function upsertRow(st, row) {
    if (!row || !row.id) {
      return;
    }
    for (var i = 0; i < st.rows.length; i++) {
      if (st.rows[i].id === row.id) {
        st.rows[i] = row;
        return;
      }
    }
    st.rows.push(row);
  }

  function rowsFromTurnTrace(ev) {
    if (!ev || !Array.isArray(ev.sections)) {
      return [];
    }
    var rows = [];
    var seen = {};
    ev.sections.forEach(function (section) {
      if (!section) {
        return;
      }
      var name = String(section.name || '').trim();
      if (!name) {
        return;
      }
      seen[name] = (seen[name] || 0) + 1;
      var id = (ev.id || 'trace') + '/' + name;
      if (seen[name] > 1) {
        id += '/' + seen[name];
      }
      var isSystem = name === 'system_body';
      rows.push({
        id: id,
        turn: 0,
        role: isSystem ? 'system' : 'context',
        kind: isSystem ? 'system' : 'context',
        text: section.content || '',
        assemble_ms: isSystem ? ev.assemble_ms || 0 : 0,
        raw: section,
      });
    });
    return rows;
  }

  function loadTrajectory(root) {
    var url = root.getAttribute('data-trajectory-url');
    if (!url) {
      return;
    }
    var st = stateFor(root);
    if (st.loading) {
      return;
    }
    st.loading = true;
    fetch(url, { headers: { Accept: 'application/json' } })
      .then(function (res) {
        return res.json().then(function (body) {
          if (!res.ok) {
            throw new Error((body && body.error) || 'trajectory failed');
          }
          return body;
        });
      })
      .then(function (body) {
        var incoming = (body && body.rows) || [];
        incoming.forEach(function (row) {
          upsertRow(st, row);
        });
        st.loaded = true;
        st.loading = false;
        renderTrajectory(root);
      })
      .catch(function () {
        st.loaded = true;
        st.loading = false;
        renderTrajectory(root);
      });
  }

  function durationTotals(rows) {
    var input = 0;
    var thinking = 0;
    var turnMs = 0;
    var tools = 0;
    rows.forEach(function (row) {
      if (row.kind === 'system' && row.assemble_ms) {
        input += row.assemble_ms;
      }
      if (row.kind === 'thinking') {
        thinking += row.duration_ms || 0;
      }
      if (row.kind === 'assistant') {
        turnMs += row.duration_ms || 0;
      }
      if (row.kind === 'tool') {
        tools += row.duration_ms || 0;
      }
    });
    var model = turnMs > 0 ? Math.max(0, turnMs - tools) : thinking;
    return { input: input, model: model, tools: tools };
  }

  function renderGantt(root, rows) {
    var el = root.querySelector('#chatagent-trajectory-gantt');
    if (!el) {
      return;
    }
    var tot = durationTotals(rows);
    var max = Math.max(tot.input, tot.model, tot.tools, 1);
    function track(label, ms, cls) {
      var pct = Math.max(2, Math.round((ms / max) * 100));
      return (
        '<div class="chatagent-gantt-row">' +
        '<span class="chatagent-gantt-label">' +
        label +
        '</span>' +
        '<div class="chatagent-gantt-track"><span class="chatagent-gantt-bar ' +
        cls +
        '" style="width:' +
        pct +
        '%"></span></div>' +
        '<span class="chatagent-gantt-ms">' +
        ms +
        ' ms</span></div>'
      );
    }
    el.innerHTML =
      track('Input', tot.input, 'is-input') +
      track('Model', tot.model, 'is-model') +
      track('Tools', tot.tools, 'is-tools');
  }

  function rowChipLabel(row) {
    if (row.kind === 'tool_call') {
      return 'CALL';
    }
    return (row.role || row.kind || '').toUpperCase();
  }

  function rowPreview(row) {
    if (!row) {
      return '';
    }
    if (row.kind === 'tool_call') {
      return row.text || row.tool_name || '';
    }
    if (row.kind === 'tool') {
      var name = row.tool_name || 'tool';
      if (row.subagent) {
        name = row.subagent;
      }
      if (row.tool_status === 'error') {
        name += ' · error';
      }
      var body = row.text || '';
      if (!body) {
        return name;
      }
      return name + ' → ' + body;
    }
    return row.text || row.tool_name || '';
  }

  function truncatePreview(text) {
    if (!text || text.length <= 240) {
      return text || '';
    }
    return text.slice(0, 240) + '…';
  }

  function renderLog(root, rows) {
    var el = root.querySelector('#chatagent-trajectory-log');
    if (!el) {
      return;
    }
    var st = stateFor(root);
    el.innerHTML = '';
    rows.forEach(function (row) {
      var item = document.createElement('button');
      item.type = 'button';
      item.className = 'chatagent-trajectory-row';
      item.setAttribute('data-testid', 'chatagent-trajectory-row');
      item.setAttribute('data-row-id', row.id);
      if (row.kind) {
        item.setAttribute('data-row-kind', row.kind);
      }
      if (row.tool_name) {
        item.setAttribute('data-tool-name', row.tool_name);
      }
      if (row.id === st.selectedId) {
        item.classList.add('is-selected');
      }
      var chip = document.createElement('span');
      chip.className =
        'flowbot-chip chatagent-trajectory-chip is-' + (row.kind || row.role);
      chip.textContent = rowChipLabel(row);
      var body = document.createElement('span');
      body.className = 'chatagent-trajectory-row-text';
      body.textContent = truncatePreview(rowPreview(row));
      item.appendChild(chip);
      item.appendChild(body);
      item.addEventListener('click', function () {
        st.selectedId = row.id;
        renderTrajectory(root);
      });
      el.appendChild(item);
    });
  }

  function selectedRow(st) {
    for (var i = 0; i < st.rows.length; i++) {
      if (st.rows[i].id === st.selectedId) {
        return st.rows[i];
      }
    }
    return null;
  }

  function inspectorTitle(row) {
    var parts = [(row.role || row.kind || '').toUpperCase()];
    if (row.turn) {
      parts[0] += ' Turn ' + row.turn;
    }
    if (row.tool_name) {
      parts.push(row.tool_name);
    } else if (row.subagent) {
      parts.push(row.subagent);
    }
    return parts.join(' · ');
  }

  function inspectorPreview(row) {
    if (row.kind === 'tool_call') {
      return row.text || '';
    }
    if (row.kind === 'tool') {
      var lines = [];
      var raw = row.raw && typeof row.raw === 'object' ? row.raw : null;
      var name = row.tool_name || (raw && raw.name) || '';
      var args = raw && raw.arguments != null ? String(raw.arguments) : '';
      if (name) {
        lines.push(name);
      }
      if (args) {
        lines.push(args);
      }
      if (row.text) {
        if (lines.length) {
          lines.push('');
        }
        lines.push(row.text);
      }
      return lines.join('\n');
    }
    return row.text || '';
  }

  function renderInspector(root) {
    var st = stateFor(root);
    var panel = root.querySelector('#chatagent-trajectory-inspector');
    var title = root.querySelector('#chatagent-trajectory-inspector-title');
    var body = root.querySelector('#chatagent-trajectory-inspector-body');
    var row = selectedRow(st);
    if (!panel || !body) {
      return;
    }
    if (!row) {
      panel.classList.add('hidden');
      panel.hidden = true;
      return;
    }
    panel.classList.remove('hidden');
    panel.hidden = false;
    if (title) {
      title.textContent = inspectorTitle(row);
    }
    root.querySelectorAll('[data-inspector-tab]').forEach(function (btn) {
      btn.classList.toggle(
        'is-active',
        btn.getAttribute('data-inspector-tab') === st.inspectorTab,
      );
    });
    if (st.inspectorTab === 'raw') {
      body.textContent = JSON.stringify(
        row.raw != null ? row.raw : row,
        null,
        2,
      );
    } else {
      body.textContent = inspectorPreview(row);
    }
  }

  function renderTrajectory(root) {
    var st = stateFor(root);
    renderGantt(root, st.rows);
    renderLog(root, st.rows);
    renderInspector(root);
  }

  ns.handleTrajectoryStreamEvent = function (ev, root) {
    if (!ev || !root) {
      return;
    }
    var st = stateFor(root);
    if (ev.type === 'turn_trace') {
      rowsFromTurnTrace(ev).forEach(function (row) {
        upsertRow(st, row);
      });
      if (st.view === 'trajectory') {
        renderTrajectory(root);
      }
    }
  };

  ns.initTrajectoryView = function (root) {
    if (!root || root.getAttribute('data-trajectory-bound') === '1') {
      return;
    }
    if (!root.getAttribute('data-trajectory-url')) {
      return;
    }
    root.setAttribute('data-trajectory-bound', '1');
    var st = stateFor(root);
    root.querySelectorAll('[data-chatagent-view]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        setView(root, btn.getAttribute('data-chatagent-view'), true);
      });
    });
    var closeBtn = root.querySelector(
      '[data-testid="chatagent-trajectory-inspector-close"]',
    );
    if (closeBtn) {
      closeBtn.addEventListener('click', function () {
        st.selectedId = '';
        renderInspector(root);
        renderLog(root, st.rows);
      });
    }
    root.querySelectorAll('[data-inspector-tab]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        st.inspectorTab = btn.getAttribute('data-inspector-tab') || 'preview';
        renderInspector(root);
      });
    });
    var initial = 'chat';
    try {
      if (
        new URL(window.location.href).searchParams.get('view') === 'trajectory'
      ) {
        initial = 'trajectory';
      }
    } catch {
      initial = 'chat';
    }
    setView(root, initial, false);
  };

  document.addEventListener('DOMContentLoaded', function () {
    document
      .querySelectorAll('[data-chatagent-root="thread"]')
      .forEach(function (root) {
        ns.initTrajectoryView(root);
      });
  });
})();
