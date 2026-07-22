// Global command palette: Ctrl/Cmd+K jump to pages / search pipelines, sessions, homelab.
(function () {
  var STORAGE_KEY = 'flowbot-command-palette-recent';
  var MAX_RECENT = 8;
  var SEARCH_URL = '/service/web/command-palette/search';
  var DEBOUNCE_MS = 150;

  var dialog;
  var input;
  var resultsEl;
  var pages = [];
  var flatItems = [];
  var activeIndex = 0;
  var debounceTimer = null;
  var abortCtrl = null;

  function $(id) {
    return document.getElementById(id);
  }

  function loadPages() {
    var el = $('command-palette-pages');
    if (!el || !el.textContent) {
      return [];
    }
    try {
      var parsed = JSON.parse(el.textContent);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }

  function readRecent() {
    try {
      var raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) {
        return [];
      }
      var parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }

  // Mirrors recordCommandPaletteRecent in internal/modules/web/command_palette.go.
  function recordRecent(item) {
    if (!item || !item.href) {
      return;
    }
    var next = [item];
    var existing = readRecent();
    for (var i = 0; i < existing.length; i++) {
      if (existing[i] && existing[i].href === item.href) {
        continue;
      }
      next.push(existing[i]);
      if (next.length >= MAX_RECENT) {
        break;
      }
    }
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    } catch {
      /* intentionally silent: private mode / quota */
    }
  }

  function matchLocal(needle, item) {
    if (!needle) {
      return true;
    }
    var title = (item.title || '').toLowerCase();
    var sub = (item.subtitle || '').toLowerCase();
    return title.indexOf(needle) !== -1 || sub.indexOf(needle) !== -1;
  }

  function filterPages(q) {
    var needle = (q || '').trim().toLowerCase();
    if (!needle) {
      return pages.slice();
    }
    var out = [];
    for (var i = 0; i < pages.length; i++) {
      if (matchLocal(needle, pages[i])) {
        out.push(pages[i]);
        if (out.length >= 8) {
          break;
        }
      }
    }
    return out;
  }

  function escapeHTML(s) {
    return String(s || '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function groupLabel(group) {
    if (group === 'pages') return 'Pages';
    if (group === 'pipelines') return 'Pipelines';
    if (group === 'sessions') return 'Sessions';
    if (group === 'homelab') return 'Homelab';
    if (group === 'recent') return 'Recent';
    return group || 'Results';
  }

  function flattenGroups(groups) {
    var items = [];
    for (var g = 0; g < groups.length; g++) {
      var block = groups[g];
      for (var i = 0; i < block.items.length; i++) {
        items.push(block.items[i]);
      }
    }
    return items;
  }

  function renderGroups(groups) {
    flatItems = flattenGroups(groups);
    if (!flatItems.length) {
      resultsEl.innerHTML =
        '<p class="flowbot-command-palette-empty" data-testid="command-palette-empty">No matching results</p>';
      activeIndex = 0;
      return;
    }
    if (activeIndex >= flatItems.length) {
      activeIndex = 0;
    }
    var html = '';
    var flatPos = 0;
    for (var g = 0; g < groups.length; g++) {
      var block = groups[g];
      if (!block.items.length) {
        continue;
      }
      html +=
        '<div class="flowbot-command-palette-group" data-testid="command-palette-group-' +
        escapeHTML(block.group) +
        '">' +
        escapeHTML(groupLabel(block.group)) +
        '</div>';
      for (var i = 0; i < block.items.length; i++) {
        var item = block.items[i];
        var selected = flatPos === activeIndex;
        // Section header already names the group; Recent shows original category.
        var meta = '';
        if (block.group === 'recent') {
          meta = groupLabel(item.sourceGroup || item.group || 'pages');
        }
        html +=
          '<button type="button" class="flowbot-command-palette-item" role="option" data-index="' +
          flatPos +
          '" data-testid="command-palette-item" aria-selected="' +
          (selected ? 'true' : 'false') +
          '">' +
          '<span class="min-w-0">' +
          '<span class="flowbot-command-palette-item-title truncate">' +
          escapeHTML(item.title) +
          '</span>' +
          (item.subtitle
            ? '<span class="flowbot-command-palette-item-sub truncate">' +
              escapeHTML(item.subtitle) +
              '</span>'
            : '') +
          '</span>' +
          (meta
            ? '<span class="flowbot-command-palette-item-meta">' +
              escapeHTML(meta) +
              '</span>'
            : '') +
          '</button>';
        flatPos++;
      }
    }
    resultsEl.innerHTML = html;
  }

  function showEmptyQuery() {
    var recent = readRecent().map(function (r) {
      var sourceGroup = r.group && r.group !== 'recent' ? r.group : 'pages';
      return {
        id: r.id || r.href,
        title: r.title || r.href,
        subtitle: r.subtitle || '',
        href: r.href,
        group: 'recent',
        sourceGroup: sourceGroup,
      };
    });
    renderGroups([
      { group: 'recent', items: recent },
      { group: 'pages', items: pages.slice(0, 8) },
    ]);
  }

  function showSearchResults(data, localPages) {
    renderGroups([
      { group: 'pages', items: localPages || [] },
      { group: 'pipelines', items: (data && data.pipelines) || [] },
      { group: 'sessions', items: (data && data.sessions) || [] },
      { group: 'homelab', items: (data && data.homelab) || [] },
    ]);
  }

  function runSearch(q) {
    var trimmed = (q || '').trim();
    if (!trimmed) {
      if (abortCtrl) {
        abortCtrl.abort();
        abortCtrl = null;
      }
      showEmptyQuery();
      return;
    }
    var localPages = filterPages(trimmed);
    if (abortCtrl) {
      abortCtrl.abort();
    }
    abortCtrl = new AbortController();
    var signal = abortCtrl.signal;
    fetch(SEARCH_URL + '?q=' + encodeURIComponent(trimmed), {
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
      signal: signal,
    })
      .then(function (res) {
        if (!res.ok) {
          throw new Error('search failed');
        }
        return res.json();
      })
      .then(function (data) {
        // Prefer server pages when present; fall back to local filter.
        var pageItems =
          data && Array.isArray(data.pages) && data.pages.length
            ? data.pages
            : localPages;
        showSearchResults(data, pageItems);
      })
      .catch(function (err) {
        if (err && err.name === 'AbortError') {
          return;
        }
        showSearchResults(
          { pipelines: [], sessions: [], homelab: [] },
          localPages,
        );
      });
  }

  function scheduleSearch() {
    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }
    debounceTimer = setTimeout(function () {
      debounceTimer = null;
      runSearch(input.value);
    }, DEBOUNCE_MS);
  }

  function openPalette() {
    if (!dialog) {
      return;
    }
    dialog.showModal();
    input.value = '';
    activeIndex = 0;
    showEmptyQuery();
    setTimeout(function () {
      input.focus();
      input.select();
    }, 0);
  }

  function closePalette() {
    if (!dialog || !dialog.open) {
      return;
    }
    dialog.close();
  }

  function selectIndex(idx) {
    if (!flatItems.length) {
      return;
    }
    if (idx < 0) {
      idx = flatItems.length - 1;
    }
    if (idx >= flatItems.length) {
      idx = 0;
    }
    activeIndex = idx;
    var buttons = resultsEl.querySelectorAll('.flowbot-command-palette-item');
    for (var i = 0; i < buttons.length; i++) {
      var on = i === activeIndex;
      buttons[i].setAttribute('aria-selected', on ? 'true' : 'false');
      if (on) {
        buttons[i].scrollIntoView({ block: 'nearest' });
      }
    }
  }

  function goToItem(item) {
    if (!item || !item.href) {
      return;
    }
    recordRecent({
      id: item.id,
      title: item.title,
      subtitle: item.subtitle || '',
      href: item.href,
      group: item.group || 'pages',
    });
    closePalette();
    window.location.assign(item.href);
  }

  function onDocumentKeydown(evt) {
    var key = evt.key;
    if ((evt.metaKey || evt.ctrlKey) && (key === 'k' || key === 'K')) {
      evt.preventDefault();
      if (dialog && dialog.open) {
        closePalette();
      } else {
        openPalette();
      }
      return;
    }
    if (!dialog || !dialog.open) {
      return;
    }
    if (key === 'Escape') {
      evt.preventDefault();
      closePalette();
      return;
    }
    if (key === 'ArrowDown') {
      evt.preventDefault();
      selectIndex(activeIndex + 1);
      return;
    }
    if (key === 'ArrowUp') {
      evt.preventDefault();
      selectIndex(activeIndex - 1);
      return;
    }
    if (key === 'Enter') {
      evt.preventDefault();
      if (flatItems[activeIndex]) {
        goToItem(flatItems[activeIndex]);
      }
    }
  }

  function updateHint() {
    var hint = document.querySelector(
      '[data-testid="nav-command-palette-hint"]',
    );
    if (!hint) {
      return;
    }
    var isMac =
      typeof navigator !== 'undefined' &&
      /Mac|iPhone|iPad|iPod/.test(navigator.platform || '');
    hint.textContent = isMac ? '⌘K' : 'Ctrl K';
  }

  function init() {
    dialog = $('command-palette');
    input = $('command-palette-input');
    resultsEl = $('command-palette-results');
    if (!dialog || !input || !resultsEl) {
      return;
    }
    pages = loadPages();
    updateHint();

    var openBtn = $('nav-command-palette');
    if (openBtn) {
      openBtn.addEventListener('click', function (evt) {
        evt.preventDefault();
        openPalette();
      });
    }

    input.addEventListener('input', function () {
      activeIndex = 0;
      scheduleSearch();
    });

    resultsEl.addEventListener('click', function (evt) {
      var btn =
        evt.target && evt.target.closest
          ? evt.target.closest('.flowbot-command-palette-item')
          : null;
      if (!btn) {
        return;
      }
      var idx = parseInt(btn.getAttribute('data-index'), 10);
      if (!isNaN(idx) && flatItems[idx]) {
        goToItem(flatItems[idx]);
      }
    });

    resultsEl.addEventListener('mousemove', function (evt) {
      var btn =
        evt.target && evt.target.closest
          ? evt.target.closest('.flowbot-command-palette-item')
          : null;
      if (!btn) {
        return;
      }
      var idx = parseInt(btn.getAttribute('data-index'), 10);
      if (!isNaN(idx) && idx !== activeIndex) {
        selectIndex(idx);
      }
    });

    document.addEventListener('keydown', onDocumentKeydown);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
