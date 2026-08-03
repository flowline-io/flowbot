# Regression eval fixtures

YAML cases are loaded by `BuiltinRegressionScenarios` / `LoadScenariosFromDir`.

- FakeModel scripts drive the agent loop (proxy outcomes + grader wiring).
- `fixtures:` seed files into an isolated workspace before the run.
- Toolsets: `echo`, `none`/`text` (no tools), `read_file`, `write_file`, `fs` (read+write+echo).
