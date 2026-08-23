# Agent Note: Production navbar CSS stale / unsized SVG

Status: implemented

## Problem

After a deploy, the ops navbar user cluster renders unstyled: the session avatar SVG fills ~300×150px (browser default for a `viewBox` SVG with no used size) and the identity / language / logout controls stack instead of sitting in one row. Home stats still look correct because they use utilities that already exist in `app.css`.

Two independent gaps produce that screenshot. Layout CSS URLs have no version query while `staticCacheMiddleware` sets `max-age=3600`, so a new binary's HTML can pair with a cached previous `custom.css` that lacks `.flowbot-nav-user*`. Independently, the session-badge SVG uses `w-3.5 h-3.5`, which were missing from the committed Tailwind bundle, so even a fresh CSS load left the icon at the replaced-element default (`svg { display: block }` from preflight, and `max-width: 100%` applies only to `img, video`).

## Decision

Layouts cache-bust `app.css` / `custom.css` / `chatagent-markdown.css` / `clip.css` with `?v=` + `version.Buildtags`, matching JS. The session-badge SVG carries `width="14"` `height="14"`. `app.css` includes `.w-3.5` / `.h-3.5`. `.flowbot-nav-user-avatar` clips overflow and sizes its `svg`.

## Alternatives considered

- **`Cache-Control: no-cache` on `/static/css/*`.** Forces a revalidate on every page load; still leaves the unsized SVG bug when CSS is current.
- **Rebuild the whole Tailwind bundle from templates.** There is no in-repo npm CSS build; the committed `app.css` is the source. Adding the two missing utilities is the local fix.
- **Drop Tailwind classes and style the icon only in `custom.css`.** Would hide the missing-utility class of bug for this one SVG; other `w-3.5` call sites would still inflate.

## Consequences

A new `Buildtags` value fetches new CSS immediately. The avatar has an intrinsic size even if a utility is missing later. Vendor CSS (`katex.min.css`) stays immutable-cached.

## Verification

`TestBaseLayout` / `TestAuthLayout` require `app.css?v=` and `custom.css?v=`. `TestSessionBadgeSVGHasIntrinsicSize` requires `width="14"` / `height="14"`. `TestCommittedCSSIncludesNavbarIconUtilities` requires `.w-3.5` / `.h-3.5` in `app.css` and `.flowbot-nav-user-avatar svg` in `custom.css`.
