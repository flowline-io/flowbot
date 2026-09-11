---
name: New provider / capability
description: Add or upgrade a third-party integration
title: "[provider] "
labels: ["enhancement", "provider"]
---

**Service**

- Name / website:
- Auth model: API token / OAuth / none
- Already discovery-only under `pkg/providers/`? Yes / No / Unknown

**Scope requested**

- [ ] Provider client only (`pkg/providers/<name>`)
- [ ] Full capability (`pkg/capability/<name>` + `capability.Invoke`)
- [ ] Webhook and/or poller event source
- [ ] CLI / skills docs
- [ ] Web UI management surface

**Priority operations**
List the first operations you need (e.g. list, get, create). Keep the first PR small.

**References**
API docs URL, OpenAPI/Swagger if any.

**Contributor interest**

- [ ] I plan to implement this (please point me at `example/` packages)
- [ ] Looking for a maintainer or volunteer

Follow [CONTRIBUTING.md](https://github.com/flowline-io/flowbot/blob/master/CONTRIBUTING.md) and the reference packages:

- `pkg/providers/example`
- `pkg/capability/example`
