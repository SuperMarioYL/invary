# Changelog

All notable changes to Invary are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
