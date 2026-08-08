# Flutter Lifecycle Documentation

Adopter documentation for the Flutter delivery lifecycle standard
([ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/architecture/006-b-seven-part-standard-structure.md);
`006-b §3.6`): what the standard's lifecycle contains, how a project
adopts it, and what behavior to expect.

## What this standard is

The Flutter delivery lifecycle standard carries the Flutter framework
lifecycle knowledge for the Anvil Runtime: the build targets, the
structural verification checks, the configuration surface, and the
pipeline templates. The Anvil Runtime resolves it as the standalone
executable `anvil-adapter-flutter` (binary name convention
`anvil-adapter-<framework>`), exchanged over the standard command
contract.

## Deployment model

Flutter releases use the **hybrid deployment model** (ADR-016): releases
are **built and packaged for distribution** (web, APK, iOS) — they are
not deployed to a server and activated in place. Activation reflects
this model: the release's dependency set and platform steps run at
activation time on the release working directory; deployment means
distributing the packaged artifact (web bundle, APK, iOS app).

## Activation

The runtime's activation phase sequence (in the CLI, `anvil deployment
activate`, TD-006) runs the standard's declared activation phases in
declared order from the release working directory:

| # | Phase | Command | Reversible |
|---|---|---|---|
| 1 | `pub_get` | `flutter pub get` | ✅ rollback: `flutter pub get` (idempotent re-resolution from the lockfile) |
| 2 | `platform_sync` | `pod install` (iOS platform steps; conditional) | ❌ irreversible |

- **`pub_get`** resolves the release's locked dependency set **before
  promotion** — the release serves exactly what its `pubspec.lock`
  declares. A failing resolution fails activation (an unresolvable
  dependency set cannot serve).
- **`platform_sync`** runs the native platform steps after dependency
  resolution. It applies when the release contains an `ios/` directory
  and the host is macOS (CocoaPods is a macOS tool); otherwise it is an
  informational no-op — platform-aware execution mirroring the build
  side (ADR-018). A failing platform step fails activation.
- **Rollback:** `pub_get` rollback re-runs `flutter pub get` in the
  restored release's working directory. `platform_sync` is
  **irreversible** — rollback reports an informational success and
  never blocks the rollback; the previous release's own activation
  re-runs its platform steps.

The manifest metadata surface carries the activation command strings
(`flutter pub get`, `pod install`) and the rollback command string
(`flutter pub get`).

## Build targets

`anvil pipeline build` (or the build template generated at init) runs
the Flutter build targets in this order:

| Target | Command | Platforms |
|---|---|---|
| `web` | `flutter build web` | linux, darwin, windows |
| `apk` | `flutter build apk --release` | linux, darwin, windows |
| `ios` | `flutter build ios --release` | darwin (requires Xcode) |

Platform-aware execution (ADR-018): an unsupported target is skipped
with a warning; `--target <name>` selects a single target; `--strict`
fails unsupported targets instead of skipping.

## Verification

`anvil artifact verify` runs the standard's structural checks against
the packaged artifact:

- `pubspec_yaml` — `pubspec.yaml` exists in the artifact root
- `lib_directory` — `lib/` exists in the artifact

## Configuration

The standard declares two configuration keys under the
`framework.flutter.` namespace, written to `anvil.yaml` at init and
validated by the standard's `validate` command:

- `framework.flutter.targets` — comma-separated build targets
  (default `web,apk`; known targets `web`, `apk`, `ios`)
- `framework.flutter.build_args` — optional extra `flutter build`
  arguments (whitespace-separated, no shell metacharacters)

## Templates

At init, the standard supplies `.anvil/pipelines/build.yaml` (the build
pipeline above) and `.anvil/pipelines/ci.yaml` (a generic CI scaffold).

## Adopting this standard

1. Install the standard: `anvil standard install anvil-standard-flutter 1.0.0`
   (or `anvil adapter install flutter` for the pre-registry v1.x flow).
2. Initialize a project with the framework:
   `anvil init my-app --framework flutter` (or `anvil adapter use flutter`
   in an existing project).
3. Build: `anvil pipeline build` — package: `anvil artifact package` —
   verify: `anvil artifact verify <artifact>`.

The executable resolution contract is unchanged by the repository split
(ADR-025 §3.4, §12.1–12.2): `anvil-adapter-flutter` on PATH, invoked as
`<executable> <command> <json-payload>`.

## Vocabulary

Vocabulary is owned by the delivery lifecycle specification; this
standard documents its lifecycle behavior and does not redefine
semantics ([006-v2-architecture-overview §5](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/architecture/006-v2-architecture-overview.md)).
