# Lifecycle Definition — Flutter

The Lifecycle Definition is the framework's lifecycle content: what a
legal lifecycle *contains* for one framework ([ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/architecture/006-b-seven-part-standard-structure.md);
`006-b §3.2`). This document is the human-readable form of the
executable activation phase table in
[`internal/flutter/activation.go`](../internal/flutter/activation.go) —
the Go code is the single source of truth; this document is the
adopters' reference.

## Deployment model: hybrid

The Flutter standard implements the **hybrid deployment model**
(ADR-016): releases are built on a build host and deployed to targets
(web bundle, APK, iOS app) rather than deployed to a server and
activated in place (EPIC-007 §7.3). The activation phases reflect that
model — the build artifact's dependency set and the platform steps of
the native targets — not server-side steps; there is no server to
activate on.

## Activation phases

Release activation runs the declared activation phases in declaration
order from the release's working directory (the release context
`working_dir` passed by the runtime):

| # | Phase | Command | Reversible |
|---|---|---|---|
| 1 | `pub_get` | `flutter pub get` | ✅ rollback: `flutter pub get` (idempotent re-resolution) |
| 2 | `platform_sync` | `pod install` (when the release contains an `ios/` directory, on macOS hosts) | ❌ irreversible |

### Semantics

- Each phase reports a structured outcome (`success`, `output`, `error`)
  through the activation contract; the JSON result is authoritative for
  the phase outcome.
- **Convention depth where applicable.** The server-model depth items of
  the Laravel standard — queue restart, cache warming order — do not
  apply to the hybrid model: Flutter releases hold no server-side queue,
  cache, or database state, so there is nothing to restart or warm.
  The hybrid depth items are dependency resolution timing relative to
  promotion (`pub_get` before promotion) and platform step ordering
  (`platform_sync` after `pub_get`).
- **Dependency resolution before promotion.** `pub_get` runs **before**
  the artifact is promoted: the artifact was built from the release's
  locked dependency set on the build host, and activation re-resolves it
  so the release serves exactly what its `pubspec.lock` declares. This
  is the hybrid analog of Laravel's migration timing relative to
  promotion (Review 19 §3.3) — promoting an artifact with an
  unresolvable dependency set breaks the release.
- **Platform steps in declared order.** `platform_sync` runs after
  `pub_get`, on the resolved dependency set. It applies only when the
  release working directory contains an `ios/` directory (no CocoaPods
  state otherwise) and on macOS hosts only (CocoaPods is a macOS tool;
  the iOS platform finalization belongs to the darwin build host) —
  platform-aware execution mirroring the build side (ADR-018). When it
  does not apply, the phase reports an informational no-op
  (`Success=true`, no command run) — absent platform state is not a
  failure.
- **Irreversible phase.** `platform_sync`'s effects cannot be undone by
  a rollback — the previous release's platform integration cannot be
  restored because the previous release's own activation re-runs its
  platform sync. A rollback request on an irreversible phase returns an
  informational success that does **not** block the rollback.

## Failure semantics

| Phase | On failure |
|---|---|
| `pub_get` | Activation stops. A release with an unresolvable dependency set cannot be activated; the result reports the resolution error details. |
| `platform_sync` | Activation stops. A broken platform step means the artifact cannot serve its native target; the result reports the step error details. |
| Rollback operations | A failing rollback operation (e.g. `pub_get` rollback) reports the failure; the JSON result is authoritative for the operation outcome. |

## Rollback semantics

| Operation | Behavior |
|---|---|
| `pub_get` rollback | Re-runs `flutter pub get` in the restored release's working directory — resolution is idempotent and lockfile-driven, so re-running it re-establishes the previous release's locked dependency state |
| Irreversible phase rollback (`platform_sync`) | Informational success, no command executed, rollback proceeds — irreversibility never blocks rollback (007 §5) |
| Rollback order | Reverse activation order is the orchestrator's responsibility; the phase table provides each phase's own rollback operation |

## Manifest command metadata

The `manifest` command returns the full command strings stored in the
artifact manifest at packaging time (ADR-017) and executed by the
orchestrator during release activation and rollback:

- **Activation commands:** `flutter pub get`, `pod install` (in
  execution order)
- **Rollback commands:** `flutter pub get`

The `pod install` entry is conditional by nature — it applies when the
release contains an `ios/` directory and the host is macOS; the metadata
records the command, the executable phase table carries the conditions.
The rollback metadata carries only the reversible phase's operation —
irreversible phases reverse nothing.

## What the lifecycle contains for Flutter

| Lifecycle position | Content |
|---|---|
| Activation | The hybrid model's declared activation phases: `pub_get` (dependency resolution, before promotion) and `platform_sync` (platform steps, after), with per-phase failure and rollback semantics |
| Build | `flutter build web`, `flutter build apk --release`, `flutter build ios --release` in declared order (web → apk → ios), with platform metadata (ios is darwin-only) |
| Verification | Structural checks of the built artifact (`pubspec.yaml` present, `lib/` directory present) plus lifecycle-conformity checks of the hybrid model (TS-018-03-02): shared-resource wiring (`dependency_lockfile`), dependency-resolution timing (`dependency_timing`), platform step readiness (`platform_sync_ready`), rollback behavior (`rollback_behavior`) |
| Configuration | `framework.flutter.*` keys (targets, build_args) with value validation |
| Templates | The build and CI pipeline definitions generated at project init |

The stage model, phase sequence, and transition rules are owned by the
delivery lifecycle specification; this standard supplies content within
the defined lifecycle — it does not invent a lifecycle (007 §5).
