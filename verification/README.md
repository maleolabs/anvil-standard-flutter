# Verification — Flutter Structural Checks

The Verification part carries the framework's verification rules: the
checks a release must pass, specific to the framework's requirements
([ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/architecture/006-b-seven-part-standard-structure.md);
`006-b §3.3`).

This standard supplies **structural checks** — the verified v1.x
surface, preserved (TS-P7-25). Lifecycle-conformity checks (the v2
verification depth) are authored in EPIC-018 scope (TS-018-03-02).

## Declared checks

| Check | What it validates |
|---|---|
| `pubspec_yaml` | `pubspec.yaml` exists in the artifact root — the Flutter project manifest |
| `lib_directory` | `lib/` exists in the artifact — the Flutter application source directory |

Both checks accept either a directory (the extracted artifact) or an
Anvil artifact archive (tar.gz; entries scanned directly, with the
optional `app/` deployable-content prefix stripped).

## Contract rules

- Checks are declared in the capability declaration; the runtime invokes
  **only declared checks** (TS-P7-08 AC-3). Undeclared checks are never
  called.
- Gate semantics and evidence requirements belong to the specification,
  not to this standard (ADR-033): the standard adds checks, it never
  weakens gates.
- Outcomes merge into the runtime's verification report as lifecycle
  evidence.

## Implementation

The checks live in `internal/flutter/verification.go` and are executed
by the standard's `verify` command (`cmd/flutter-adapter`), exchanged
over the standard command contract.
