(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var TODO_TOOL_NAMES = { todo_write: true, list_todos: true };

  function statusSlug(status) {
    switch (status) {
      case 'in_progress':
        return 'in-progress';
      case 'completed':
        return 'completed';
      case 'cancelled':
        return 'cancelled';
      default:
        return 'pending';
    }
  }

  function todosProgress(todos) {
    var total = todos ? todos.length : 0;
    var done = 0;
    var active = 0;
    for (var i = 0; i < total; i++) {
      var status = todos[i].status || 'pending';
      if (status === 'completed') {
        done++;
      } else if (status !== 'cancelled') {
        active++;
      }
    }
    return { done: done, total: total, active: active };
  }

  function countLabel(todos) {
    if (!todos || todos.length === 0) {
      return '0 items';
    }
    var progress = todosProgress(todos);
    if (progress.done === progress.total) {
      return progress.done + '/' + progress.total + ' done';
    }
    if (progress.active <= 0) {
      return progress.done + '/' + progress.total + ' done';
    }
    return (
      progress.active + ' active · ' + progress.done + '/' + progress.total
    );
  }

  function progressPercentLabel(todos) {
    var progress = todosProgress(todos);
    if (progress.total === 0) {
      return '0%';
    }
    return String(Math.round((progress.done * 100) / progress.total)) + '%';
  }

  function progressWidth(todos) {
    var progress = todosProgress(todos);
    if (progress.total === 0) {
      return '0%';
    }
    return String(Math.round((progress.done * 100) / progress.total)) + '%';
  }

  function parseTodoSnapshot(stdout) {
    if (!stdout) {
      return null;
    }
    var text = String(stdout).trim();
    if (!text) {
      return null;
    }
    var start = text.indexOf('{"todos"');
    if (start < 0) {
      start = text.indexOf('{');
    }
    if (start > 0) {
      text = text.slice(start);
    }
    try {
      var payload = JSON.parse(text);
      if (!payload || !Array.isArray(payload.todos)) {
        return null;
      }
      return payload.todos;
    } catch {
      return null;
    }
  }

  function renderTodoItem(item) {
    var status = item.status || 'pending';
    var slug = statusSlug(status);

    var li = document.createElement('li');
    li.className = 'chatagent-todos-item chatagent-todos-item--' + slug;
    li.setAttribute('data-testid', 'chatagent-todo-' + item.item_id);
    li.setAttribute('data-todo-id', item.item_id);
    li.setAttribute('data-todo-status', status);

    var marker = document.createElement('span');
    marker.className = 'chatagent-todos-marker chatagent-todos-marker--' + slug;
    marker.setAttribute('aria-hidden', 'true');

    var content = document.createElement('span');
    content.className = 'chatagent-todos-content';
    content.textContent = item.content || '';

    li.appendChild(marker);
    li.appendChild(content);
    return li;
  }

  function renderTodosList(listEl, todos) {
    listEl.innerHTML = '';
    if (!todos || todos.length === 0) {
      var empty = document.createElement('p');
      empty.className = 'chatagent-todos-empty';
      empty.textContent = 'No todos yet.';
      listEl.appendChild(empty);
      return;
    }
    var ul = document.createElement('ul');
    ul.className = 'chatagent-todos-list';
    for (var i = 0; i < todos.length; i++) {
      ul.appendChild(renderTodoItem(todos[i]));
    }
    listEl.appendChild(ul);
  }

  function removeTodosPanel() {
    var panel = document.getElementById('chatagent-todos-panel');
    if (panel) {
      panel.remove();
    }
  }

  function ensureTodosPanel(threadRoot) {
    var panel = document.getElementById('chatagent-todos-panel');
    if (panel) {
      return panel;
    }
    var anchor = threadRoot
      ? threadRoot.querySelector('#chatagent-messages')
      : document.getElementById('chatagent-messages');
    if (!anchor || !anchor.parentElement) {
      return null;
    }

    panel = document.createElement('details');
    panel.id = 'chatagent-todos-panel';
    panel.className =
      'chatagent-todos-panel flowbot-surface shrink-0 mx-1 mb-3';
    panel.setAttribute('data-testid', 'chatagent-todos-panel');
    panel.open = true;

    var summary = document.createElement('summary');
    summary.className = 'chatagent-todos-summary';

    var label = document.createElement('span');
    label.className = 'chatagent-todos-summary-label';
    label.textContent = 'Todos';

    var meta = document.createElement('span');
    meta.className = 'chatagent-todos-summary-meta';
    meta.setAttribute('data-testid', 'chatagent-todos-count');

    var progressWrap = document.createElement('span');
    progressWrap.className = 'chatagent-todos-progress-wrap';

    var progress = document.createElement('span');
    progress.className = 'chatagent-todos-progress';
    progress.setAttribute('aria-hidden', 'true');

    var fill = document.createElement('span');
    fill.className = 'chatagent-todos-progress-fill';
    progress.appendChild(fill);

    var progressLabel = document.createElement('span');
    progressLabel.className = 'chatagent-todos-progress-label';
    progressLabel.setAttribute('data-testid', 'chatagent-todos-progress');

    progressWrap.appendChild(progress);
    progressWrap.appendChild(progressLabel);

    summary.appendChild(label);
    summary.appendChild(meta);
    summary.appendChild(progressWrap);

    var body = document.createElement('div');
    body.className = 'chatagent-todos-body';
    body.id = 'chatagent-todos-list';
    body.setAttribute('data-testid', 'chatagent-todos-list');

    panel.appendChild(summary);
    panel.appendChild(body);
    anchor.parentElement.insertBefore(panel, anchor);
    return panel;
  }

  function updateTodosPanel(todos, threadRoot) {
    if (!todos || todos.length === 0) {
      removeTodosPanel();
      return;
    }

    ensureTodosPanel(threadRoot);
    var panel = document.getElementById('chatagent-todos-panel');
    var listEl = document.getElementById('chatagent-todos-list');
    if (!panel || !listEl) {
      return;
    }

    renderTodosList(listEl, todos);
    var countEl = panel.querySelector('[data-testid="chatagent-todos-count"]');
    if (countEl) {
      countEl.textContent = countLabel(todos);
    }
    var fill = panel.querySelector('.chatagent-todos-progress-fill');
    if (fill) {
      fill.style.width = progressWidth(todos);
    }
    var progressLabel = panel.querySelector(
      '[data-testid="chatagent-todos-progress"]',
    );
    if (progressLabel) {
      progressLabel.textContent = progressPercentLabel(todos);
    }
    panel.open = true;
  }

  function hydrateTodosFromToolCards(threadRoot) {
    if (!threadRoot) {
      return;
    }
    var latest = null;
    threadRoot
      .querySelectorAll('[data-testid="chatagent-message-tool"]')
      .forEach(function (wrap) {
        var badge = wrap.querySelector('[data-testid="chatagent-tool-name"]');
        if (!badge || !TODO_TOOL_NAMES[(badge.textContent || '').trim()]) {
          return;
        }
        var stdout = wrap.querySelector(
          '[data-testid="chatagent-tool-stdout"]',
        );
        if (!stdout) {
          return;
        }
        var parsed = parseTodoSnapshot(stdout.textContent || '');
        if (parsed) {
          latest = parsed;
        }
      });
    if (latest && latest.length > 0) {
      updateTodosPanel(latest, threadRoot);
    }
  }

  function refreshTodosFromServer(threadRoot) {
    if (!threadRoot) {
      return Promise.resolve();
    }
    var url = threadRoot.getAttribute('data-todos-url') || '';
    if (!url) {
      return Promise.resolve();
    }
    if (typeof flowbotCSRFHeadersAsync === 'function') {
      return flowbotCSRFHeadersAsync()
        .then(function (csrfHeaders) {
          return fetch(url, {
            credentials: 'same-origin',
            headers: csrfHeaders,
          });
        })
        .then(function (res) {
          if (!res.ok) {
            return null;
          }
          return res.json();
        })
        .then(function (data) {
          if (data && Array.isArray(data.todos) && data.todos.length > 0) {
            updateTodosPanel(data.todos, threadRoot);
          }
        })
        .catch(function () {
          /* intentionally silent */
        });
    }
    return fetch(url, { credentials: 'same-origin' })
      .then(function (res) {
        if (!res.ok) {
          return null;
        }
        return res.json();
      })
      .then(function (data) {
        if (data && Array.isArray(data.todos) && data.todos.length > 0) {
          updateTodosPanel(data.todos, threadRoot);
        }
      })
      .catch(function () {
        /* intentionally silent */
      });
  }

  function handleTodoToolEvent(ev, card, threadRoot) {
    if (!ev || !TODO_TOOL_NAMES[ev.name || '']) {
      return;
    }
    if (ev.status !== 'completed' && ev.status !== 'error') {
      return;
    }
    var stdout = ev.stdout || '';
    if (!stdout && card && card.stdout) {
      stdout = card.stdout.textContent || '';
    }
    var todos = parseTodoSnapshot(stdout);
    if (todos && todos.length > 0) {
      updateTodosPanel(todos, threadRoot);
    }
  }

  ns.updateTodosPanel = updateTodosPanel;
  ns.handleTodoToolEvent = handleTodoToolEvent;
  ns.hydrateTodosFromToolCards = hydrateTodosFromToolCards;
  ns.refreshTodosFromServer = refreshTodosFromServer;
})();
