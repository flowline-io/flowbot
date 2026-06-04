# Theme Picker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the light/dark-only theme toggle with an 8-theme Alpine.js dropdown picker plus quick light/dark toggle in the navbar.

**Architecture:** Two-file change. `app.js` loses the old vanilla JS theme toggle listener. `base.templ` gains an Alpine `x-data` component managing theme state, a quick light/dark toggle button, and a DaisyUI dropdown listing 8 themes. No CSS vendor changes -- `themes.css` already contains all 32 DaisyUI themes.

**Tech Stack:** a-h/templ (Go templates), Alpine.js 3.14.9, DaisyUI 5.0.9, Tailwind CSS v4 browser runtime

---

### Task 1: Remove old theme toggle code from app.js

**Files:**
- Modify: `public/js/app.js`

- [ ] **Step 1: Remove theme toggle code from app.js**

Remove the `setTheme` function (lines 6-9) and the `DOMContentLoaded` theme toggle listener (lines 11-20), plus their associated comments. Keep everything else (Alpine toast store init, `showToast`).

Resulting file:

```javascript
// Alpine.js shared data stores and utilities
document.addEventListener('alpine:init', () => {
  Alpine.store('toasts', []);
});

// Toast notification system - used by pipeline-editor.js and other components
// eslint-disable-next-line no-unused-vars
function showToast(message, type) {
  type = type || 'info';
  var container = document.getElementById('toast-container');
  if (!container) return;

  var item = document.createElement('div');
  item.className = 'toast-item toast-' + type;
  item.textContent = message;

  container.appendChild(item);

  setTimeout(function () {
    item.classList.add('toast-removing');
    setTimeout(function () {
      if (item.parentNode) item.parentNode.removeChild(item);
    }, 300);
  }, 4000);
}
```

- [ ] **Step 2: Verify app.js is clean**

```bash
wc -l public/js/app.js
```

Expected: ~26 lines (down from 41). Confirm no syntax errors by checking in a browser or with Node.

- [ ] **Step 3: Commit**

```bash
git add public/js/app.js
git commit -m "refactor: remove vanilla theme toggle from app.js"
```

---

### Task 2: Rewrite navbar in base.templ with Alpine theme controls

**Files:**
- Modify: `pkg/views/layout/base.templ`

- [ ] **Step 1: Replace the old theme toggle and CSS with Alpine-driven controls**

In `pkg/views/layout/base.templ`, replace lines 40-44 (the old `dropdown dropdown-end` block with the theme toggle button) and lines 59-60 (the `[data-theme]` CSS rules) with the new Alpine component.

**Replace lines 40-44** (the old theme toggle button block):

```html
				<div class="dropdown dropdown-end">
					<button tabindex="0" class="btn btn-ghost btn-sm btn-square" data-testid="theme-toggle" aria-label="Toggle theme">
						<svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 hidden dark-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/></svg>
						<svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 hidden light-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/></svg>
					</button>
				</div>
```

**With the new theme controls:**

