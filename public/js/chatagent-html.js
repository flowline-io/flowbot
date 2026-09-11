(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  function i18n(key, fallback) {
    if (typeof window.flowbotI18n === 'function') {
      return window.flowbotI18n(key, fallback);
    }
    return fallback;
  }

  function createSandboxedFrame(doc, title) {
    var iframe = document.createElement('iframe');
    iframe.className = 'chatagent-html-frame';
    iframe.setAttribute('sandbox', 'allow-scripts');
    iframe.setAttribute('referrerpolicy', 'no-referrer');
    iframe.setAttribute('allow', '');
    iframe.setAttribute('data-testid', 'chatagent-html-frame');
    if (title) {
      iframe.title = title;
    }
    iframe.srcdoc = doc || '';
    return iframe;
  }

  function setTab(root, tab) {
    if (!root) {
      return;
    }
    var buttons = root.querySelectorAll('[data-html-tab]');
    buttons.forEach(function (btn) {
      btn.classList.toggle('is-active', btn.getAttribute('data-html-tab') === tab);
    });
    var preview = root.querySelector('[data-html-pane="preview"]');
    var source = root.querySelector('[data-html-pane="source"]');
    if (preview) {
      preview.classList.toggle('hidden', tab !== 'preview');
    }
    if (source) {
      source.classList.toggle('hidden', tab !== 'source');
    }
  }

  function bindArtifact(root) {
    if (!root || root.getAttribute('data-html-bound') === '1') {
      return;
    }
    root.setAttribute('data-html-bound', '1');
    root.addEventListener('click', function (ev) {
      var tabBtn = ev.target.closest('[data-html-tab]');
      if (tabBtn && root.contains(tabBtn)) {
        setTab(root, tabBtn.getAttribute('data-html-tab'));
        return;
      }
      if (ev.target.closest('[data-html-expand]') && root.contains(ev.target)) {
        openOverlay(root);
      }
    });
  }

  function openOverlay(root) {
    var dialog = document.getElementById('chatagent-html-overlay');
    var frame = document.getElementById('chatagent-html-overlay-frame');
    var source = document.getElementById('chatagent-html-overlay-source');
    var titleEl = document.getElementById('chatagent-html-overlay-title');
    if (!dialog || !frame) {
      return;
    }
    var title =
      (root.querySelector('[data-testid="chatagent-html-title"]') || {})
        .textContent || 'HTML';
    var doc = '';
    var sourceEl = root.querySelector('[data-testid="chatagent-html-source"]');
    doc = sourceEl ? sourceEl.textContent : '';
    if (titleEl) {
      titleEl.textContent = title;
    }
    frame.setAttribute('sandbox', 'allow-scripts');
    frame.setAttribute('referrerpolicy', 'no-referrer');
    frame.setAttribute('allow', '');
    frame.srcdoc = doc;
    if (source) {
      source.textContent = doc;
      source.classList.add('hidden');
    }
    frame.classList.remove('hidden');
    setOverlayTab('preview');
    if (typeof dialog.showModal === 'function') {
      dialog.showModal();
    }
  }

  function setOverlayTab(tab) {
    var dialog = document.getElementById('chatagent-html-overlay');
    if (!dialog) {
      return;
    }
    dialog.querySelectorAll('[data-html-overlay-tab]').forEach(function (btn) {
      btn.classList.toggle(
        'is-active',
        btn.getAttribute('data-html-overlay-tab') === tab,
      );
    });
    var frame = document.getElementById('chatagent-html-overlay-frame');
    var source = document.getElementById('chatagent-html-overlay-source');
    if (frame) {
      frame.classList.toggle('hidden', tab !== 'preview');
    }
    if (source) {
      source.classList.toggle('hidden', tab !== 'source');
    }
  }

  function bindOverlay() {
    var dialog = document.getElementById('chatagent-html-overlay');
    if (!dialog || dialog.getAttribute('data-html-bound') === '1') {
      return;
    }
    dialog.setAttribute('data-html-bound', '1');
    dialog.addEventListener('click', function (ev) {
      var tabBtn = ev.target.closest('[data-html-overlay-tab]');
      if (tabBtn) {
        setOverlayTab(tabBtn.getAttribute('data-html-overlay-tab'));
        return;
      }
      if (ev.target.closest('[data-html-overlay-close]')) {
        dialog.close();
      }
    });
  }

  ns.mountHtmlArtifact = function (card, ev) {
    if (!card || !card.wrap || !ev || !ev.html) {
      return;
    }
    var existing = card.wrap.querySelector('[data-html-artifact]');
    if (existing) {
      existing.remove();
    }
    var tpl = document.getElementById('chatagent-html-artifact-template');
    if (!tpl || !tpl.content || !tpl.content.firstElementChild) {
      return;
    }
    var title = ev.title || 'HTML';
    var root = tpl.content.firstElementChild.cloneNode(true);
    if (ev.artifact_id) {
      root.setAttribute('data-html-artifact-id', ev.artifact_id);
    }
    var titleEl = root.querySelector('[data-testid="chatagent-html-title"]');
    if (titleEl) {
      titleEl.textContent = title;
    }
    var pane = root.querySelector('[data-html-pane="preview"]');
    if (pane) {
      pane.appendChild(createSandboxedFrame(ev.html, title));
    }
    var source = root.querySelector('[data-testid="chatagent-html-source"]');
    if (source) {
      source.textContent = ev.html;
    }
    card.wrap.appendChild(root);
    bindArtifact(root);
    ns.refreshHtmlArtifactLive(card.wrap.parentElement);
  };

  function collapseSupersededArtifact(el) {
    el.setAttribute('data-html-artifact-superseded', '');
    var tabs = el.querySelector('.chatagent-html-tabs');
    if (tabs) {
      tabs.remove();
    }
    var expand = el.querySelector('[data-html-expand]');
    if (expand) {
      expand.remove();
    }
    el.querySelectorAll('[data-html-pane]').forEach(function (pane) {
      pane.remove();
    });
    if (el.querySelector('[data-testid="chatagent-html-superseded"]')) {
      return;
    }
    var note = document.createElement('p');
    note.className = 'chatagent-html-superseded';
    note.setAttribute('data-testid', 'chatagent-html-superseded');
    note.textContent = i18n(
      'client.chatagent.html.updated',
      'Updated — this version is no longer live.',
    );
    var toolbar = el.querySelector('.chatagent-html-toolbar');
    if (toolbar && toolbar.nextSibling) {
      el.insertBefore(note, toolbar.nextSibling);
    } else {
      el.appendChild(note);
    }
  }

  ns.refreshHtmlArtifactLive = function (container) {
    var root = container || document.getElementById('chatagent-messages');
    if (!root) {
      return;
    }
    var cards = root.querySelectorAll('[data-html-artifact]');
    var latest = {};
    cards.forEach(function (el, idx) {
      var id = el.getAttribute('data-html-artifact-id');
      if (id) {
        latest[id] = idx;
      }
    });
    cards.forEach(function (el, idx) {
      var id = el.getAttribute('data-html-artifact-id');
      if (!id) {
        return;
      }
      if (latest[id] !== idx) {
        collapseSupersededArtifact(el);
      }
    });
  };

  ns.bindHtmlArtifacts = function (container) {
    bindOverlay();
    var root = container || document.getElementById('chatagent-messages');
    if (!root) {
      return;
    }
    root.querySelectorAll('[data-html-artifact]').forEach(bindArtifact);
    ns.refreshHtmlArtifactLive(root);
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      ns.bindHtmlArtifacts();
    });
  } else {
    ns.bindHtmlArtifacts();
  }
})();
