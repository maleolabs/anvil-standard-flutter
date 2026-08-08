# Manifest — Flutter Delivery Lifecycle Standard

The Manifest part of the standard ([ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/planning/ANVIL_V2_TRANSITION_PLAN.md)
in the Core repository; `006-b-seven-part-standard-structure.md §3.1`;
007 §4): who the standard is, what contract version it targets, its
capability declaration, and its framework-version support scope.

## Identity

| Field | Value | Meaning |
|---|---|---|
| **id** | `anvil-standard-flutter` | Standard identity, stable across releases (first-party naming per ADR-025 §3.1) |
| **name / title** | Flutter delivery lifecycle standard | The standard's human-readable name |
| **version** | `1.0.0` | This release's version on the standard's independent semver line (ADR-021 §3.4) |
| **target contract version** | `1.0.0` | The delivery lifecycle specification contract version this release targets; the major version is the compatibility unit (ADR-024 §3.1) |
| **deployment model** | `hybrid` | Releases are built and packaged for distribution (web bundle, APK, iOS app), not deployed to a server and activated in place (ADR-016; `DeploymentModelHybrid` in `internal/contracts`) |
| **framework-version support scope** | Flutter stable `3.0.0`–`3.32.0` | The Flutter versions this release supports (ADR-021 §3.2; declared in `capability.frameworkVersion`) |

## Capability declaration

The standard executable (`anvil-adapter-flutter`) answers the standard
command contract with the following declaration (`capabilities` command,
`internal/flutter/adapter.go`). The declaration is the standard's
complete lifecycle surface per 007 §4 — the runtime invokes **only
declared capability**; undeclared capability is never called
(TS-P7-08 AC-3).

### Lifecycle phases (activation)

The activation phases run in declaration order, each as
`<program> <args>` from the release's working directory (the hybrid
deployment model — there is no server to activate on):

| # | Phase | Command | Irreversible |
|---|---|---|---|
| 1 | `pub_get` | `flutter pub get` | no — rollback: `flutter pub get` (idempotent re-resolution from the lockfile) |
| 2 | `platform_sync` | `pod install` (iOS platform steps; conditional — applies when the release contains an `ios/` directory and the host is macOS, else an informational no-op) | yes |

- **Dependency resolution before promotion.** `pub_get` runs **before**
  the artifact is promoted — the release serves exactly what its
  `pubspec.lock` declares; an unresolvable dependency set fails
  activation (the hybrid analog of Laravel's migration timing relative
  to promotion, Review 19 §3.3).
- **Rollback semantics.** `pub_get` rollback re-runs `flutter pub get`
  in the restored release's working directory. `platform_sync` is
  **irreversible** — a rollback reports an informational result and
  never blocks on it; the previous release's own activation re-runs its
  platform steps. Full per-phase failure and rollback semantics:
  [lifecycle/README.md](../lifecycle/README.md).

### Build phases

`web`, `apk`, `ios` — the declared build phases in build execution order
(`internal/flutter/build.go` — the build target table is the single
source of the BuildPhases declaration): `flutter build web`,
`flutter build apk --release`, `flutter build ios --release`, each with
its ADR-018 platform metadata. Dependency resolution (`flutter pub
get`) is an **activation phase** (`pub_get`) and the template's
`dependencies` stage — it is not a declared build phase; see
Templates below.

### Verification checks

`pubspec_yaml`, `lib_directory` — the structural verification rules
([verification/README.md](../verification/README.md)).

### Config extensions

`framework.flutter.targets`, `framework.flutter.build_args` — declared
by the `extension` command and validated by the `validate` command under
the `framework.flutter.` namespace (ADR-005 §4.4;
[templates/README.md](../templates/README.md)).

### Templates

The Flutter build pipeline definition (`build.yaml` — a `dependencies`
stage running `flutter pub get`, then the `build` stage with the
web → apk → ios targets; returned through the `template` command,
`internal/flutter/template.go`) and the generic CI scaffold
(`ci.yaml`) ([templates/README.md](../templates/README.md)).

### Deployment model

`hybrid` — releases are built and packaged for distribution; activation
runs the release's dependency set and platform steps on the release
working directory (ADR-016).

## Command surface

The standard executable implements the full standard command contract:
`capabilities`, `build`, `activate`, `verify`, `extension`, `validate`,
`template`, `manifest`. The subprocess contract — JSON payload as a
single argument, JSON result on stdout, exit-code convention — is
preserved unchanged from the pre-split adapter contract (ADR-025 §3.4,
§12.2).

## Machine-readable manifest

The machine-readable manifest is
[`standard/manifest.json`](manifest.json). It follows the registry
metadata format conventions the Anvil Runtime registry client reads
(`registry-metadata.schema.json`, `internal/registry/metadata.go` in the
Core repository): the same field names, semver patterns, and capability
shape.

| Field | Value | Meaning |
|---|---|---|
| `id` | `anvil-standard-flutter` | Standard identity, stable across releases |
| `version` | `1.0.0` | This release's version on the standard's independent semver line |
| `contractVersion` | `1.0.0` | The delivery lifecycle specification contract version this release targets; the major version is the compatibility unit (ADR-024 §3.1) |
| `capability.frameworkVersion` | Flutter stable `3.0.0`–`3.32.0` | The framework-version support scope: the Flutter versions this release supports |

## Release-time fields

The registry metadata format additionally carries distribution,
lifecycle, and trust sections (ADR-030, ADR-022). Those fields describe
a **published release** — the release asset location, the governed
lifecycle state, and the content digests/attestation — and are populated
by the standard's release pipeline when a versioned artifact is
published to the registry (EPIC-016 scope, TS-016-03-02).

> **Parseability note.** Because this source manifest intentionally
> carries authoring-time fields only, it is **NOT a valid registry
> metadata document** for the Core registry client's strict parser
> (`registry-metadata.schema.json` requires distribution, lifecycle, and
> trust). Do not attempt to parse `standard/manifest.json` with the
> registry parser — it becomes parseable only after TS-016-03-02 fills
> the release-time fields at publication time. This README is the single
> source of truth for the source manifest's semantics; the published
> registry metadata document is generated from this manifest plus the
> release-time fields.

## Versioning

The standard versions independently from the Core runtime (ADR-021
§3.4): standard releases and runtime releases are decoupled, and every
release declares the contract version it targets and its
framework-version support scope (see
[compatibility/README.md](../compatibility/README.md)). A manifest
change is a release event: bump `version` for any content change, and
re-declare `contractVersion`/`capability` whenever the standard's
compatibility surface changes (a standard release never breaks the
contract it declares — ADR-021 §3.4).
