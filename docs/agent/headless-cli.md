# Headless CLI (`flowbot-agent`)

Local headless coding agent inspired by [Cursor Headless CLI](https://cursor.com/docs/cli/headless). Binary: `cmd/agent` → `bin/flowbot-agent`.

- Runs the Observe-Think-Act loop and coding tools **locally**
- Calls Flowbot `POST /agent/v1/chat/completions` for LLM (server holds provider keys)
- Intended as a drop-in for gateway's `cursor_binary` flag contract

## Build

```bash
go tool task build:agent
```

## Token

```bash
go tool task run:composer -- admin token create
# select scope: agent:headless
```

## Config

Copy [`cmd/agent/agent.yaml.example`](../../cmd/agent/agent.yaml.example):

```yaml
flowbot_url: "http://127.0.0.1:6060"
access_token: "..."   # agent:headless
```

Env overrides (higher precedence):

- `FLOWBOT_URL` (also accepts `FLOWBOT_SERVER_URL` for gateway parity)
- `FLOWBOT_AGENT_TOKEN` (also accepts `FLOWBOT_TOKEN`)

## Usage

```bash
flowbot-agent -p --force --trust --workspace /path/to/repo --output-format text "Fix the failing test"
```

| Flag | Behavior |
|------|----------|
| `-p` / `--print` | Required; non-interactive |
| `--force` | Allow write tools + `run_terminal` |
| without `--force` | Read-only tools only |
| `--trust` | Accepted for Cursor CLI compatibility; **no-op in v1** |
| `--workspace` | Workspace root (default cwd) |
| `--output-format` | `text` only (v1) |
| `-config` | Path to `agent.yaml` (default `agent.yaml`) |
| `-log-level` / `-timeout` | Local ops helpers (not part of Cursor gateway contract) |

Final assistant text goes to stdout; diagnostics to stderr / `flog`.

Look for `flowbot-agent run complete` (success) or `run failed` / non-zero exit (failure).

### Streaming note

The server proxy completes the model turn, then emits OpenAI SSE (including `data: [DONE]`) in one burst. Clients still use `stream: true`; this is not token-by-token proxying.

## Gateway

Set in `gateway.yaml`:

```yaml
cursor_binary: flowbot-agent
agent_access_token: "..."   # agent:headless
```

Default `cursor_binary` remains Cursor `agent`. Details: [Local CLI Gateway](./local-cli-gateway.md).
