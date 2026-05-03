# User Guide

Core concepts and usage guides for Flowbot's orchestration engines.

## Contents

- [Pipeline Engine](./pipeline.md) — Event-driven multi-step automation with retry and checkpointing
- [Pipeline Template Engine](./pipeline-template.md) — Go `text/template`-based parameter rendering with conditionals, loops, and FuncMap
- [Workflow Engine](./workflow.md) — YAML-defined task DAGs with capability invocation, shell commands, Docker, and remote machines
- [Notifications](./notifications.md) — Multi-channel notification configuration (Slack, Pushover, ntfy, Message Pusher)

## Concepts

Flowbot operates three runtime engines:

1. **Pipelines** react to `DataEvent` messages published via Redis Stream. Each pipeline consists of a trigger and ordered steps that invoke capability operations.

2. **Workflows** execute task DAGs defined in YAML. Tasks can invoke capabilities, run Docker containers, execute shell commands, or connect to remote machines.

3. **Notifications** deliver messages across multiple channels using a unified provider interface.

For architecture details, see [Architecture](../architecture/README.md).
