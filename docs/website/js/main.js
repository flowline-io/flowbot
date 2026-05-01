// ============================================================
// Flowbot Website — Interactive Effects
// ============================================================

(function () {
  'use strict';

  // --- Mobile Nav Toggle ---
  const navToggle = document.querySelector('.nav-toggle');
  const navLinks = document.querySelector('.nav-links');
  if (navToggle && navLinks) {
    navToggle.addEventListener('click', function () {
      navLinks.classList.toggle('open');
    });
  }

  // --- Active nav link based on scroll ---
  const sections = document.querySelectorAll('.section[id]');
  const navAnchors = document.querySelectorAll('.nav-links a[href^="#"]');

  function updateActiveLink() {
    let current = '';
    sections.forEach(function (sec) {
      const top = sec.offsetTop - 120;
      if (window.scrollY >= top) {
        current = sec.getAttribute('id');
      }
    });
    navAnchors.forEach(function (a) {
      a.classList.remove('active');
      if (a.getAttribute('href') === '#' + current) {
        a.classList.add('active');
      }
    });
  }

  if (sections.length && navAnchors.length) {
    window.addEventListener('scroll', updateActiveLink, { passive: true });
  }

  // Active link for subpages
  (function () {
    const path = window.location.pathname;
    document.querySelectorAll('.nav-links a[href]').forEach(function (link) {
      const href = link.getAttribute('href');
      if (href === path || (href !== '/' && href !== '/#' && path.indexOf(href) === 0)) {
        link.classList.add('active');
      }
    });
  })();

  // --- Terminal Typing Effect ---
  (function () {
    const terminal = document.getElementById('typing-terminal');
    if (!terminal) return;

    const lines = [
      { text: '$ flowbot scan', cls: 'cmd', delay: 800 },
      { text: 'Scanning /homelab/apps/...', cls: 'output', delay: 400 },
      { text: 'Found 12 apps: archivebox, atuin, beszel, karakeep, linkwarden, miniflux, kanboard, fireflyiii, transmission, uptimekuma, gitea, adguard', cls: 'output', delay: 700 },
      { text: '$ flowbot run pipeline backup', cls: 'cmd', delay: 600 },
      { text: '[pipeline:backup] DAG executed successfully', cls: 'output', delay: 500 },
      { text: '[pipeline:backup] 8/8 steps completed, 0 errors', cls: 'output', delay: 0 },
    ];

    const container = terminal.querySelector('.terminal-lines');
    const cursor = terminal.querySelector('.terminal-cursor');
    let lineIdx = 0;

    function typeLine(line, callback) {
      const div = document.createElement('div');
      div.className = 'terminal-line';
      container.appendChild(div);

      const span = document.createElement('span');
      span.className = line.cls;
      div.appendChild(span);

      let i = 0;
      const interval = setInterval(function () {
        span.textContent += line.text.charAt(i);
        i++;
        if (i >= line.text.length) {
          clearInterval(interval);
          if (callback) setTimeout(callback, line.delay);
        }
      }, 15);
    }

    function startTyping() {
      if (lineIdx >= lines.length) {
        if (cursor) cursor.style.animation = 'blink 1s step-end infinite';
        return;
      }
      typeLine(lines[lineIdx], function () {
        lineIdx++;
        startTyping();
      });
    }

    // Pause cursor blink during typing
    if (cursor) cursor.style.animation = 'none';
    setTimeout(startTyping, 600);
  })();

  // --- Background Node Graph (Canvas) ---
  (function () {
    const canvas = document.getElementById('bg-canvas');
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    let animFrame;

    function resize() {
      const rect = canvas.parentElement.getBoundingClientRect();
      canvas.width = rect.width * window.devicePixelRatio;
      canvas.height = rect.height * window.devicePixelRatio;
      canvas.style.width = rect.width + 'px';
      canvas.style.height = rect.height + 'px';
      ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
    }

    function draw() {
      const W = canvas.width / window.devicePixelRatio;
      const H = canvas.height / window.devicePixelRatio;

      ctx.clearRect(0, 0, W, H);

      const nodes = [];
      const cols = Math.floor(W / 120) + 1;
      const rows = Math.floor(H / 120) + 1;
      for (let r = 0; r < rows; r++) {
        for (let c = 0; c < cols; c++) {
          nodes.push({
            x: c * 120 + (Math.random() - 0.5) * 60,
            y: r * 120 + (Math.random() - 0.5) * 60,
            r: 1.5 + Math.random() * 1.5,
          });
        }
      }

      const time = Date.now() / 3000;

      // Draw nodes
      nodes.forEach(function (n) {
        ctx.beginPath();
        ctx.arc(n.x, n.y, n.r, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(0, 229, 255, 0.25)';
        ctx.fill();
      });

      // Draw edges
      nodes.forEach(function (a) {
        nodes.forEach(function (b) {
          const dx = b.x - a.x;
          const dy = b.y - a.y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          if (dist < 160 && dist > 20) {
            ctx.beginPath();
            ctx.moveTo(a.x, a.y);
            ctx.lineTo(b.x, b.y);
            ctx.strokeStyle = 'rgba(0, 229, 255, 0.08)';
            ctx.lineWidth = 0.5;
            ctx.stroke();
          }
        });
      });

      // Animated flow dots on some edges
      const flowEdges = nodes.flatMap(function (a) {
        return nodes.filter(function (b) {
          const dx = b.x - a.x;
          const dy = b.y - a.y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          return dist < 160 && dist > 60;
        }).map(function (b) {
          return { a: a, b: b };
        });
      }).slice(0, 12);

      flowEdges.forEach(function (edge) {
        const t = (time + edge.a.x * 0.01 + edge.a.y * 0.01) % 1;
        const x = edge.a.x + (edge.b.x - edge.a.x) * t;
        const y = edge.a.y + (edge.b.y - edge.a.y) * t;

        ctx.beginPath();
        ctx.arc(x, y, 2, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(0, 255, 65, 0.55)';
        ctx.fill();

        // Glow
        ctx.beginPath();
        ctx.arc(x, y, 4, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(0, 229, 255, 0.15)';
        ctx.fill();
      });

      animFrame = requestAnimationFrame(draw);
    }

    resize();
    draw();
    window.addEventListener('resize', resize);
  })();

  // --- Hero Topology Canvas ---
  (function () {
    const canvas = document.getElementById('hero-canvas');
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    let animFrame;

    function resize() {
      const rect = canvas.parentElement.getBoundingClientRect();
      canvas.width = rect.width * window.devicePixelRatio;
      canvas.height = rect.height * window.devicePixelRatio;
      canvas.style.width = rect.width + 'px';
      canvas.style.height = rect.height + 'px';
      ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
    }

    function draw() {
      const W = canvas.width / window.devicePixelRatio;
      const H = canvas.height / window.devicePixelRatio;
      const cx = W / 2;
      const cy = H / 2;
      const time = Date.now() / 1000;

      ctx.clearRect(0, 0, W, H);

      // Outer ring of scattered nodes
      const outerNodes = [];
      for (let i = 0; i < 14; i++) {
        const angle = (Math.PI * 2 / 14) * i + Math.sin(time + i) * 0.15;
        const radius = Math.min(W, H) * 0.38 + Math.sin(time * 0.7 + i) * 18;
        outerNodes.push({
          x: cx + Math.cos(angle) * radius,
          y: cy + Math.sin(angle) * radius,
          r: 5 + Math.sin(time + i) * 1.5,
        });
      }

      // Draw outer nodes
      outerNodes.forEach(function (n, i) {
        ctx.beginPath();
        ctx.arc(n.x, n.y, n.r, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(136, 136, 160, 0.5)';
        ctx.fill();
        ctx.strokeStyle = 'rgba(136, 136, 160, 0.2)';
        ctx.lineWidth = 1;
        ctx.stroke();

        // Label
        const labels = ['archivebox', 'atuin', 'beszel', 'karakeep', 'linkwarden',
          'miniflux', 'kanboard', 'fireflyiii', 'transmission', 'uptimekuma',
          'gitea', 'adguard', 'cloudflare', 'pushover'];
        ctx.font = '7px ' + getComputedStyle(document.body).fontFamily;
        ctx.fillStyle = 'rgba(136, 136, 160, 0.4)';
        ctx.textAlign = 'center';
        ctx.fillText(labels[i % labels.length], n.x, n.y - 12);
      });

      // Center hub ring
      const hubRadius = Math.min(W, H) * 0.14;
      ctx.beginPath();
      ctx.arc(cx, cy, hubRadius, 0, Math.PI * 2);
      ctx.strokeStyle = 'rgba(0, 229, 255, 0.6)';
      ctx.lineWidth = 2;
      ctx.stroke();

      // Inner ring
      ctx.beginPath();
      ctx.arc(cx, cy, hubRadius * 0.55, 0, Math.PI * 2);
      ctx.strokeStyle = 'rgba(0, 255, 65, 0.4)';
      ctx.lineWidth = 1.5;
      ctx.stroke();

      // Center dot
      ctx.beginPath();
      ctx.arc(cx, cy, 6, 0, Math.PI * 2);
      ctx.fillStyle = '#00e5ff';
      ctx.fill();
      ctx.beginPath();
      ctx.arc(cx, cy, 12, 0, Math.PI * 2);
      ctx.fillStyle = 'rgba(0, 229, 255, 0.18)';
      ctx.fill();

      // Flow lines from outer to center (data flowing in)
      outerNodes.forEach(function (n, i) {
        const t = (time * 0.6 + i * 0.3) % 1;
        const midX = n.x + (cx - n.x) * t;
        const midY = n.y + (cy - n.y) * t;

        // Line from outer to center
        ctx.beginPath();
        ctx.moveTo(n.x, n.y);
        ctx.lineTo(cx, cy);
        ctx.strokeStyle = 'rgba(0, 229, 255, 0.06)';
        ctx.lineWidth = 0.8;
        ctx.stroke();

        // Flowing dot
        ctx.beginPath();
        ctx.arc(midX, midY, 2.5, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(0, 229, 255, 0.7)';
        ctx.fill();
      });

      // Outgoing parallel lines (bottom right)
      const outStartX = cx + hubRadius * 1.1;
      const outStartY = cy;
      for (let i = -2; i <= 2; i++) {
        const yOff = i * 8;
        ctx.beginPath();
        const endX = W * 0.85;
        ctx.moveTo(outStartX, outStartY + yOff);
        ctx.lineTo(endX, outStartY + yOff);
        ctx.strokeStyle = 'rgba(0, 255, 65, 0.3)';
        ctx.lineWidth = 1;
        ctx.stroke();

        // Pulsing dot on outgoing line
        const t = (time * 0.8 + i * 0.5) % 1;
        const px = outStartX + (endX - outStartX) * t;
        ctx.beginPath();
        ctx.arc(px, outStartY + yOff, 2, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(0, 255, 65, 0.6)';
        ctx.fill();
      }

      // Outgoing labels
      ctx.font = '6px ' + getComputedStyle(document.body).fontFamily;
      ctx.fillStyle = 'rgba(0, 255, 65, 0.5)';
      ctx.textAlign = 'left';
      ctx.fillText('REST', W * 0.76, outStartY - 16);
      ctx.fillText('CLI', W * 0.80, outStartY - 6);
      ctx.fillText('Chat', W * 0.82, outStartY + 4);
      ctx.fillText('Form', W * 0.79, outStartY + 14);
      ctx.fillText('Webhook', W * 0.77, outStartY + 24);

      animFrame = requestAnimationFrame(draw);
    }

    resize();
    draw();
    window.addEventListener('resize', resize);
  })();

  // --- Workflow DAG Background Canvas ---
  (function () {
    const canvas = document.getElementById('workflow-canvas');
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    let animFrame;

    function resize() {
      const rect = canvas.parentElement.getBoundingClientRect();
      canvas.width = rect.width * window.devicePixelRatio;
      canvas.height = rect.height * window.devicePixelRatio;
      canvas.style.width = rect.width + 'px';
      canvas.style.height = rect.height + 'px';
      ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
    }

    function draw() {
      const W = canvas.width / window.devicePixelRatio;
      const H = canvas.height / window.devicePixelRatio;
      const time = Date.now() / 1000;
      ctx.clearRect(0, 0, W, H);

      // Draw a larger DAG
      function dagNode(x, y, label) {
        const w = label.length * 9 + 24;
        const h = 28;
        ctx.beginPath();
        ctx.roundRect(x - w / 2, y - h / 2, w, h, 6);
        ctx.fillStyle = 'rgba(18, 18, 29, 0.7)';
        ctx.fill();
        ctx.strokeStyle = 'rgba(30, 30, 48, 0.5)';
        ctx.lineWidth = 1;
        ctx.stroke();
        ctx.font = '9px ' + getComputedStyle(document.body).fontFamily;
        ctx.fillStyle = 'rgba(136, 136, 160, 0.5)';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(label, x, y);
      }

      function dagEdge(x1, y1, x2, y2) {
        ctx.beginPath();
        ctx.moveTo(x1, y1);
        ctx.lineTo(x2, y2);
        ctx.strokeStyle = 'rgba(0, 229, 255, 0.10)';
        ctx.lineWidth = 1;
        ctx.stroke();

        // Arrowhead
        const angle = Math.atan2(y2 - y1, x2 - x1);
        const len = 6;
        const ax = x2 - Math.cos(angle) * 16;
        const ay = y2 - Math.sin(angle) * 16;
        ctx.beginPath();
        ctx.moveTo(ax, ay);
        ctx.lineTo(
          ax - Math.cos(angle - 0.5) * len,
          ay - Math.sin(angle - 0.5) * len
        );
        ctx.lineTo(
          ax - Math.cos(angle + 0.5) * len,
          ay - Math.sin(angle + 0.5) * len
        );
        ctx.closePath();
        ctx.fillStyle = 'rgba(0, 229, 255, 0.15)';
        ctx.fill();
      }

      // Layout
      const cols = 5;
      const rows = 3;
      const startX = W * 0.1;
      const endX = W * 0.9;
      const stepX = (endX - startX) / (cols - 1);
      const startY = H * 0.2;
      const endY = H * 0.8;
      const stepY = (endY - startY) / (rows - 1);

      const labels = [
        ['cron', '', 'reader.fetch', '', 'llm.summarize'],
        ['', 'message', '', '', 'notify.send'],
        ['webhook', '', 'kanban.move', '', 'pipeline.run'],
      ];

      const nodes = [];
      for (let r = 0; r < rows; r++) {
        for (let c = 0; c < cols; c++) {
          if (labels[r] && labels[r][c]) {
            nodes.push({
              x: startX + c * stepX + Math.sin(time + r * c) * 10,
              y: startY + r * stepY + Math.cos(time * 0.7 + r) * 10,
              label: labels[r][c],
            });
          }
        }
      }

      // Edges
      for (let i = 0; i < nodes.length - 1; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          if (Math.abs(nodes[j].x - nodes[i].x) < stepX * 2 &&
              Math.abs(nodes[j].y - nodes[i].y) < stepY * 2 &&
              nodes[j].x > nodes[i].x) {
            dagEdge(nodes[i].x + 30, nodes[i].y, nodes[j].x - 30, nodes[j].y);
          }
        }
      }

      // Nodes
      nodes.forEach(function (n) { dagNode(n.x, n.y, n.label); });

      // Flowing dots on edges
      const allEdges = [];
      for (let i = 0; i < nodes.length - 1; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          if (Math.abs(nodes[j].x - nodes[i].x) < stepX * 2 &&
              Math.abs(nodes[j].y - nodes[i].y) < stepY * 2 &&
              nodes[j].x > nodes[i].x) {
            allEdges.push([nodes[i], nodes[j]]);
          }
        }
      }

      allEdges.slice(0, 8).forEach(function (pair, idx) {
        const t = (time * 0.5 + idx * 0.35) % 1;
        const x = pair[0].x + 30 + (pair[1].x - 30 - pair[0].x - 30) * t;
        const y = pair[0].y + (pair[1].y - pair[0].y) * t;
        ctx.beginPath();
        ctx.arc(x, y, 2, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(0, 229, 255, 0.4)';
        ctx.fill();
      });

      animFrame = requestAnimationFrame(draw);
    }

    resize();
    draw();
    window.addEventListener('resize', resize);
  })();

  // --- Smooth scroll for anchor links ---
  document.querySelectorAll('a[href^="#"]').forEach(function (anchor) {
    anchor.addEventListener('click', function (e) {
      const targetId = this.getAttribute('href').substring(1);
      const target = document.getElementById(targetId);
      if (target) {
        e.preventDefault();
        target.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    });
  });
})();
