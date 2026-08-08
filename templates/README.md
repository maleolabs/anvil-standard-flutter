# Templates — Generated Content

The Templates part carries the generated content a project receives from
the standard at init time ([ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/architecture/006-b-seven-part-standard-structure.md);
`006-b §3.4`; 007 §7).

## Build pipeline template

[`build.yaml`](build.yaml) is the committed copy of the build pipeline
definition the standard returns through the `template` command: the
Flutter build steps in execution order with their ADR-018 platform
metadata. The runtime validates it through the pipeline loader and
writes it to `.anvil/pipelines/build.yaml` at generation time.

The pipeline covers the Flutter hybrid build steps at the same
convention depth as the Laravel template (TS-018-02-02; Review 19 §3.3):

| Stage | Task | Command | Purpose |
|---|---|---|---|
| `dependencies` | `flutter-pub-get` | `flutter pub get` | Resolves the package graph before any build can run (the same position Laravel's `composer install` holds); not a build target — no ADR-018 metadata, no timeout |
| `build` | `flutter-web` | `flutter build web` | Web bundle; platforms linux, darwin, windows; target `web`; timeout 10m |
| `build` | `flutter-apk` | `flutter build apk --release` | Android APK; platforms linux, darwin, windows; target `apk`; timeout 15m |
| `build` | `flutter-ios` | `flutter build ios --release` | iOS app; platform darwin (Xcode); target `ios` |

The build stage tasks mirror the standard's build target table
(`internal/flutter/build.go`) — the single source of build knowledge —
so the local engine keeps its platform-aware execution (skip unsupported
targets with a warning; `--target` selection; `--strict` failure). The
explicit timeouts (10m web, 15m apk) are preserved: Flutter builds (first
Gradle run in particular) routinely exceed the engine's 5-minute default.

## CI pipeline scaffold

[`ci.yaml`](ci.yaml) is the committed copy of the CI scaffold (build +
test placeholder stages) the standard returns alongside the build
definition. The CI pipeline is generic placeholder data, not framework
knowledge (ADR-026 decision 1) — supplying it keeps the `ci.yaml` output
of framework initializations complete.

## Configuration extension

The standard declares its framework-specific configuration keys through
the `extension` command and validates values through the `validate`
command. The keys live under the `framework.flutter.` namespace
(ADR-005 §4.4; 007 §7 — namespace isolation is enforced by the runtime;
the standard owns the validation rules for its own values):

| Key | Default | Rule |
|---|---|---|
| `framework.flutter.targets` | `web,apk` | Non-empty comma-separated list of known targets (`web`, `apk`, `ios`), no duplicates — each known target is executed once, in table order |
| `framework.flutter.build_args` | *(none)* | Optional whitespace-separated `flutter build` arguments; shell metacharacters rejected |

The runtime enforces namespace isolation; the standard owns the value
validation rules (`internal/flutter/config.go`).

## Maintenance — template freshness

Template freshness — tracking Flutter framework version updates — is a
standard maintenance responsibility (007 §7; Transition Plan §4.7):
when the framework's build surface changes, the template, the build
target table, and the config extension rules are updated together in one
standard release, and the framework-version support scope
(`standard/manifest.json` → `capability.frameworkVersion`) is extended.

The freshness log below records each template review against the
framework versions the standard declares support for. A template change
without a log entry is a release-blocking gap:

| Date | Flutter versions reviewed | Template revision | Change |
|---|---|---|---|
| 2026-08-08 | 3.24.x, 3.27.x, 3.29.x, 3.32.x (current support scope) | v2 (TS-018-02-02) | Added `dependencies` stage (`flutter pub get`) ahead of the build stage; build stage unchanged (web → apk → ios with ADR-018 metadata) |
| 2026-08-06 | 3.0.0–3.32.x (initial support scope) | v1 (TS-016-02-01) | Initial adapter-owned build + CI templates (ADR-020 extraction) |
