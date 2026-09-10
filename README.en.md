[简体中文](./README.md) · [Website](https://invary.lei6393.com) · [GitHub](https://github.com/SuperMarioYL/invary)

<picture>
  <source media="(max-width: 600px) and (prefers-color-scheme: dark)" srcset="./assets/presentation/hero-mobile-dark.svg">
  <source media="(max-width: 600px)" srcset="./assets/presentation/hero-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="./assets/presentation/hero-dark.svg">
  <img src="./assets/presentation/hero-light.svg" width="960" alt="Hero diagram">
</picture>

# invary

**Find tool-call contract breaks in saved responses.**

Invary checks a captured tool call against four explicit invariants and reports the failing evidence locally.

## Why use it

A response can look plausible yet omit a required argument, use the wrong type, or invent a field. A saved response and tool schema make these failures reproducible without another model request.

- **Offline reproduction** — A saved trace and schema are sufficient.
- **Specific failure evidence** — Each invariant names the violated contract.
- **Scriptable gate** — Failed checks return a nonzero status.

## Architecture

<picture>
  <source media="(max-width: 600px) and (prefers-color-scheme: dark)" srcset="./assets/presentation/architecture-mobile-dark.svg">
  <source media="(max-width: 600px)" srcset="./assets/presentation/architecture-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="./assets/presentation/architecture-dark.svg">
  <img src="./assets/presentation/architecture-light.svg" width="960" alt="Architecture diagram">
</picture>

The CLI loads trace and schema JSON, selects the first tool call, and evaluates valid_json, required_fields, arg_types and no_schema_drift. The report prints evidence per rule. Any failed rule, including a warning-level drift, returns a nonzero exit.

| Component | Responsibility |
| --- | --- |
| `Trace + schema` | cmd/check.go |
| `Schema parser` | internal/invariant |
| `Four invariants` | internal/invariant/checks.go |
| `Evidence table` | CLI stdout and exit code |

## Install and quickstart

Build with the version declared in the repository manifest. Run the example from the repository root.

```bash
git clone https://github.com/SuperMarioYL/invary.git
cd invary
go build .
```

The script checks two complete bundled synthetic traces and verifies the expected success and rejection exit codes.

```bash
python3 examples/presentation-demo.py
```

## Recorded demo

<picture>
  <source media="(max-width: 600px) and (prefers-color-scheme: dark)" srcset="./assets/presentation/process-mobile-dark.svg">
  <source media="(max-width: 600px)" srcset="./assets/presentation/process-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="./assets/presentation/process-dark.svg">
  <img src="./assets/presentation/process-light.svg" width="960" alt="Process diagram">
</picture>

One fixture passes all four checks; the other fails no_schema_drift on the extra timezone field.

```text
trace:   examples/sample_deepseek_output.json
schema:  examples/tool-schema.json
tool:    get_weather
args:    {"location":"Tokyo","unit":"celsius"}

INVARIANT        SEV      STATE  EVIDENCE
valid_json       error    ✓ pass -
required_fields  error    ✓ pass -
arg_types        error    ✓ pass -
no_schema_drift  warn     ✓ pass -

4 pass / 0 fail
command exit: 0
trace:   examples/sample_glm_output.json
schema:  examples/tool-schema.json
tool:    get_weather
args:    {"location":"Tokyo","timezone":"Asia/Tokyo"}

INVARIANT        SEV      STATE  EVIDENCE
valid_json       error    ✓ pass -
required_fields  error    ✓ pass -
arg_types        error    ✓ pass -
no_schema_drift  warn     ✗ FAIL undeclared argument field(s): timezone

3 pass / 1 fail
command exit: 1
```

The complete command and output are recorded in [docs/demo-results.json](./docs/demo-results.json). Inputs and reproduction code are included in the repository.

![Existing terminal recording](./assets/demo.gif)

The existing recording is retained for context; the text example above documents the reproducible scenario.

## Usage

The CLI exposes the following operations. Commands after the example use your own paths or identifiers.

```bash
go run . check --trace examples/sample_deepseek_output.json --schema examples/tool-schema.json
go run . check --trace examples/sample_glm_output.json --schema examples/tool-schema.json
go run . init
```

`invary init` writes an `invariants.yaml` and `tool-schema.json` starter set (the same files as `examples/`) into the current directory; it refuses instead of overwriting existing files.

## Configuration

--trace and --schema are required for check. The four rules are built in and no API key or configuration file is needed. Warn severity affects triage; it does not make a failed rule pass the gate.

The `invariants.yaml` written by `invary init` (since v0.2.0) documents the invariant DSL shape for customizing the contract set; `check` itself does not read that file.

## Integrations and responsibilities

<picture>
  <source media="(max-width: 600px) and (prefers-color-scheme: dark)" srcset="./assets/presentation/integrations-mobile-dark.svg">
  <source media="(max-width: 600px)" srcset="./assets/presentation/integrations-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="./assets/presentation/integrations-dark.svg">
  <img src="./assets/presentation/integrations-light.svg" width="960" alt="Integrations diagram">
</picture>

The following routes are implemented in the source. Choose the input that matches your task and keep the resulting artifact with your project.

| Route | Implemented role |
| --- | --- |
| Captured JSON | Full response or tool-call object |
| Tool schema | Declared properties and required fields |
| Terminal report | Rule verdicts and evidence |
| Exit status | Pipeline failure signal |

## Limits and next steps

- check evaluates only the first tool call in a response. It is not a complete JSON Schema implementation.
- run remains a milestone stub; this release does not perform a live multi-provider differential run. init (since v0.2.0) works locally.
- Provider names in fixtures identify example files, not measured provider reliability.

Live multi-provider comparison and configurable invariant loading are future milestones. Saved-response checking is the implemented workflow.

## License and contributions

See [LICENSE](./LICENSE). When reporting an issue, include a minimal input, the command, and the observed output.
