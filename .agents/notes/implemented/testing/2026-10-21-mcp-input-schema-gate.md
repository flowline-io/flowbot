# Agent Note: MCP input schema gate

Status: implemented

## Problem

Claude Code 2.1.300 drops parameters for the rest of the conversation when the model API rejects a tool input schema. The call then fails with no schema error to point at. Flowbot builds every advertised `inputSchema` from `hub.ParamDef`, so one bad conversion affects the whole catalog. Contract tests checked `outputSchema` only.

## Decision

`pkg/mcp.ValidateInputSchema` applies the conservative check, then compiles the schema with `jsonschema.Draft2020`. The schema object on the wire has no `$schema` keyword. The top-level type is `object`. Every `required` name is a key of `properties` at that object. Every `properties` entry, at any depth, has a `description` and a single-string `type` of `string`, `number`, `integer`, `boolean`, `object`, or `array`. `items` has a `type` and does not need a `description`. A bare `object` or `array` is valid. An `enum` key must be a non-empty array. The whole schema tree, including `items`, rejects `$ref`, `$schema`, `oneOf`, `anyOf`, `allOf`, `not`, `patternProperties`, `unevaluatedProperties`, `if`, `then`, `else`, `dependentSchemas`, and `prefixItems`.

`TestInputSchemaContract` in `internal/server/mcp` runs that check on `BuildCatalog(every CatalogSpec(), nil, nil)`, including the built-in pipeline, workflow, function, hub, and audit tools. `mcp.include` and `mcp.exclude` are not applied. Tools removed by the MCP deny-list are not checked. The test fails when a `pkg/capability/*/register.go` file exports `CatalogSpec` and that spec is not passed in. `TestValidateInputSchema_Schema` in `pkg/mcp` feeds synthetic schemas, including a nil schema, that never enter `BuildCatalog`. `TestInputSchemaContract` and `TestValidateInputSchema` are on the `test:mcp-contract` `-run` list in `taskfile.yaml`. `.github/workflows/mcp-contract.yml` runs that task.

The unsigned client checklist record is the table in [MCP list pages, output cap, and client setup](../feature/2026-10-17-mcp-tool-quality-and-client-setup.md).

## Alternatives considered

- **Require `additionalProperties: false` in the same gate.** Rejected: that is the OpenAI strict direction, and it conflicts with keeping schemas small enough for Claude. Codex behavior stays on the manual checklist.
- **Require a `description` on `items`, and require `items` or `properties` on every array or object.** Rejected: `[]string` parameters and bare `object` / `array` parameters already ship.
- **Reuse the hand-written spec list in `lint_test.go`.** Rejected: that list omits `drone`, and a new `CatalogSpec` would not fail it.
- **Add a second workflow for the gate.** Rejected: MCP contract CI is already `.github/workflows/mcp-contract.yml`, which runs `go tool task test:mcp-contract`. The `-run` whitelist lives in `taskfile.yaml`.
- **Put `$schema` on the advertised schema.** Rejected: `$schema` is one of the forbidden keywords, and the library's default draft is allowed to change. The test compiler pins `Draft2020`.
- **Register a tool with a broken schema and watch Claude Code.** Rejected: that tool would be advertised to every client. `TestValidateInputSchema_Schema` uses schemas that never enter `BuildCatalog`.

## Consequences

A schema that is valid JSON Schema and still uses a forbidden keyword fails CI. `additionalProperties` is unconstrained by this gate. Deny-listed tools can carry a schema the gate does not see, because those tools are not advertised.

## Verification

`go tool task test:mcp-contract` runs `TestInputSchemaContract` and `TestValidateInputSchema_Schema`.
