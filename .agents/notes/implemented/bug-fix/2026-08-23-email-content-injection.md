# Agent Note: SMTP send sanitizes untrusted MIME fields

Status: implemented

## Problem

CodeQL `go/email-injection` (CWE-640) flags `smtp.Client.Data()` `Write` in `pkg/providers/email/smtp.go`. Capability `email.send` accepts To/Cc/Bcc/Subject/Text/HTML from invoke params, so threat-model taint reaches the SMTP DATA writer. Header CRLF can inject extra recipients; HTML can carry script/handlers; a raw body can smuggle `=` encoded-words. Copilot Autofix's `html.EscapeString` on the whole message would corrupt MIME headers and would still not stop header injection (`EscapeString` leaves CR/LF unchanged).

## Decision

`sanitizeSendInput` is the send-path barrier: `net/mail.ParseAddress` plus CRLF/NUL rejection on To/Cc/Bcc/Subject/FromName, bluemonday `UGCPolicy` on HTML, and stripping of C0/DEL from bodies except tab/LF/CR. `buildMIMEMessage` RFC 2047 Q-encodes Subject and quoted-printable-encodes every body part (`=` becomes `=3D`). Header injection is stopped by CRLF/NUL rejection and writing headers before the body separator. MIME part injection is stopped by a 128-bit random `flowbot-` boundary; quoted-printable does not encode `--`. `rejectMIMEHeaders` re-checks From/Subject/To/Cc/Bcc before assembly, including Bcc which is envelope-only and never a MIME header. A `codeql[go/email-injection]` comment sits on the line immediately above DATA `Write` because the query has no sanitizer class. `html.EscapeString` is not applied to HTML or text/plain.

## Alternatives considered

- **`html.EscapeString` at the DATA sink (Copilot Autofix).** Escaping the assembled MIME byte slice breaks `From`/`boundary` structure; escaping only HTML after bluemonday double-escapes tags; escaping text/plain turns `a < b` into entities. It also does not strip CR/LF.
- **Treat the alert as a false positive and only dismiss it in GitHub.** Leaves header/HTML/MIME injection unhardened.
- **CodeQL models-as-data `barrierModel`.** Default GitHub code scanning setup does not load a repo model pack, so the alert would stay.

## Consequences

Callers can still send HTML, but scripts/handlers are stripped. Non-ASCII subjects become encoded-words. Bodies with `=` are quoted-printable (`=3D`). Multipart parts start with `--flowbot-<random>`. Bcc is a recipient only; it does not appear in the MIME message. The scanner-visible suppression is the preceding-line `codeql[go/email-injection]` comment; that comment is invalid if `smtpSend` `Write` is given a message that skipped `sanitizeSendInput` and `buildMIMEMessage`.

## Verification

`TestBuildMIMEMessage` covers CRLF rejection on subject/To/FromName/Cc/Bcc, quoted From display names, Bcc omitted from MIME, HTML tags not entity-escaped, script stripping, quoted-printable `a=b` → `a=3Db`, UTF-8 subject encoded-words, C0 stripping, and multipart `--flowbot-` boundaries. `TestBuildMIMEMessageRejectsUnsanitizedAddressCRLF` covers To/Cc/Bcc CRLF when `buildMIMEMessage` is called without `sanitizeSendInput`. `go test ./pkg/providers/email/ -run 'TestBuildMIMEMessage|TestBuildMIMEMessageRejectsUnsanitizedAddressCRLF'`.