```html
				<div x-data="{
					theme: 'light',
					open: false,
					themes: ['light', 'dark', 'cupcake', 'synthwave', 'cyberpunk', 'forest', 'dracula', 'nord'],
					setTheme(name) {
						document.documentElement.setAttribute('data-theme', name);
						localStorage.setItem('flowbot-theme', name);
						this.theme = name;
						this.open = false;
					},
					toggleLightDark() {
						this.setTheme(this.theme === 'light' ? 'dark' : 'light');
					},
					init() {
						this.theme = document.documentElement.getAttribute('data-theme') || 'light';
					}
				}" class="flex items-center gap-0.5" @keydown.escape="open = false">
					<button @click="toggleLightDark()" class="btn btn-ghost btn-sm btn-square" data-testid="theme-quick-toggle" aria-label="Toggle light/dark">
						<svg x-show="theme === 'light'" x-cloak xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/></svg>
						<svg x-show="theme !== 'light'" x-cloak xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/></svg>
					</button>
					<div class="dropdown dropdown-end">
						<button tabindex="0" @click="open = !open" class="btn btn-ghost btn-sm btn-square" data-testid="theme-picker" aria-label="Pick theme">
							<svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.926 0 1.648-.746 1.648-1.688 0-.437-.18-.835-.437-1.125-.29-.289-.438-.652-.438-1.125a1.64 1.64 0 0 1 1.668-1.668h1.996c3.051 0 5.555-2.503 5.555-5.554C21.965 6.012 17.461 2 12 2z"/></svg>
						</button>
						<ul x-show="open" @click.outside="open = false" class="dropdown-content menu bg-base-100 rounded-box shadow-sm w-36 z-50 mt-1 p-1 border border-base-300">
							<li><button type="button" @click="setTheme('light')" :class="theme === 'light' ? 'active' : ''">Light</button></li>
							<li><button type="button" @click="setTheme('dark')" :class="theme === 'dark' ? 'active' : ''">Dark</button></li>
							<li><button type="button" @click="setTheme('cupcake')" :class="theme === 'cupcake' ? 'active' : ''">Cupcake</button></li>
							<li><button type="button" @click="setTheme('synthwave')" :class="theme === 'synthwave' ? 'active' : ''">Synthwave</button></li>
							<li><button type="button" @click="setTheme('cyberpunk')" :class="theme === 'cyberpunk' ? 'active' : ''">Cyberpunk</button></li>
							<li><button type="button" @click="setTheme('forest')" :class="theme === 'forest' ? 'active' : ''">Forest</button></li>
							<li><button type="button" @click="setTheme('dracula')" :class="theme === 'dracula' ? 'active' : ''">Dracula</button></li>
							<li><button type="button" @click="setTheme('nord')" :class="theme === 'nord' ? 'active' : ''">Nord</button></li>
						</ul>
					</div>
				</div>
```

**Replace the old CSS icon visibility rules (lines 59-60):**

Remove:
```css
[data-theme="light"] .dark-icon { display: inline-block !important; margin-left: 3px; }
[data-theme="dark"] .light-icon { display: inline-block !important; margin-left: 3px; }
```

These are no longer needed since Alpine `x-show` handles icon visibility now. The second line (line 60 currently, will shift) gets removed entirely from the `<style>` block in `<body>`.

- [ ] **Step 2: Verify the templ file is syntactically correct**

Read back the file to confirm:
- The old dropdown/button block is replaced
- The old CSS rules are removed
- The new Alpine div sits between nav links and the logout button (inside `navbar-end`)
- All HTML tags are properly closed

- [ ] **Step 3: Commit**

```bash
git add pkg/views/layout/base.templ
git commit -m "feat: add 8-theme Alpine dropdown picker to navbar"
```

---

### Task 3: Regenerate templ code and build

**Files:**
- Regenerate: `pkg/views/layout/base_templ.go` (auto-generated from base.templ)

- [ ] **Step 1: Regenerate templ Go code**

```bash
go tool templ generate pkg/views/layout/
```

Expected: No errors. `base_templ.go` is updated.

- [ ] **Step 2: Check generated code for Alpine directives**

```bash
grep -c 'x-data' pkg/views/layout/base_templ.go
```

Expected: `1` (the x-data block is present in generated output).

- [ ] **Step 3: Build the server**

```bash
go build ./cmd/server/
```

Expected: No build errors.

- [ ] **Step 4: Run lint**

```bash
go tool task lint
```

Expected: No new lint errors.

- [ ] **Step 5: Commit**

```bash
git add pkg/views/layout/base_templ.go
git commit -m "chore: regenerate templ code for theme picker"
```

---

### Task 4: Manual verification

- [ ] **Step 1: Start the dev server**

```bash
go run ./cmd/server/
```

- [ ] **Step 2: Verify in browser**
  - Open `http://localhost:8080/service/web/home`
  - Confirm two theme buttons in navbar: sun/moon icon + palette icon
  - Click palette icon -- verify dropdown opens with 8 theme names
  - Click "Dracula" -- verify theme changes, dropdown closes, active theme highlighted
  - Click sun/moon quick toggle -- verify flips between light and dark
  - Reload page -- verify theme persists from localStorage
  - Set theme to "synthwave", reload -- verify no flash of wrong theme
  - Open dropdown, press Escape -- verify dropdown closes
  - Open dropdown, click outside -- verify dropdown closes
  - Clear localStorage, reload -- verify falls back to light theme
