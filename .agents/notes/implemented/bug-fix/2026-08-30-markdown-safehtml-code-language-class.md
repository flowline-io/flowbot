# Agent Note: Preserve fenced code language class in MarkdownToSafeHTML

Status: implemented

## Problem

Chatagent codeblock chrome labeled fences as `text` and microlighter skipped highlighting even when the source fence was `` ```python ``. goldmark emitted `<code class="language-python">`, but `MarkdownSanitizePolicy` only allowed `class` on `span`, so bluemonday stripped the language class before the browser saw the HTML.

## Decision

Allow a safe `class` attribute on `code` and `pre` in `MarkdownSanitizePolicy` (same character class pattern as KaTeX spans, plus `+` for rare fence names). Keep sanitization otherwise unchanged.

## Alternatives considered

- **Infer language client-side from fence text.** Rejected: the class is already correct pre-sanitize; recovering it after the fact is fragile.
- **Bypass sanitize for code blocks.** Rejected: still need UGC policy for the rest of the message HTML.

## Consequences

- `MarkdownToSafeHTML` keeps `language-*` on fenced blocks; `chatagent-codeblocks.js` language labels and microlighter aliases work again.
- Regression covered by `TestMarkdownToSafeHTML/preserves_fenced_code_language_class`.

## Verification

- `go test ./pkg/utils/ -run TestMarkdownToSafeHTML/preserves_fenced_code_language_class` asserts `<code class="language-python">` survives sanitize.
