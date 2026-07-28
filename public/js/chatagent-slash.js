(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var SKILL_CHIP_CLASS = 'chatagent-skill-chip';
  var PICKER_CLASS = 'chatagent-slash-picker';
  var SLASH_RE = /(^|[\s\n])\/([a-zA-Z0-9_-]*)$/;
  var CHIP_TRAILER_RE = /^[\u00a0 \u200B]+$/;
  var skillsCache = null;
  var skillsPromise = null;
  var pickerByInput = new WeakMap();

  function isRichInput(el) {
    return !!(
      el &&
      el.getAttribute &&
      el.getAttribute('contenteditable') != null
    );
  }

  function isSkillChip(node) {
    return !!(
      node &&
      node.nodeType === 1 &&
      node.classList &&
      node.classList.contains(SKILL_CHIP_CLASS)
    );
  }

  function serializeNode(node) {
    if (!node) {
      return '';
    }
    if (node.nodeType === 3) {
      return node.nodeValue || '';
    }
    if (isSkillChip(node)) {
      var name = node.getAttribute('data-skill') || '';
      return name ? '/' + name : node.textContent || '';
    }
    if (node.nodeName === 'BR') {
      return '\n';
    }
    var out = '';
    var child = node.firstChild;
    while (child) {
      out += serializeNode(child);
      child = child.nextSibling;
    }
    if (node.nodeName === 'DIV' || node.nodeName === 'P') {
      if (out && out.charAt(out.length - 1) !== '\n') {
        out += '\n';
      }
    }
    return out;
  }

  ns.getInputText = function (el) {
    if (!el) {
      return '';
    }
    if (!isRichInput(el)) {
      return el.value || '';
    }
    return serializeNode(el)
      .replace(/\u200B/g, '')
      .replace(/\u00a0/g, ' ')
      .replace(/\n+$/, '');
  };

  ns.clearInput = function (el) {
    if (!el) {
      return;
    }
    if (!isRichInput(el)) {
      el.value = '';
      return;
    }
    el.textContent = '';
    syncEmptyClass(el);
    hidePickerFor(el);
  };

  ns.setInputDisabled = function (el, disabled) {
    if (!el) {
      return;
    }
    if (isRichInput(el)) {
      el.contentEditable = disabled ? 'false' : 'true';
      el.setAttribute('aria-disabled', disabled ? 'true' : 'false');
      el.classList.toggle('is-disabled', !!disabled);
      return;
    }
    el.disabled = !!disabled;
  };

  function richShell(el) {
    return el &&
      el.parentElement &&
      el.parentElement.classList.contains('chatagent-rich-shell')
      ? el.parentElement
      : null;
  }

  function isInputVisuallyEmpty(el) {
    if (!el) {
      return true;
    }
    if (el.querySelector('.' + SKILL_CHIP_CLASS)) {
      return false;
    }
    return !ns.getInputText(el).trim();
  }

  function syncEmptyClass(el) {
    if (!el || !isRichInput(el)) {
      return;
    }
    var empty = isInputVisuallyEmpty(el);
    var shell = richShell(el);
    if (shell) {
      shell.classList.toggle('is-empty', empty);
      var ph = shell.querySelector('.chatagent-rich-placeholder');
      if (ph) {
        // Prefer the HTML hidden attribute so placeholder cannot linger over chips.
        ph.hidden = !empty;
        ph.classList.toggle('hidden', !empty);
      }
    }
    el.classList.toggle('is-empty', empty);
  }

  function createSkillChip(name) {
    var chip = document.createElement('span');
    chip.className = SKILL_CHIP_CLASS;
    chip.contentEditable = 'false';
    chip.setAttribute('data-skill', name);
    chip.setAttribute('data-testid', 'chatagent-skill-chip');
    chip.setAttribute('title', name);
    chip.textContent = '/' + name;
    return chip;
  }

  function loadSkills(url) {
    if (!url) {
      return Promise.resolve([]);
    }
    if (skillsCache) {
      return Promise.resolve(skillsCache);
    }
    if (skillsPromise) {
      return skillsPromise;
    }
    skillsPromise = fetch(url, { credentials: 'same-origin' })
      .then(function (res) {
        if (!res.ok) {
          throw new Error('skills http ' + res.status);
        }
        return res.json();
      })
      .then(function (json) {
        var data = (json && json.data) || {};
        var list = Array.isArray(data.skills) ? data.skills : [];
        skillsCache = list
          .map(function (s) {
            return {
              name: String((s && s.name) || '').trim(),
              description: String((s && s.description) || '').trim(),
            };
          })
          .filter(function (s) {
            return !!s.name;
          });
        return skillsCache;
      })
      .catch(function () {
        // intentionally silent: picker shows empty state when skills API is unavailable
        skillsPromise = null;
        return [];
      });
    return skillsPromise;
  }

  function ensurePicker(input) {
    var state = pickerByInput.get(input);
    if (state) {
      return state;
    }
    var composer = input.closest('.agents-composer');
    var composerBox = input.closest('.agents-composer-box');
    var threadWrap = input.closest('.chatagent-input-wrap');
    var picker = document.createElement('div');
    picker.className = PICKER_CLASS + ' hidden';
    picker.setAttribute('data-testid', 'chatagent-slash-picker');
    picker.setAttribute('role', 'listbox');
    picker.setAttribute('aria-label', 'Skills');
    // Keep picker out of the contenteditable tree so typed "/" cannot show through.
    if (composer || composerBox) {
      // Float below the whole composer card (page top has no room above).
      picker.classList.add('chatagent-slash-picker-below');
      (composer || composerBox).appendChild(picker);
    } else if (threadWrap) {
      // Thread sits at the bottom — float the menu above the follow-up field.
      threadWrap.appendChild(picker);
    } else if (input.parentElement) {
      input.parentElement.appendChild(picker);
    }
    state = {
      picker: picker,
      items: [],
      active: 0,
      open: false,
      query: '',
      range: null,
    };
    pickerByInput.set(input, state);
    return state;
  }

  function hidePickerFor(input) {
    var state = pickerByInput.get(input);
    if (!state) {
      return;
    }
    state.open = false;
    state.items = [];
    state.active = 0;
    state.query = '';
    state.range = null;
    state.picker.classList.add('hidden');
    state.picker.textContent = '';
  }

  function filterSkills(skills, query) {
    var q = (query || '').toLowerCase();
    var out = [];
    for (var i = 0; i < skills.length; i++) {
      var s = skills[i];
      if (
        !q ||
        s.name.toLowerCase().indexOf(q) >= 0 ||
        s.description.toLowerCase().indexOf(q) >= 0
      ) {
        out.push(s);
        if (out.length >= 8) {
          break;
        }
      }
    }
    return out;
  }

  function renderPicker(state) {
    state.picker.textContent = '';
    if (!state.items.length) {
      var empty = document.createElement('div');
      empty.className = 'chatagent-slash-empty';
      empty.textContent = state.query
        ? 'No matching skills'
        : 'No skills available';
      state.picker.appendChild(empty);
      state.picker.classList.remove('hidden');
      state.open = true;
      return;
    }
    for (var i = 0; i < state.items.length; i++) {
      (function (idx) {
        var skill = state.items[idx];
        var btn = document.createElement('button');
        btn.type = 'button';
        btn.className =
          'chatagent-slash-item' + (idx === state.active ? ' is-active' : '');
        btn.setAttribute('role', 'option');
        btn.setAttribute('data-testid', 'chatagent-slash-item');
        btn.setAttribute('data-skill', skill.name);
        var nameEl = document.createElement('span');
        nameEl.className = 'chatagent-slash-item-name';
        nameEl.textContent = '/' + skill.name;
        btn.appendChild(nameEl);
        if (skill.description) {
          var desc = document.createElement('span');
          desc.className = 'chatagent-slash-item-desc';
          desc.textContent = skill.description;
          btn.appendChild(desc);
        }
        btn.addEventListener('mousedown', function (ev) {
          ev.preventDefault();
        });
        btn.addEventListener('click', function () {
          if (typeof state.onSelect === 'function') {
            state.onSelect(skill);
          }
        });
        state.picker.appendChild(btn);
      })(i);
    }
    state.picker.classList.remove('hidden');
    state.open = true;
  }

  function setActive(state, idx) {
    if (!state.items.length) {
      return;
    }
    state.active =
      ((idx % state.items.length) + state.items.length) % state.items.length;
    var kids = state.picker.querySelectorAll('.chatagent-slash-item');
    for (var i = 0; i < kids.length; i++) {
      kids[i].classList.toggle('is-active', i === state.active);
    }
    if (kids[state.active] && kids[state.active].scrollIntoView) {
      kids[state.active].scrollIntoView({ block: 'nearest' });
    }
  }

  function getCaretTextInfo(input) {
    var sel = window.getSelection();
    if (!sel || !sel.rangeCount || !sel.isCollapsed) {
      return null;
    }
    var range = sel.getRangeAt(0);
    if (!input.contains(range.startContainer)) {
      return null;
    }
    var pre = range.cloneRange();
    pre.selectNodeContents(input);
    pre.setEnd(range.startContainer, range.startOffset);
    var before = pre.toString().replace(/\u200B/g, '');
    return { before: before, range: range };
  }

  function deleteSlashQuery(input, matchLen) {
    var sel = window.getSelection();
    if (!sel || !sel.rangeCount) {
      return null;
    }
    var range = sel.getRangeAt(0);
    var node = range.startContainer;
    var offset = range.startOffset;
    if (node.nodeType !== 3) {
      return range;
    }
    var value = node.nodeValue || '';
    var start = offset - matchLen;
    if (start < 0) {
      return range;
    }
    node.nodeValue = value.slice(0, start) + value.slice(offset);
    var next = document.createRange();
    next.setStart(node, start);
    next.collapse(true);
    sel.removeAllRanges();
    sel.addRange(next);
    return next;
  }

  function placeCaret(node, before) {
    var sel = window.getSelection();
    if (!sel || !node || !node.parentNode) {
      return;
    }
    var r = document.createRange();
    if (before) {
      r.setStartBefore(node);
    } else {
      r.setStartAfter(node);
    }
    r.collapse(true);
    sel.removeAllRanges();
    sel.addRange(r);
  }

  function removeChipUnit(chip) {
    if (!chip || !chip.parentNode) {
      return null;
    }
    var next = chip.nextSibling;
    var parent = chip.parentNode;
    parent.removeChild(chip);
    if (
      next &&
      next.nodeType === 3 &&
      CHIP_TRAILER_RE.test(next.nodeValue || '')
    ) {
      var afterTrailer = next.nextSibling;
      parent.removeChild(next);
      return afterTrailer;
    }
    return next;
  }

  function insertChipAtCaret(input, skill, matchLen) {
    var range = deleteSlashQuery(input, matchLen);
    if (!range) {
      return;
    }
    var chip = createSkillChip(skill.name);
    // Trailing space keeps typing easy; backspace treats chip+trailer as one unit.
    var space = document.createTextNode('\u00a0');
    range.insertNode(space);
    range.insertNode(chip);
    placeCaret(space, false);
    hidePickerFor(input);
    input.focus();
    syncEmptyClass(input);
    // Re-sync after layout so chip presence is never missed for placeholder hide.
    window.requestAnimationFrame(function () {
      syncEmptyClass(input);
    });
  }

  function removeChipBeforeCaret(input) {
    var sel = window.getSelection();
    if (!sel || !sel.rangeCount || !sel.isCollapsed) {
      return false;
    }
    var range = sel.getRangeAt(0);
    var container = range.startContainer;
    var offset = range.startOffset;

    if (container === input && offset > 0) {
      var prev = input.childNodes[offset - 1];
      if (
        prev &&
        prev.nodeType === 3 &&
        CHIP_TRAILER_RE.test(prev.nodeValue || '') &&
        isSkillChip(prev.previousSibling)
      ) {
        var afterTrailer = removeChipUnit(prev.previousSibling);
        if (afterTrailer) {
          placeCaret(afterTrailer, true);
        } else {
          var end = document.createRange();
          end.selectNodeContents(input);
          end.collapse(false);
          sel.removeAllRanges();
          sel.addRange(end);
        }
        syncEmptyClass(input);
        return true;
      }
      if (isSkillChip(prev)) {
        var afterChip = removeChipUnit(prev);
        if (afterChip) {
          placeCaret(afterChip, true);
        } else {
          var end2 = document.createRange();
          end2.selectNodeContents(input);
          end2.collapse(false);
          sel.removeAllRanges();
          sel.addRange(end2);
        }
        syncEmptyClass(input);
        return true;
      }
    }

    if (container.nodeType === 3) {
      var value = container.nodeValue || '';
      var before = value.slice(0, offset);
      if (
        CHIP_TRAILER_RE.test(before) &&
        isSkillChip(container.previousSibling)
      ) {
        var after = removeChipUnit(container.previousSibling);
        if (after) {
          placeCaret(after, true);
        } else if (container.parentNode) {
          var end3 = document.createRange();
          end3.selectNodeContents(input);
          end3.collapse(false);
          sel.removeAllRanges();
          sel.addRange(end3);
        }
        syncEmptyClass(input);
        return true;
      }
      if (offset === 0 && isSkillChip(container.previousSibling)) {
        removeChipUnit(container.previousSibling);
        syncEmptyClass(input);
        return true;
      }
    }

    if (isSkillChip(container)) {
      var afterSel = removeChipUnit(container);
      if (afterSel) {
        placeCaret(afterSel, true);
      } else {
        var end4 = document.createRange();
        end4.selectNodeContents(input);
        end4.collapse(false);
        sel.removeAllRanges();
        sel.addRange(end4);
      }
      syncEmptyClass(input);
      return true;
    }

    return false;
  }

  function removeSelectedChips(input) {
    var sel = window.getSelection();
    if (!sel || !sel.rangeCount || sel.isCollapsed) {
      return false;
    }
    var chips = input.querySelectorAll('.' + SKILL_CHIP_CLASS);
    var removed = false;
    for (var i = 0; i < chips.length; i++) {
      if (sel.containsNode(chips[i], true)) {
        removeChipUnit(chips[i]);
        removed = true;
      }
    }
    if (removed) {
      syncEmptyClass(input);
    }
    return removed;
  }

  function refreshSlashState(input, skillsURL) {
    var info = getCaretTextInfo(input);
    if (!info) {
      hidePickerFor(input);
      return;
    }
    var m = info.before.match(SLASH_RE);
    if (!m) {
      hidePickerFor(input);
      return;
    }
    var query = m[2] || '';
    var matchLen = 1 + query.length;
    var state = ensurePicker(input);
    state.query = query;
    state.onSelect = function (skill) {
      insertChipAtCaret(input, skill, matchLen);
    };
    loadSkills(skillsURL).then(function (skills) {
      if (!pickerByInput.get(input)) {
        return;
      }
      // Caret may have moved; re-check.
      var again = getCaretTextInfo(input);
      if (!again || !again.before.match(SLASH_RE)) {
        hidePickerFor(input);
        return;
      }
      var m2 = again.before.match(SLASH_RE);
      state.query = (m2 && m2[2]) || '';
      state.items = filterSkills(skills, state.query);
      state.active = 0;
      state.onSelect = function (skill) {
        var cur = getCaretTextInfo(input);
        var mm = cur && cur.before.match(SLASH_RE);
        var len = mm ? 1 + (mm[2] || '').length : matchLen;
        insertChipAtCaret(input, skill, len);
      };
      renderPicker(state);
    });
  }

  ns.slashPickerOpen = function (input) {
    var state = pickerByInput.get(input);
    return !!(state && state.open);
  };

  ns.slashPickerHandlesKey = function (input, ev) {
    var state = pickerByInput.get(input);
    if (!state || !state.open) {
      return false;
    }
    if (ev.key === 'Escape') {
      ev.preventDefault();
      hidePickerFor(input);
      return true;
    }
    if (ev.key === 'ArrowDown') {
      ev.preventDefault();
      setActive(state, state.active + 1);
      return true;
    }
    if (ev.key === 'ArrowUp') {
      ev.preventDefault();
      setActive(state, state.active - 1);
      return true;
    }
    if (ev.key === 'Enter' || ev.key === 'Tab') {
      if (!state.items.length) {
        hidePickerFor(input);
        return ev.key === 'Tab';
      }
      ev.preventDefault();
      var skill = state.items[state.active];
      if (skill && typeof state.onSelect === 'function') {
        state.onSelect(skill);
      }
      return true;
    }
    return false;
  };

  ns.initSlashSkills = function (root, input) {
    if (!root || !input || !isRichInput(input)) {
      return;
    }
    var skillsURL = root.getAttribute('data-skills-url') || '';
    // Clear any materialized placeholder text left by older ::before implementations.
    var shell = richShell(input);
    var placeholderEl = shell
      ? shell.querySelector('.chatagent-rich-placeholder')
      : null;
    var placeholderText = placeholderEl
      ? (placeholderEl.textContent || '').trim()
      : '';
    if (placeholderText && ns.getInputText(input).trim() === placeholderText) {
      input.textContent = '';
    }
    syncEmptyClass(input);
    if (skillsURL) {
      loadSkills(skillsURL);
    }

    input.addEventListener('input', function () {
      syncEmptyClass(input);
      refreshSlashState(input, skillsURL);
    });

    input.addEventListener('keyup', function (ev) {
      if (
        ev.key === 'Escape' ||
        ev.key === 'Enter' ||
        ev.key === 'ArrowUp' ||
        ev.key === 'ArrowDown' ||
        ev.key === 'Tab'
      ) {
        return;
      }
      refreshSlashState(input, skillsURL);
    });

    input.addEventListener('keydown', function (ev) {
      if (ns.slashPickerHandlesKey(input, ev)) {
        // Prevent chat.js Enter-to-send and contenteditable caret moves.
        ev.stopImmediatePropagation();
        return;
      }
      if (ev.key === 'Backspace') {
        if (removeSelectedChips(input) || removeChipBeforeCaret(input)) {
          ev.preventDefault();
          hidePickerFor(input);
          syncEmptyClass(input);
        }
      }
    });

    input.addEventListener('blur', function () {
      // Delay so picker click can run first.
      window.setTimeout(function () {
        if (document.activeElement === input) {
          return;
        }
        hidePickerFor(input);
      }, 150);
    });

    input.addEventListener('paste', function (ev) {
      var items = (ev.clipboardData && ev.clipboardData.items) || [];
      for (var i = 0; i < items.length; i++) {
        if (items[i].type && items[i].type.indexOf('image/') === 0) {
          // Attachment queue handles image paste.
          return;
        }
      }
      var text =
        (ev.clipboardData && ev.clipboardData.getData('text/plain')) || '';
      if (!text) {
        return;
      }
      ev.preventDefault();
      if (
        document.queryCommandSupported &&
        document.queryCommandSupported('insertText')
      ) {
        document.execCommand('insertText', false, text);
      } else {
        var sel = window.getSelection();
        if (!sel || !sel.rangeCount) {
          return;
        }
        var range = sel.getRangeAt(0);
        range.deleteContents();
        var node = document.createTextNode(text);
        range.insertNode(node);
        range.setStartAfter(node);
        range.collapse(true);
        sel.removeAllRanges();
        sel.addRange(range);
      }
      syncEmptyClass(input);
      refreshSlashState(input, skillsURL);
    });
  };
})();
