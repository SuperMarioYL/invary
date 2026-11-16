# Changelog

All notable changes to Invary are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-11-17

### Fixed

- `invary check` scored only the first tool call in a response. Parallel tool
  calls are the normal OpenAI-compatible shape, so a trace whose second call
  broke the contract still printed `4 pass / 0 fail` and exited 0. Every tool
  call in the trace is now evaluated; multi-call traces label each section
  (`call N/M`), close with `N of M tool calls breached a contract`, and fail the
  gate if any call breaches. The single-call report is unchanged.
- Schema parsing accepted any JSON object as an empty contract: `--schema "{}"`
  or a document with no function definition parsed successfully, then
  `required_fields` and `arg_types` passed vacuously while `no_schema_drift`
  reported every real argument as undeclared. A schema that declares no function
  is now rejected with the accepted shapes named, while the OpenAI tools array
  and a chat-completions request body (`{"tools":[…]}`) are accepted and resolve
  to their first tool — the tools array used to fail with a raw Go type error.
- A chat-completions response in which the model answered with content instead
  of calling the tool was reported as an unrecognized file shape ("expected a
  chat-completions response"). It now reports that the response carries no
  tool call because the model answered with content, including the
  `finish_reason` when recorded.

### Changed

- Documentation no longer contradicts the binary: the website stopped calling
  `init` a milestone stub (it has worked since 0.2.0), the starter
  `invariants.yaml` no longer promises custom invariant loading as a "v0.2+"
  feature that never shipped, and the limits sections describe the new
  every-tool-call evaluation.
- Version surfaces bumped in lockstep to 0.4.0 (`VERSION`, `invary --version`,
  this changelog, the website's `meta.content_version` and the checked-in
  `dist/` release archives), all covered by the version-consistency test.

## [0.3.0] - 2026-10-14

### Fixed

- The GitHub release for a tag shipped zero downloadable assets: no release
  workflow existed, so the "download a release binary" install path was dead.
  A goreleaser release workflow now builds and attaches per-platform archives
  (`darwin`/`linux`/`windows` × `amd64`/`arm64`, tar.gz with zip for windows)
  plus a sha256 `checksums.txt` on every `v*.*.*` tag push; the `VERSION` file
  is packed into each archive and the binary version is injected via
  `-X …cmd.Version` so artifacts and `invary --version` agree.
- `invary check` on a trace file that is not valid JSON (e.g. a truncated
  copy-paste from provider logs) reported a false "trace has no tool_calls"
  diagnosis instead of the JSON syntax error. `invary check` now reports
  `trace is not valid JSON: …` — the same root-causing the schema path already
  applied. Valid-but-unrecognized JSON keeps the previous shape error.

### Changed

- Version surfaces bumped in lockstep to 0.3.0 (`VERSION` file,
  `invary --version`, this changelog) and the website (`web/site.json`) now
  carries a `meta.content_version` surface too, covered by the same
  version-consistency test.

## [0.2.0] - 2026-09-10

### Fixed

- CLI errors printed the message twice on stderr (cobra's `Error: …` line plus
  a second raw line from the executor). Errors now print exactly once.
- `invary check` could print invalid UTF-8 in the `args:` preview line when the
  arguments exceeded 80 bytes and contained multi-byte characters (e.g. a
  Chinese city name); the preview now truncates on a rune boundary.
- A conforming trace saved in the minimal `{"arguments": "{\"…\"}"}` wire shape
  (arguments as a JSON string, exactly as they appear in an OpenAI-compatible
  response) was falsely reported as `1 pass / 3 fail` ("arguments parsed to
  string"); the one-layer unwrap already used for `tool_calls` traces now
  applies to the minimal shape too, so the same arguments evaluate identically
  in every accepted trace shape.

### Added

- `invary init`: writes an `invariants.yaml` + `tool-schema.json` starter set
  into the current directory (the README-documented m3 behavior; previously
  the command returned a "not available" error). It never overwrites existing
  files.

### Changed

- Version surfaces bumped in lockstep to 0.2.0 (`VERSION` file and
  `invary --version`).
- Repo-wide `gofmt` pass (5 files shipped unformatted in 0.1.0; no behavior
  change).

## [0.1.0] - 2026-08-14

### Added

- m1: the four built-in contract invariants (`valid_json`, `required_fields`,
  `arg_types`, `no_schema_drift`) and the offline single-trace evaluator
  `invary check --trace <file> --schema <file>`, plus per-provider sample
  traces under `examples/`.
