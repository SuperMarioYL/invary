# Changelog

All notable changes to Invary are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
