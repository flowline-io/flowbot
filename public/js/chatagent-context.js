(function () {
  'use strict';

  var ns = (window.FlowbotChatAgent = window.FlowbotChatAgent || {});

  var RING_CIRCUMFERENCE = 62.832;

  var CONTEXT_CATEGORY_LABELS = {
    system_prompt: 'System prompt',
    system_tools: 'Tool definitions',
    skills: 'Skills',
    messages: 'Conversation',
    autocompact_buffer: 'Autocompact buffer',
    free_space: 'Free space',
  };

  var CONTEXT_CATEGORY_COLORS = {
    system_prompt: '#9ca3af',
    system_tools: '#a78bfa',
    skills: '#fb923c',
    messages: '#b4534a',
    autocompact_buffer: '#64748b',
    free_space: '#1f2937',
  };

  var CONTEXT_BAR_ORDER = [
    'system_prompt',
    'system_tools',
    'skills',
    'messages',
    'autocompact_buffer',
    'free_space',
  ];

  var contextControls = new WeakMap();

  function formatTokenCount(n) {
    var value = Number(n) || 0;
    if (value >= 1000) {
      var scaled = value / 1000;
      if (scaled >= 100) {
        return Math.round(scaled) + 'K';
      }
      return scaled.toFixed(1).replace(/\.0$/, '') + 'K';
    }
    return String(Math.round(value));
  }

  function formatTokenRange(used, windowSize) {
    return (
      '~' +
      formatTokenCount(used) +
      ' / ' +
      formatTokenCount(windowSize) +
      ' Tokens'
    );
  }

  function contextUsagePercent(total, windowSize, reported) {
    if (windowSize > 0 && total > 0) {
      return (total / windowSize) * 100;
    }
    if (typeof reported === 'number' && !Number.isNaN(reported)) {
      return reported;
    }
    return 0;
  }

  function updateContextRing(ringWrap, percent) {
    if (!ringWrap) {
      return;
    }
    var progress = ringWrap.querySelector('.chatagent-context-ring-progress');
    if (!progress) {
      return;
    }
    var clamped = Math.max(0, Math.min(100, percent || 0));
    progress.setAttribute(
      'stroke-dashoffset',
      String(RING_CIRCUMFERENCE * (1 - clamped / 100)),
    );
  }

  function updateContextSummary(popover, report) {
    if (!popover || !report) {
      return;
    }
    var percentEl = popover.querySelector('#chatagent-context-percent');
    var tokensEl = popover.querySelector('#chatagent-context-tokens');
    var pct = Math.round(report.total_percent || 0);
    if (percentEl) {
      percentEl.textContent = pct + '% Full';
    }
    if (tokensEl) {
      tokensEl.textContent = formatTokenRange(
        report.total_tokens || 0,
        report.context_window || 0,
      );
    }
  }

  function renderContextUsage(popover, report) {
    if (!popover || !report) {
      return;
    }
    updateContextSummary(popover, report);

    var barEl = popover.querySelector('#chatagent-context-bar');
    var legendEl = popover.querySelector('#chatagent-context-legend');
    if (!barEl || !legendEl) {
      return;
    }

    var byID = {};
    (report.categories || []).forEach(function (cat) {
      byID[cat.id] = cat;
    });

    barEl.textContent = '';
    CONTEXT_BAR_ORDER.forEach(function (id) {
      var cat = byID[id];
      if (!cat || cat.percent <= 0) {
        return;
      }
      var seg = document.createElement('div');
      seg.className = 'chatagent-context-bar-segment';
      seg.setAttribute('data-category', id);
      seg.style.width = cat.percent + '%';
      seg.style.backgroundColor = CONTEXT_CATEGORY_COLORS[id] || '#6b7280';
      barEl.appendChild(seg);
    });

    legendEl.textContent = '';
    CONTEXT_BAR_ORDER.forEach(function (id) {
      if (id === 'free_space') {
        return;
      }
      var cat = byID[id];
      if (!cat) {
        return;
      }
      var row = document.createElement('div');
      row.className = 'chatagent-context-legend-row';

      var labelWrap = document.createElement('div');
      labelWrap.className = 'chatagent-context-legend-label';

      var swatch = document.createElement('span');
      swatch.className = 'chatagent-context-swatch';
      swatch.style.backgroundColor = CONTEXT_CATEGORY_COLORS[id] || '#6b7280';

      var label = document.createElement('span');
      label.textContent = CONTEXT_CATEGORY_LABELS[id] || cat.label || id;

      labelWrap.appendChild(swatch);
      labelWrap.appendChild(label);

      var tokens = document.createElement('span');
      tokens.className = 'chatagent-context-legend-tokens';
      tokens.textContent = formatTokenCount(cat.tokens);

      row.appendChild(labelWrap);
      row.appendChild(tokens);
      legendEl.appendChild(row);

      if (id === 'skills' && report.skills && report.skills.length > 0) {
        report.skills.forEach(function (skill) {
          var skillRow = document.createElement('div');
          skillRow.className =
            'chatagent-context-legend-row chatagent-context-skill-row';

          var skillLabelWrap = document.createElement('div');
          skillLabelWrap.className = 'chatagent-context-legend-label';

          var skillLabel = document.createElement('span');
          skillLabel.textContent = skill.name || 'skill';
          skillLabelWrap.appendChild(skillLabel);

          var skillTokens = document.createElement('span');
          skillTokens.className = 'chatagent-context-legend-tokens';
          skillTokens.textContent = formatTokenCount(skill.tokens);

          skillRow.appendChild(skillLabelWrap);
          skillRow.appendChild(skillTokens);
          legendEl.appendChild(skillRow);
        });
      }
    });
  }

  ns.initContextControl = function(threadRoot) {
    var contextURL = threadRoot.getAttribute('data-context-url') || '';
    var ringWrap = threadRoot.querySelector('.chatagent-context-ring-wrap');
    var popover = threadRoot.querySelector('.chatagent-context-popover');
    if (!contextURL || !ringWrap || !popover) {
      return null;
    }

    var ringBtn = ringWrap.querySelector('.chatagent-context-ring');
    var closeBtn = popover.querySelector('.chatagent-context-close');
    var errorEl = popover.querySelector('#chatagent-context-error');
    var cachedReport = null;
    var open = false;
    var outsideListener = null;

    function showPopoverError(message) {
      ns.showError(errorEl, message);
    }

    function setOpen(next) {
      open = next;
      if (!popover) {
        return;
      }
      popover.classList.toggle('hidden', !open);
      if (outsideListener) {
        document.removeEventListener('click', outsideListener);
        outsideListener = null;
      }
      if (open) {
        outsideListener = function (event) {
          if (
            ringWrap.contains(event.target) ||
            popover.contains(event.target)
          ) {
            return;
          }
          setOpen(false);
        };
        window.setTimeout(function () {
          if (open && outsideListener) {
            document.addEventListener('click', outsideListener);
          }
        }, 0);
      }
    }

    function applyReport(report) {
      cachedReport = report;
      updateContextRing(ringWrap, report.total_percent || 0);
      if (open) {
        renderContextUsage(popover, report);
      }
    }

    function fetchContextReport(forceRender) {
      return fetch(contextURL, { headers: { Accept: 'application/json' } })
        .then(function (res) {
          if (!res.ok) {
            return res
              .json()
              .catch(function () {
                return {};
              })
              .then(function (data) {
                throw new Error(
                  (data && data.error) || 'Failed to load context usage',
                );
              });
          }
          return res.json();
        })
        .then(function (report) {
          showPopoverError('');
          applyReport(report);
          if (forceRender && open) {
            renderContextUsage(popover, report);
          }
          return report;
        })
        .catch(function (err) {
          if (open) {
            showPopoverError(err.message || 'Failed to load context usage');
          }
          return null;
        });
    }

    function togglePopover() {
      if (open) {
        setOpen(false);
        return;
      }
      setOpen(true);
      showPopoverError('');
      if (cachedReport) {
        renderContextUsage(popover, cachedReport);
        return;
      }
      fetchContextReport(true);
    }

    if (ringBtn) {
      ringBtn.addEventListener('click', function (event) {
        event.stopPropagation();
        togglePopover();
      });
    }
    if (closeBtn) {
      closeBtn.addEventListener('click', function (event) {
        event.stopPropagation();
        setOpen(false);
      });
    }

    fetchContextReport(false);

    var control = {
      handleUsage: function (ev) {
        var pct = contextUsagePercent(
          ev.total_tokens,
          ev.context_window,
          ev.context_percent,
        );
        updateContextRing(ringWrap, pct);
        if (open && popover) {
          var summary = cachedReport
            ? Object.assign({}, cachedReport)
            : {
                total_tokens: ev.total_tokens || 0,
                context_window: ev.context_window || 0,
                total_percent: pct,
                categories: [],
                skills: [],
              };
          summary.total_tokens = ev.total_tokens || summary.total_tokens;
          summary.context_window = ev.context_window || summary.context_window;
          summary.total_percent = pct;
          updateContextSummary(popover, summary);
        }
      },
      onRunComplete: function () {
        if (open) {
          fetchContextReport(true);
        } else {
          fetchContextReport(false);
        }
      },
    };

    contextControls.set(threadRoot, control);
    return control;
  }

  ns.getContextControl = function(threadRoot) {
    return contextControls.get(threadRoot) || null;
  }
})();
