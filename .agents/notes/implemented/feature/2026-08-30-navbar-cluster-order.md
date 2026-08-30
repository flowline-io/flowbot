# Agent Note: Navbar cluster order

Status: implemented

## Problem

The ops navbar packed search, inbox, product groups, identity, locale, theme, and logout into one right-hand strip. Search and inbox sat to the left of the primary groups, so global tools mixed with product navigation and the user cluster felt like a leftover pile.

## Decision

Split the bar into three clusters, left to right:

1. **Brand + primary groups** (`navbar-start`): logo, then Agent / Automate / Integrate / System (desktop).
2. **Tools** (`.flowbot-nav-tools` in `navbar-end`): command palette, then inbox.
3. **User** (`.flowbot-nav-user`): session identity, then locale / theme / logout.

Inbox stays desktop-only; mobile still uses the hamburger. A second hairline separates identity from locale/theme/logout.

## Alternatives considered

- **Center the product groups** with DaisyUI `navbar-center`. Rejected: DaisyUI start/end are 50% width, so a center cluster overlaps rather than sitting in a true middle gutter.
- **Fold locale, theme, and logout into a session dropdown.** Rejected: one-click logout and theme toggle would hide behind a menu; this change only reorders visible controls.

## Consequences

- Desktop HTML order is brand groups, then search, then inbox, then the user cluster. Tests lock that order via `data-testid` positions.
- Search remains visible on small viewports; inbox does not.

## Verification

- `TestBaseLayout` case `desktop navbar order is brand groups then tools then user` asserts marker order and the `nav-primary` / `flowbot-nav-tools` clusters.
- Existing badge poller counts and command-palette markup still apply.
