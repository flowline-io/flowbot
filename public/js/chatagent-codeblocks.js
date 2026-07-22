(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var COLLAPSE_LINE_THRESHOLD = 18;

  var copyIconSVG =
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" class="w-3 h-3" aria-hidden="true"><path d="M5 6.5A1.5 1.5 0 0 1 6.5 5h6A1.5 1.5 0 0 1 14 6.5v6a1.5 1.5 0 0 1-1.5 1.5h-6A1.5 1.5 0 0 1 5 12.5v-6Z"></path><path d="M3.5 2A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11V6.5a3 3 0 0 1 3-3H11A1.5 1.5 0 0 0 9.5 2h-6Z"></path></svg>';

  function languageFromCode(codeEl) {
    var classes = (codeEl.className || '').split(/\s+/);
    for (var i = 0; i < classes.length; i++) {
      var cls = classes[i];
      if (cls.indexOf('language-') === 0) {
        return cls.slice('language-'.length) || 'text';
      }
    }
    return 'text';
  }

  function lineCount(text) {
    if (!text) {
      return 0;
    }
    return text.split('\n').length;
  }

  function enhanceOne(pre) {
    if (!pre || pre.closest('.chatagent-codeblock')) {
      return;
    }
    var code = pre.querySelector('code');
    if (!code) {
      return;
    }
    var parent = pre.parentNode;
    if (!parent) {
      return;
    }

    var wrap = document.createElement('div');
    wrap.className = 'chatagent-codeblock';
    wrap.setAttribute('data-testid', 'chatagent-codeblock');

    var header = document.createElement('div');
    header.className = 'chatagent-codeblock-header';

    var lang = document.createElement('span');
    lang.className = 'chatagent-codeblock-lang';
    lang.setAttribute('data-testid', 'chatagent-codeblock-lang');
    lang.textContent = languageFromCode(code);

    var actions = document.createElement('div');
    actions.className = 'chatagent-codeblock-actions';

    var copyBtn = document.createElement('button');
    copyBtn.type = 'button';
    copyBtn.className = 'btn btn-ghost btn-xs chatagent-codeblock-copy';
    copyBtn.title = 'Copy code';
    copyBtn.setAttribute('aria-label', 'Copy code');
    copyBtn.setAttribute('data-testid', 'chatagent-codeblock-copy');
    copyBtn.setAttribute('data-clip-copy', '');
    copyBtn.setAttribute('data-clip-markdown', code.textContent || '');
    copyBtn.innerHTML = copyIconSVG;

    header.appendChild(lang);
    actions.appendChild(copyBtn);
    header.appendChild(actions);

    parent.insertBefore(wrap, pre);
    wrap.appendChild(header);
    wrap.appendChild(pre);

    if (lineCount(code.textContent || '') >= COLLAPSE_LINE_THRESHOLD) {
      wrap.classList.add('is-collapsed');
      var toggle = document.createElement('button');
      toggle.type = 'button';
      toggle.className = 'chatagent-codeblock-toggle';
      toggle.setAttribute('data-testid', 'chatagent-codeblock-toggle');
      toggle.textContent = 'Show more';
      toggle.addEventListener('click', function () {
        var collapsed = wrap.classList.toggle('is-collapsed');
        toggle.textContent = collapsed ? 'Show more' : 'Show less';
      });
      wrap.appendChild(toggle);
    }
  }

  ns.enhanceCodeBlocks = function (root) {
    if (!root) {
      return;
    }
    var nodes = root.querySelectorAll(
      '.chatagent-markdown pre > code, .markdown-body pre > code',
    );
    for (var i = 0; i < nodes.length; i++) {
      enhanceOne(nodes[i].parentElement);
    }
  };
})();
