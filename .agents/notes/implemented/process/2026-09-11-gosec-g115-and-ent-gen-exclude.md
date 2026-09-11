# Agent Note: gosec G115 guards and ent/gen exclude-dir

Status: implemented

## Problem

`go tool task gosec` reported G115 integer-overflow casts in capability params, webauth TOTP counters, and event chip palette indexing, plus an unhandled Fiber `Redirect().To` error (G104) in `pkg/route`. Separately, on Windows gosec still *parses* `internal/store/ent/gen` under `-exclude-generated` and can fail with `strconv.Atoi` on absolute paths, producing a package-level golang error even when issues are otherwise clean.

## Decision

- Add/use range-checked converters in `pkg/utils` (`Uint64ToInt`, `Int64ToUint64`, existing `Uint64ToInt64`) at the flagged call sites so out-of-range values fail closed (`ok == false` / skip / error) instead of wrapping.
- Handle `Redirect().To` errors in `pkg/route` authorize redirect.
- Pass `-exclude-dir=internal/store/ent/gen` to gosec so generated Ent code is not parsed on Windows. Generated code remains out of scope; this is an extra exclusion beyond `-exclude-generated` for the parser bug.

## Alternatives considered

- **`#nosec G115` on each cast.** Rejected: real overflow paths in param parsing and pre-epoch TOTP times are better rejected than silenced.
- **Globally exclude G115.** Rejected: would hide unsafe truncations elsewhere.
- **Rely only on `-exclude-generated`.** Rejected: still hits the Windows parse failure on this tree.

## Consequences

- Oversized unsigned params to `IntParam` / `Int64Param` return `ok == false`.
- TOTP `CodeAt` errors if the time step is negative; `VerifyTOTP` skips negative steps in the ±1 window.
- Gosec no longer walks `internal/store/ent/gen`; regenerate still goes through `go tool task ent`, not gosec.

## Verification

```bash
go tool task gosec
go test ./pkg/utils/ ./pkg/capability/ ./pkg/webauth/ ./pkg/route/ ./pkg/views/partials/
```
