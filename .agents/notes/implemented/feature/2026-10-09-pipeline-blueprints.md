# Agent Note: Pipeline blueprints

Status: implemented

## Problem

New operators who have connected apps do not have a safe, parameterized way to start a pipeline. The repo only shipped a handful of workflow examples and a pipeline schema file. Home Assistant Blueprints and Huginn Scenarios succeed as shareable templates with inputs; Flowbot had no equivalent, and copying YAML required forking the graph immediately.

## Decision

v1 covers **pipeline** templates only. A blueprint is a wrapper YAML (`kind: pipeline_blueprint`) with `id`, `requires.capabilities`, typed `inputs`, and a `definition` body that uses `{{inputs.*}}`. The engine still runs a materialized `EditorDefinition`.

The Web UI is a **template library** (official embed + user paste/upload). Instantiate creates a linked pipeline row (origin + input values + unexpanded YAML). Linked instances may change inputs and the enable flag; steps stay read-only until irreversible Take control. Catalog updates never apply until the instance owner confirms. Official templates cannot be deleted; imported templates cannot be deleted while instances remain. There is no remote URL/GitHub/Gist fetch.

Input types are `string`, `number`, `boolean`, and `notify_channel`. Capability/operation are authored, not selected. Missing required capabilities keep the card visible and disable instantiate.

Official catalog lives in `pkg/pipeline/blueprints/` and is embedded in the binary.

## Alternatives considered

- **Pipeline and workflow in v1.** Rejected: the cold-start scenarios are event/cron pipelines; workflow already has `inputs`.
- **Stamp-only import with no origin.** Rejected: loses Take control and explicit re-import.
- **Linked instances that may edit YAML.** Rejected: re-import becomes a three-way merge.
- **Runtime fetch of official templates from GitHub.** Rejected: catalog must work offline and must not depend on GitHub.
- **Arbitrary URL import.** Rejected: SSRF and supply-chain risk. Paste/upload matches existing `pipeline apply --file` trust.
- **Capability-selector inputs.** Rejected: CapType is the provider ID; swapping would need traits.

## Consequences

- `flowbot pipeline apply` rejects blueprint documents and refuses to overwrite a linked instance (take control first).
- Linked pipelines use a dedicated instance page instead of the YAML editor (inputs + enable; steps read-only).
- Users who need a different capability graph Take control or import a second YAML with a new `id`.
- Adding an official template is a PR that adds a file under `pkg/pipeline/blueprints/`.

## Verification

- `go test ./pkg/pipeline/ -run 'TestParseAndMaterialize|TestLooksLikeBlueprint|TestLoadBuiltin|TestApplyYAMLRejectsBlueprint|TestApplyYAMLRejectsLinkedInstance|TestBindRequiredInput|TestBlueprintServiceInstantiate'`
- `go test ./internal/store/ -run TestPipelineBlueprintTemplateCRUD`
- `go test ./internal/modules/web/ -run 'TestBlueprintListPage|TestBlueprintDownloadOfficial|TestWebserviceRules'`
- `go test ./cmd/cli/command/ -run TestAllLeafCommandsHaveRunE`
- BDD (Docker): `tests/specs/blueprint_page_spec_test.go`
