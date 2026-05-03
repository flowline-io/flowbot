// ============================================================
// Flowbot Website — Interactive Effects
// ============================================================

(function () {
  "use strict";

  // --- Mobile Nav Toggle ---
  const navToggle = document.querySelector(".nav-toggle");
  const navLinks = document.querySelector(".nav-links");
  if (navToggle && navLinks) {
    navToggle.addEventListener("click", function () {
      navLinks.classList.toggle("open");
    });
  }

  // --- Active nav link based on scroll ---
  const sections = document.querySelectorAll(".section[id]");
  const navAnchors = document.querySelectorAll('.nav-links a[href^="#"]');

  function updateActiveLink() {
    let current = "";
    sections.forEach(function (sec) {
      const top = sec.offsetTop - 120;
      if (window.scrollY >= top) {
        current = sec.getAttribute("id");
      }
    });
    navAnchors.forEach(function (a) {
      a.classList.remove("active");
      if (a.getAttribute("href") === "#" + current) {
        a.classList.add("active");
      }
    });
  }

  if (sections.length && navAnchors.length) {
    window.addEventListener("scroll", updateActiveLink, { passive: true });
  }

  // Active link for subpages
  (function () {
    const path = window.location.pathname;
    document.querySelectorAll(".nav-links a[href]").forEach(function (link) {
      var href = link.getAttribute("href");
      // Skip anchor-only links (those are handled by scroll)
      if (href.charAt(0) === "#") return;
      // Skip external links
      if (href.indexOf("://") !== -1) return;
      // Match if path ends with the href (supports directory-prefixed links)
      if (href && path.endsWith(href)) {
        link.classList.add("active");
        return;
      }
      // Match if path ends with href/index.html (directory-style links)
      if (href && (path + "index.html").endsWith(href)) {
        link.classList.add("active");
        return;
      }
      // Docs section: highlight Docs nav on any /docs/ page
      if (path.indexOf("/docs/") !== -1 && href.indexOf("docs/") === 0) {
        link.classList.add("active");
        return;
      }
      // Fallback: match last path segment
      var currentPage = path.split("/").filter(Boolean).pop() || "index.html";
      if (currentPage === href) {
        link.classList.add("active");
      }
    });
  })();

  // --- Terminal Typing Effect ---
  (function () {
    const terminal = document.getElementById("typing-terminal");
    if (!terminal) return;

    const lines = [
      { text: "$ flowbot scan", cls: "cmd", delay: 800 },
      { text: "Scanning /homelab/apps/...", cls: "output", delay: 400 },
      {
        text: "Found 12 apps: archivebox, atuin, beszel, karakeep, linkwarden, miniflux, kanboard, fireflyiii, transmission, uptimekuma, gitea, adguard",
        cls: "output",
        delay: 700,
      },
      { text: "$ flowbot run pipeline backup", cls: "cmd", delay: 600 },
      {
        text: "[pipeline:backup] DAG executed successfully",
        cls: "output",
        delay: 500,
      },
      {
        text: "[pipeline:backup] 8/8 steps completed, 0 errors",
        cls: "output",
        delay: 0,
      },
    ];

    const container = terminal.querySelector(".terminal-lines");
    const cursor = terminal.querySelector(".terminal-cursor");
    let lineIdx = 0;

    function typeLine(line, callback) {
      const div = document.createElement("div");
      div.className = "terminal-line";
      container.appendChild(div);

      const span = document.createElement("span");
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
        if (cursor) cursor.style.animation = "blink 1s step-end infinite";
        return;
      }
      typeLine(lines[lineIdx], function () {
        lineIdx++;
        startTyping();
      });
    }

    // Pause cursor blink during typing
    if (cursor) cursor.style.animation = "none";
    setTimeout(startTyping, 600);
  })();

  // --- Background Node Graph (Canvas) ---
  (function () {
    const canvas = document.getElementById("bg-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    let animFrame;

    function resize() {
      const rect = canvas.parentElement.getBoundingClientRect();
      canvas.width = rect.width * window.devicePixelRatio;
      canvas.height = rect.height * window.devicePixelRatio;
      canvas.style.width = rect.width + "px";
      canvas.style.height = rect.height + "px";
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
        ctx.fillStyle = "rgba(0, 229, 255, 0.25)";
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
            ctx.strokeStyle = "rgba(0, 229, 255, 0.08)";
            ctx.lineWidth = 0.5;
            ctx.stroke();
          }
        });
      });

      // Animated flow dots on some edges
      const flowEdges = nodes
        .flatMap(function (a) {
          return nodes
            .filter(function (b) {
              const dx = b.x - a.x;
              const dy = b.y - a.y;
              const dist = Math.sqrt(dx * dx + dy * dy);
              return dist < 160 && dist > 60;
            })
            .map(function (b) {
              return { a: a, b: b };
            });
        })
        .slice(0, 12);

      flowEdges.forEach(function (edge) {
        const t = (time + edge.a.x * 0.01 + edge.a.y * 0.01) % 1;
        const x = edge.a.x + (edge.b.x - edge.a.x) * t;
        const y = edge.a.y + (edge.b.y - edge.a.y) * t;

        ctx.beginPath();
        ctx.arc(x, y, 2, 0, Math.PI * 2);
        ctx.fillStyle = "rgba(0, 255, 65, 0.55)";
        ctx.fill();

        // Glow
        ctx.beginPath();
        ctx.arc(x, y, 4, 0, Math.PI * 2);
        ctx.fillStyle = "rgba(0, 229, 255, 0.15)";
        ctx.fill();
      });

      animFrame = requestAnimationFrame(draw);
    }

    resize();
    draw();
    window.addEventListener("resize", resize);
  })();

  // --- Hero Topology Canvas ---
  (function () {
    const canvas = document.getElementById("hero-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    let animFrame;

    function resize() {
      const rect = canvas.parentElement.getBoundingClientRect();
      canvas.width = rect.width * window.devicePixelRatio;
      canvas.height = rect.height * window.devicePixelRatio;
      canvas.style.width = rect.width + "px";
      canvas.style.height = rect.height + "px";
      ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
    }

    function draw() {
      const W = canvas.width / window.devicePixelRatio;
      const H = canvas.height / window.devicePixelRatio;
      const cx = W / 2;
      const cy = H / 2;
      const time = Date.now() / 1000;
      const D = Math.min(W, H);

      ctx.clearRect(0, 0, W, H);

      // ---- Orbital rings ----
      const orbits = [
        { r: D * 0.38, nodes: 6, speed: 0.12, offset: 0 },
        { r: D * 0.3, nodes: 5, speed: 0.2, offset: Math.PI / 5 },
        { r: D * 0.2, nodes: 4, speed: 0.32, offset: Math.PI / 3 },
      ];

      // App labels shown near outer orbit nodes
      var appLabels = [
        "karakeep",
        "archivebox",
        "miniflux",
        "kanboard",
        "fireflyiii",
        "atuin",
        "beszel",
        "uptimekuma",
        "adguard",
        "gitea",
        "linkwarden",
      ];

      orbits.forEach(function (orbit, oi) {
        // Draw faint orbital path
        ctx.beginPath();
        ctx.arc(cx, cy, orbit.r, 0, Math.PI * 2);
        ctx.strokeStyle = "rgba(0, 229, 255, 0.07)";
        ctx.lineWidth = 0.8;
        ctx.setLineDash([3, 12]);
        ctx.stroke();
        ctx.setLineDash([]);

        // Draw orbiting nodes
        for (let i = 0; i < orbit.nodes; i++) {
          const angle =
            ((Math.PI * 2) / orbit.nodes) * i +
            time * orbit.speed +
            orbit.offset;
          const x = cx + Math.cos(angle) * orbit.r;
          const y = cy + Math.sin(angle) * orbit.r;
          const pulse = 1 + Math.sin(time * 3 + i * 1.7) * 0.3;
          const nodeR = 3 * pulse;

          // Glow halo
          var grd = ctx.createRadialGradient(x, y, 0, x, y, nodeR * 4);
          grd.addColorStop(0, "rgba(0, 229, 255, 0.35)");
          grd.addColorStop(0.5, "rgba(0, 229, 255, 0.08)");
          grd.addColorStop(1, "rgba(0, 229, 255, 0)");
          ctx.beginPath();
          ctx.arc(x, y, nodeR * 4, 0, Math.PI * 2);
          ctx.fillStyle = grd;
          ctx.fill();

          // Core dot
          ctx.beginPath();
          ctx.arc(x, y, nodeR, 0, Math.PI * 2);
          ctx.fillStyle = "#00e5ff";
          ctx.fill();

          // Label for outer orbit only
          if (oi === 0) {
            var labelIdx = i % appLabels.length;
            const labelAngle = angle;
            const labelR = orbit.r + 15;
            const lx = cx + Math.cos(labelAngle) * labelR;
            const ly = cy + Math.sin(labelAngle) * labelR;
            ctx.font = "500 9px 'JetBrains Mono', 'Fira Code', monospace";
            ctx.fillStyle = "rgba(136, 136, 160, 0.55)";
            ctx.textAlign = "center";
            ctx.textBaseline = "middle";
            ctx.fillText(appLabels[labelIdx], lx, ly);
          }
        }
      });

      // ---- Data particles flowing inward ----
      const particleCount = 18;
      for (let i = 0; i < particleCount; i++) {
        const seed = i * 0.37;
        const t = (time * 0.5 + seed) % 1;
        // Ease-in curve for acceleration toward center
        const et = t * t;
        const startAngle = seed * Math.PI * 2;
        const startR = D * 0.42;
        const sx = cx + Math.cos(startAngle) * startR;
        const sy = cy + Math.sin(startAngle) * startR;
        const ex = cx + Math.cos(startAngle + 0.3) * D * 0.06;
        const ey = cy + Math.sin(startAngle + 0.3) * D * 0.06;
        const px = sx + (ex - sx) * et;
        const py = sy + (ey - sy) * et;

        // Trail line
        var startT = Math.max(0, t - 0.12);
        var trailSx = sx + (ex - sx) * (startT * startT);
        var trailSy = sy + (ey - sy) * (startT * startT);
        ctx.beginPath();
        ctx.moveTo(trailSx, trailSy);
        ctx.lineTo(px, py);
        ctx.strokeStyle = "rgba(0, 229, 255, " + 0.15 * (1 - t) + ")";
        ctx.lineWidth = 1.2;
        ctx.stroke();

        // Particle dot
        ctx.beginPath();
        ctx.arc(px, py, 1.5, 0, Math.PI * 2);
        ctx.fillStyle = "rgba(0, 229, 255, " + 0.7 * (1 - t * 0.5) + ")";
        ctx.fill();
      }

      // ---- Center Hub ----
      const hubR = D * 0.1;

      // Outer glow ring
      var ringGrd = ctx.createRadialGradient(
        cx,
        cy,
        hubR * 0.6,
        cx,
        cy,
        hubR * 1.4,
      );
      ringGrd.addColorStop(0, "rgba(0, 229, 255, 0)");
      ringGrd.addColorStop(0.7, "rgba(0, 229, 255, 0.15)");
      ringGrd.addColorStop(1, "rgba(0, 229, 255, 0)");
      ctx.beginPath();
      ctx.arc(cx, cy, hubR * 1.4, 0, Math.PI * 2);
      ctx.fillStyle = ringGrd;
      ctx.fill();

      // Hub border ring
      ctx.beginPath();
      ctx.arc(cx, cy, hubR, 0, Math.PI * 2);
      ctx.strokeStyle = "rgba(0, 229, 255, 0.7)";
      ctx.lineWidth = 2;
      ctx.stroke();

      // Hub inner fill
      var hubGrd = ctx.createRadialGradient(cx, cy, 0, cx, cy, hubR);
      hubGrd.addColorStop(0, "rgba(0, 229, 255, 0.2)");
      hubGrd.addColorStop(1, "rgba(0, 229, 255, 0.04)");
      ctx.beginPath();
      ctx.arc(cx, cy, hubR, 0, Math.PI * 2);
      ctx.fillStyle = hubGrd;
      ctx.fill();

      // Pulsing inner dot
      var corePulse = 1 + Math.sin(time * 2.5) * 0.25;
      var coreGrd = ctx.createRadialGradient(
        cx,
        cy,
        0,
        cx,
        cy,
        hubR * 0.45 * corePulse,
      );
      coreGrd.addColorStop(0, "#00e5ff");
      coreGrd.addColorStop(0.4, "rgba(0, 229, 255, 0.6)");
      coreGrd.addColorStop(1, "rgba(0, 229, 255, 0)");
      ctx.beginPath();
      ctx.arc(cx, cy, hubR * 0.45 * corePulse, 0, Math.PI * 2);
      ctx.fillStyle = coreGrd;
      ctx.fill();

      // ---- Outgoing data streams (right side, beam-like) ----
      var outStartX = cx + hubR + 4;
      var outEndX = W * 0.92;
      var beams = 4;

      for (var b = 0; b < beams; b++) {
        var by = cy + (b - (beams - 1) / 2) * (hubR * 0.9);
        var lineAlpha = 0.25 + Math.sin(time * 1.5 + b) * 0.1;

        // Beam line
        ctx.beginPath();
        ctx.moveTo(outStartX, by);
        ctx.lineTo(outEndX, by);
        ctx.strokeStyle = "rgba(0, 255, 65, " + lineAlpha + ")";
        ctx.lineWidth = 1.2;
        ctx.stroke();

        // Beam glow
        ctx.beginPath();
        ctx.moveTo(outStartX, by);
        ctx.lineTo(outEndX, by);
        ctx.strokeStyle = "rgba(0, 255, 65, " + lineAlpha * 0.4 + ")";
        ctx.lineWidth = 4;
        ctx.stroke();

        // Traveling pulses
        var pulseCount = 2;
        for (var p = 0; p < pulseCount; p++) {
          var pt = (time * 0.6 + b * 0.25 + p * 0.5) % 1;
          var ppx = outStartX + (outEndX - outStartX) * pt;
          ctx.beginPath();
          ctx.arc(ppx, by, 1.8, 0, Math.PI * 2);
          ctx.fillStyle = "rgba(0, 255, 65, 0.8)";
          ctx.fill();
          ctx.beginPath();
          ctx.arc(ppx, by, 3.5, 0, Math.PI * 2);
          ctx.fillStyle = "rgba(0, 255, 65, 0.2)";
          ctx.fill();
        }
      }

      // ---- Interface labels under beams ----
      var ifaceLabels = ["REST", "CLI", "Chat", "Webhook"];
      ctx.font = "600 10px 'Inter', 'Helvetica Neue', sans-serif";
      ctx.textAlign = "left";
      var labelX = outEndX - 60;
      ifaceLabels.forEach(function (lbl, i) {
        var ly = cy + (i - (ifaceLabels.length - 1) / 2) * (hubR * 0.9);
        ctx.fillStyle =
          "rgba(0, 255, 65, " + (0.4 + Math.sin(time + i) * 0.1) + ")";
        ctx.fillText(lbl, labelX, ly + 3);
      });

      animFrame = requestAnimationFrame(draw);
    }

    resize();
    draw();
    window.addEventListener("resize", resize);
  })();

  // --- Workflow DAG Background Canvas ---
  (function () {
    const canvas = document.getElementById("workflow-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    let animFrame;

    function resize() {
      const rect = canvas.parentElement.getBoundingClientRect();
      canvas.width = rect.width * window.devicePixelRatio;
      canvas.height = rect.height * window.devicePixelRatio;
      canvas.style.width = rect.width + "px";
      canvas.style.height = rect.height + "px";
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
        ctx.fillStyle = "rgba(18, 18, 29, 0.7)";
        ctx.fill();
        ctx.strokeStyle = "rgba(30, 30, 48, 0.5)";
        ctx.lineWidth = 1;
        ctx.stroke();
        ctx.font = "9px " + getComputedStyle(document.body).fontFamily;
        ctx.fillStyle = "rgba(136, 136, 160, 0.5)";
        ctx.textAlign = "center";
        ctx.textBaseline = "middle";
        ctx.fillText(label, x, y);
      }

      function dagEdge(x1, y1, x2, y2) {
        ctx.beginPath();
        ctx.moveTo(x1, y1);
        ctx.lineTo(x2, y2);
        ctx.strokeStyle = "rgba(0, 229, 255, 0.10)";
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
          ay - Math.sin(angle - 0.5) * len,
        );
        ctx.lineTo(
          ax - Math.cos(angle + 0.5) * len,
          ay - Math.sin(angle + 0.5) * len,
        );
        ctx.closePath();
        ctx.fillStyle = "rgba(0, 229, 255, 0.15)";
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
        ["cron", "", "reader.fetch", "", "llm.summarize"],
        ["", "message", "", "", "notify.send"],
        ["webhook", "", "kanban.move", "", "pipeline.run"],
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
          if (
            Math.abs(nodes[j].x - nodes[i].x) < stepX * 2 &&
            Math.abs(nodes[j].y - nodes[i].y) < stepY * 2 &&
            nodes[j].x > nodes[i].x
          ) {
            dagEdge(nodes[i].x + 30, nodes[i].y, nodes[j].x - 30, nodes[j].y);
          }
        }
      }

      // Nodes
      nodes.forEach(function (n) {
        dagNode(n.x, n.y, n.label);
      });

      // Flowing dots on edges
      const allEdges = [];
      for (let i = 0; i < nodes.length - 1; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          if (
            Math.abs(nodes[j].x - nodes[i].x) < stepX * 2 &&
            Math.abs(nodes[j].y - nodes[i].y) < stepY * 2 &&
            nodes[j].x > nodes[i].x
          ) {
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
        ctx.fillStyle = "rgba(0, 229, 255, 0.4)";
        ctx.fill();
      });

      animFrame = requestAnimationFrame(draw);
    }

    resize();
    draw();
    window.addEventListener("resize", resize);
  })();

  // --- Smooth scroll for anchor links ---
  document.querySelectorAll('a[href^="#"]').forEach(function (anchor) {
    anchor.addEventListener("click", function (e) {
      const targetId = this.getAttribute("href").substring(1);
      const target = document.getElementById(targetId);
      if (target) {
        e.preventDefault();
        target.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    });
  });
})();
