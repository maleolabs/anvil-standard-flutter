# Verification — Flutter

The Verification part of the standard (ADR-021 §3.2): the Flutter
verification rules — structural checks and lifecycle-conformity checks.
This document is the human-readable form of the executable checks in
[`internal/flutter/verification.go`](../internal/flutter/verification.go);
the Go code is the single source of truth.

The Verification part carries the framework's verification rules: the
checks a release must pass, specific to the framework's requirements
([ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/architecture/006-b-seven-part-standard-structure.md);
`006-b §3.3`).

## Declared checks

The standard declares six verification checks in its capability
declaration. The runtime invokes only declared checks (`verify` command)
during artifact verification; each check inspects the artifact path — an
extracted directory or an Anvil artifact archive (tar.gz) — and reports a
pass/fail outcome (`name`, `passed`, `details`).

### Structural checks (preserved v1.x surface, TS-P7-25)

| # | Check | Validates | Rule |
|---|---|---|---|
| 1 | `pubspec_yaml` | `pubspec.yaml` exists in the artifact root | The Flutter project manifest is present |
| 2 | `lib_directory` | `lib/` directory exists in the artifact | The Flutter application source directory is present |

### Lifecycle-conformity checks (TS-018-03-02, ADR-033 §3, Review 19 §3.3)

The lifecycle-conformity checks verify the hybrid deployment model's
lifecycle behavior (ADR-016), at the convention depth of the four Review
19 §3.3 items, adapted to the model: the locked dependency set is the
hybrid shared resource, dependency resolution timing relative to
promotion is the hybrid analog of migration timing, and the platform
step carries the second lifecycle point (the hybrid model declares no
server-side queue, so the queue-restart item does not apply — there is
nothing to restart or warm, lifecycle/README.md §Semantics).

| # | Check | Validates | Rule |
|---|---|---|---|
| 3 | `dependency_lockfile` | `pubspec.lock` exists in the artifact root | The release's locked dependency set — the hybrid shared resource — is wired: activation's `pub_get` re-resolves exactly the set the artifact was built from |
| 4 | `dependency_timing` | `pubspec.yaml` **and** `pubspec.lock` exist, and the locked set covers every dependency declared in the manifest | Dependency resolution can run at the declared timing — before promotion — and reproduce the built set, not resolve a different one (the hybrid analog of Laravel's migration timing) |
| 5 | `platform_sync_ready` | When the release contains an `ios/` directory, `ios/Podfile` exists | The platform step (`platform_sync`, `pod install`) can run at its declared lifecycle point; a release without `ios/` matches the phase's informational no-op and passes with that note |
| 6 | `rollback_behavior` | Every activation phase declares rollback coverage; the rollback command surface derived from the phase table stays the declared single re-resolution | Rollback produces the declared state — **standard-internal coherence as a drift guard**: `pub_get` rollback is the idempotent re-resolution (`flutter pub get`); `platform_sync` is irreversible — informational, never blocks rollback (TS-P7-10 AC-2). The check verifies the standard's own declared surface stays coherent; it does **not** read the artifact's embedded manifest (ADR-017) |

Both check categories accept either a directory (the extracted artifact)
or an Anvil artifact archive (tar.gz; entries scanned directly, with the
optional `app/` deployable-content prefix stripped).

## Semantics

- **All-of vs any-of.** All checks require every listed path or property
  (all-of); there is no any-of check in this standard.
- **Archive handling.** For artifact archives, entries are scanned
  directly — no full extraction is performed. Anvil artifact archives
  store deployable content under the `app/` prefix; both prefixed and
  unprefixed entries are accepted so plain directories and archives behave
  consistently.
- **Directory checks in archives.** A directory is present when an
  explicit directory entry exists or when a regular entry lives beneath
  the directory path (packaging stores only regular files).
- **Evidence is re-checkable.** The lifecycle-conformity checks read
  evidence embedded in the artifact — the locked dependency set, the
  manifest/lock pair, the platform integration input, the declared phase
  table — so any consumer can re-run a check and derive the same outcome
  (verification-contract.md §5; a claim is not evidence).
- **Unknown checks.** An undeclared check name reports a failed outcome
  with an explanatory message — the runtime never invokes undeclared
  checks (declared-check enforcement is the runtime's responsibility).

## Known limitations

- `dependency_lockfile` and `dependency_timing` read the standard's
  declared evidence shapes: `pubspec.yaml` (two-space-indented
  `dependencies:` / `dev_dependencies:` entries) and `pubspec.lock`
  (two-space-indented `packages:` entries), the canonical shapes
  `flutter create` and `pub get` write. A manifest that heavily
  customizes the indentation shape yields no extractable entries — the
  check fails closed rather than guessing; a manifest that declares no
  dependencies at all passes with a note (nothing to re-check).
- `dependency_timing` coverage is **name-level**: it verifies every
  declared dependency name is present in the locked set. Version
  conformity (the locked version satisfying the declared constraint) is
  not verifiable from the artifact without build-time evidence — the
  runtime's verification report should be paired with build outputs for
  version-level claims.
- `platform_sync_ready` verifies **readiness** — the platform step
  (`pod install`) can run at its declared lifecycle point because the
  release carries `ios/` with its `ios/Podfile` input. It does not
  verify post-step execution evidence (the step ran successfully);
  execution outcomes are the activation contract's record, and the
  runtime's verification report should be paired with activation
  outcomes for execution-level claims.
- `platform_sync_ready` uses the fixed platform input path `ios/Podfile`;
  a project that relocates the Podfile outside the canonical path is not
  re-checkable from the artifact.
- `rollback_behavior` is a standard-internal coherence check: it
  verifies the standard's own declared rollback surface stays coherent
  (a drift guard). It does not read the artifact's embedded manifest
  (ADR-017) — artifact-manifest rollback metadata is the runtime's
  manifest contract surface.

## Contract rules

- Checks are declared in the capability declaration; the runtime invokes
  **only declared checks** (TS-P7-08 AC-3). Undeclared checks are never
  called.
- Gate semantics and evidence requirements belong to the specification,
  not to this standard (ADR-033): the standard adds checks, it never
  weakens gates — the four lifecycle-conformity checks are additive to
  the preserved structural surface.
- Outcomes merge into the runtime's verification report as lifecycle
  evidence (007 §6) in the standard outcome shape, without
  transformation.

## Implementation

The checks live in `internal/flutter/verification.go` and are executed
by the standard's `verify` command (`cmd/flutter-adapter`), exchanged
over the standard command contract.
