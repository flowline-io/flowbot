# Agent Note: Skip Windows App Execution Aliases when spawning host commands

Status: implemented

## Problem

On Windows, `python` on `PATH` often resolves first to `%LOCALAPPDATA%\Microsoft\WindowsApps\python.exe`. That file is a zero-byte App Execution Alias (a reparse point), not an interpreter. `os/exec` still treats it as the match. `CreateProcess` on it exits 9009, so `run_code` and function entrypoints fail even when a real `python.exe` appears later on `PATH`. The same alias pattern exists for `python3` and `py`.

## Decision

`OSExecutionEnv` resolves a bare argv0 on Windows before `exec.Command`. The search walks `PATH` with `PATHEXT`, skips directories named `WindowsApps`, and skips non-regular or empty files. The first remaining regular non-empty file is executed by absolute path. If none exists, the original name is passed through so `os/exec` lookup is unchanged. Explicit paths are not rewritten. Sandbox exec is unchanged: container argv stays the bare name `python`.

## Alternatives considered

- **Reorder `PATH` or disable the App Execution Alias on the developer machine.** Fixes one workstation. Every other Windows host with the store stub in front of a real interpreter still fails.
- **Resolve `python` to a host absolute path inside `pkg/exec` `interpreterCommand`.** Function and agent runs that use the sandbox would then try to execute a Windows path inside the container.
- **Invoke `py -3` on Windows.** `WindowsApps\py.exe` is the same kind of stub, and the sandbox image is invoked as `python`, not `py`.

## Consequences

Host `run_code` / entrypoint processes whose bare name is shadowed by a WindowsApps alias use the later real binary. Commands that exist only as a working WindowsApps alias still go through `os/exec`. Non-Windows hosts and sandbox argv are unchanged.

## Verification

`TestLookupExecutableSkipsWindowsAppsAndEmptyFiles`, `TestLookupExecutableUsesFirstRealFile`, and `TestLookupExecutableMisses` in `pkg/agent/env`. `TestRunCode_Python` and `TestRunEntrypoint/python_with_stdin_JSON` in `pkg/exec` run the host interpreter.
