# Theme Picker Design

## Overview

Replace the current light/dark-only theme toggle with an 8-theme dropdown picker plus a quick light/dark toggle button in the navbar. Themes persist to `localStorage` and restore on page load.

### Themes

`light`, `dark`, `cupcake`, `synthwave`, `cyberpunk`, `forest`, `dracula`, `nord`

---

## Architecture

Changes touch 2 files only (CSS already contains all 32 themes):

| What | File | Change |
|---|---|---|
| Template | `pkg/views/layout/base.templ` | Replace single toggle button with Alpine dropdown + quick-toggle button |
| JS | `public/js/app.js` | Remove old vanilla theme toggle `DOMContentLoaded` listener; keep `setTheme()` as utility |

No CSS vendor changes needed -- `public/vendor/themes.css` (downloaded via `scripts/vendor.sh` from `daisyui@5.0.9/dist/themes.css`) already includes all 32 DaisyUI theme definitions.

---

## Data Flow

```
localStorage (flowbot-theme)
       |
       v
<head> inline script                   <-- prevents flash on load (unchanged)
  reads localStorage, sets data-theme
       |
       v
Alpine x-data="themePicker"
  state: { theme, open }
  themes: [light, dark, ...]

  setTheme(name):
    -> documentElement.setAttribute('data-theme', name)
    -> localStorage.setItem('flowbot-theme', name)
    -> this.theme = name
    -> this.open = false

  toggleLightDark():
    -> setTheme(this.theme === 'light' ? 'dark' : 'light')

  init():
    -> this.theme = documentElement.getAttribute('data-theme') || 'light'
```

### localStorage Schema

- Key: `flowbot-theme` (unchanged)
- Value: one of `light | dark | cupcake | synthwave | cyberpunk | forest | dracula | nord`

---

## Component Design

### Navbar Layout

Navbar right side contains two theme controls:

```
[Quick Toggle]  [Theme Dropdown Trigger]
    sun/moon         palette icon + caret
```

**Quick Toggle**: Displays sun icon when current theme is `light`, moon icon for all other themes. Click flips between `light` and `dark`.

**Theme Dropdown**: Palette icon button. Click opens a DaisyUI dropdown listing all 8 themes. Current active theme shows a checkmark. Click any theme name applies it and closes the dropdown.

### Dropdown Behavior

- Opens on click of palette icon button
- Closes on: theme selection, click outside (`@click.outside`), Escape key (`@keydown.escape`)
- Active theme shows a checkmark indicator
- Non-light/non-dark themes: clicking the quick toggle switches to `light` if currently a non-light theme, otherwise to `dark`

---

## Implementation Notes

### Alpine Component

Implemented as an inline `x-data` on the navbar container or a dedicated wrapper element. No separate JS file needed.

```html
<div x-data="{
  theme: 'light',
  open: false,
  themes: ['light','dark','cupcake','synthwave','cyberpunk','forest','dracula','nord'],
  setTheme(name) { ... },
  toggleLightDark() { ... },
  init() { this.theme = document.documentElement.getAttribute('data-theme') || 'light'; },
  quickToggleIcon() { return this.theme === 'light' ? 'sun' : 'moon'; }
}">
```

### Inline Head Script (unchanged)

The existing flash-prevention script in `<head>` stays as-is:

```javascript
(function(){
  var t = localStorage.getItem('flowbot-theme');
  if (t) document.documentElement.setAttribute('data-theme', t);
})()
```

### app.js Cleanup

Remove the `DOMContentLoaded` theme toggle listener (lines 13-20 of current `app.js`). The `setTheme()` function can either:
- Stay as a utility for potential non-Alpine callers (safer)
- Be removed if no external callers exist

### SVG Icons

Use inline SVGs in the `.templ` file rather than external icon libraries. Keep the existing sun/moon SVGs for the quick toggle. Use a generic palette/swatch SVG for the dropdown trigger.

### CSS for Icon Visibility

Replace the current `[data-theme="light"] .dark-icon` / `[data-theme="dark"] .light-icon` rules with Alpine-driven `x-show` / `x-cloak` toggling. This eliminates the need for theme-specific CSS selectors for icon visibility.

---

## Edge Cases

| Scenario | Behavior |
|---|---|
| `localStorage` corrupted / invalid theme name | Falls back to `light` (hardcoded `data-theme` attribute on `<html>`) |
| `localStorage` unavailable (private browsing) | `setTheme()` silently skips `setItem`; theme applies via `data-theme` attribute |
| User clears `localStorage` | Next page load falls back to `light` |
| Rapid dropdown clicks | Each click calls `setTheme()` independently; only final value matters |
| CSS not loaded yet | Inline `<head>` script runs before CSS; browser renders default style until CSS arrives |
| JS disabled | Navbar shows no theme controls; `data-theme="light"` applied from HTML attribute |
| Non-light/non-dark theme active, user clicks quick toggle | Flips to `light` (safest default) -- the quick toggle always cycles between light and dark |

---

## Testing

### Unit / Layout Tests

No unit tests needed -- this is a pure frontend addition with no Go logic changes. Verify by:
- Manual testing: open page, cycle through all 8 themes, confirm localStorage persistence
- Verify no flash of wrong theme on page reload with each theme
- Verify dropdown opens/closes correctly (click, outside click, Escape)
- Verify quick toggle flips between light and dark from any starting theme

### Existing Test Impact

Removing the `DOMContentLoaded` listener in `app.js` means the old `data-testid="theme-toggle"` element is replaced. Any Go test that references `data-testid="theme-toggle"` in HTML assertions must be updated to match the new attributes.
