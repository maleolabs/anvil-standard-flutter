# Adopting the Flutter Delivery Lifecycle

This guide explains how a Flutter project **adopts the Flutter delivery
lifecycle standard** (`anvil-standard-flutter`) and what the adopted
lifecycle does: what activation runs, what rollback reverses, what
verification checks, what the configuration surface is, what is
irreversible, and how compatibility is validated at adoption. It is the
adopter-facing entry point of the standard's Documentation part (007 §9).

The standard's executable content enforces what this guide describes
(manifesto §6 — documentation is a claim, enforcement is a fact): the
phase table, checks, and validation rules live in the executable and are
declared in the [Manifest](../standard/README.md); this guide only
points at them.

> **Terminology.** "Standard" here means the delivery lifecycle standard
> `anvil-standard-flutter` — the distributable unit of Flutter lifecycle
> knowledge. "Adapter" is the v1.x term for the standard's executable
> (`anvil-adapter-flutter`); the term mapping is part of the standard
> command contract (ADR-032).

## 1. What the standard declares

The standard declares its identity and capability surface in the
[Manifest](../standard/README.md) and its compatibility in the
[Compatibility part](../compatibility/README.md):

| Declaration | Value |
|---|---|
| **Standard id / name** | `anvil-standard-flutter` — Flutter delivery lifecycle standard |
| **Target contract version** | `1.0.0` (delivery lifecycle specification, ADR-024 §3.1) |
| **Framework-version support scope** | Flutter stable `3.0.0`–`3.32.0` |
| **Deployment model** | `hybrid` — releases are built and packaged for distribution (web bundle, APK, iOS app), not deployed to a server and activated in place |
| **Activation phases** | `pub_get`, `platform_sync` (see [Activation](#5-what-activation-runs)) |
| **Verification checks** | 2 structural checks (see [Verification](#6-what-verification-checks)) |
| **Config extensions** | 2 keys under `framework.flutter.*` (see [Configuration surface](#7-configuration-surface)) |

Compatibility is validated **at adoption** — the registry rejects a
standard that violates the specification's contracts, and the declared
contract version and framework-version support scope are checked before
the standard becomes installable — and **re-verified at runtime** when
the runtime executes the standard (007 §8, [compatibility/README.md](../compatibility/README.md)).

## 2. Prerequisites

- **Anvil Runtime** whose supported contract majors include the
  standard's target contract version (`1.0.0`, major `1` — the contract
  major is the compatibility unit, ADR-024 §3.1).
- **The standard installed through the registry flow** (below) — a
  framework declaration without the installed standard hard-fails
  initialization (ADR-026 decision 3).
- **Flutter SDK** on the build host — the standard executes
  `flutter pub get` and `flutter build <target>`; the release's locked
  dependency set resolves against the installed Flutter version.
- **CocoaPods** on macOS build hosts for the iOS target (`pod install`
  during activation's `platform_sync` when the release contains an
  `ios/` directory).
- **The standard executable `anvil-adapter-flutter`** on `PATH` —
  required for build/verify/activate invocations. Interim install path
  (standard releases not published yet): build from source, see
  [README.md](README.md#building).

## 3. Install the standard (registry flow)

The standard is installed through the registry flow — explicit adoption:
validation (contract version, capability, framework-version support
scope) + integrity (content digests) + attestation (publisher signature)
+ record. There is no skip or insecure path (ADR-022 §3):

```bash
# Discovery: list the offered releases of the standard
anvil standard list --index <checkout-of-anvil-standard-flutter>

# Inspect one release (its declared compatibility, capability, trust)
anvil standard inspect anvil-standard-flutter 1.0.0 --index <checkout-of-anvil-standard-flutter>

# Install — explicit adoption; the operator's trust anchors allowlist
# must pin the publisher's key (--trust-anchors or the default
# ~/.config/anvil/trust-anchors.json)
anvil standard install anvil-standard-flutter 1.0.0 --index <checkout-of-anvil-standard-flutter>
```

Installation records the installed standard (id, pinned version); the
record is what subsequent `anvil init --framework flutter` resolves
(TS-015-02-01). See [docs/release.md](release.md) for the registry flow
and the trust model.

## 4. Adopt the lifecycle in a project

```bash
# 1. Initialize a Flutter project — records the framework declaration and
#    generates the Flutter build pipeline (template content)
anvil init my-app --framework flutter
cd my-app

# 2. Build (pub get → web/apk/ios targets, per the build pipeline template)
anvil pipeline build

# 3. Package an immutable artifact
anvil artifact package

# 4. Verify the artifact (runs the 2 structural checks)
anvil artifact verify .anvil/artifacts/<artifact>.tar.gz

# 5. Distribute the packaged artifact (web bundle, APK, iOS app) —
#    deployment in the hybrid model means distributing the packaged
#    artifact to its targets, not installing it on a server
```

The hybrid deployment model (ADR-016) has no server-side install or
in-place activation: there is no server to initialize and no release to
register on one. Activation runs the release's dependency set and
platform steps on the release working directory when the release is
served (see [Activation](#5-what-activation-runs)).

Initialization writes the framework declaration (`project.framework:
flutter`) and, when the standard's record carries config extension
content, the `framework.flutter.*` keys with their declared defaults
(TS-015-03-01).

## 5. What activation runs

Release activation runs the declared activation phases in order, each as
`<program> <args>` from the release's working directory — **dependency
resolution → platform steps**:

| # | Phase | Command | Reversible? |
|---|---|---|---|
| 1 | `pub_get` | `flutter pub get` | Reversible (re-runs `flutter pub get` — idempotent lockfile-driven re-resolution) |
| 2 | `platform_sync` | `pod install` (iOS platform steps; conditional) | Irreversible |

Key semantics (full detail in the [Lifecycle Definition](../lifecycle/README.md)):

- **Dependency resolution before promotion.** `pub_get` runs **before**
  the artifact is promoted — the release serves exactly what its
  `pubspec.lock` declares; a failing resolution fails activation (an
  unresolvable dependency set cannot serve).
- **Platform steps after resolution.** `platform_sync` runs after
  `pub_get`, on the resolved dependency set. It applies only when the
  release contains an `ios/` directory and the host is macOS (CocoaPods
  is a macOS tool); otherwise it is an informational no-op — platform
  state that does not exist is not a failure.
- **Per-phase failure.** A failing phase fails the activation: `pub_get`
  failure means the dependency set cannot serve; `platform_sync` failure
  means the artifact cannot serve its native target.

## 6. What verification checks

`anvil artifact verify` runs the standard's **2 structural verification
checks** against the packaged artifact — all must pass or the
verification fails (see [verification/README.md](../verification/README.md)):

`pubspec_yaml` · `lib_directory`

The checks validate that the artifact looks like a distributable Flutter
application (the Flutter project manifest and the application source
directory) before it is distributed. Gates are mandatory and
unskippable — the standard adds checks, it never weakens gates (007 §6).

## 7. Configuration surface

The standard declares **2 configuration keys** under the
`framework.flutter.` namespace (ADR-005 §4.4) — declared by the
`extension` command, validated by the `validate` command, and enforced
by the runtime on `framework.<name>.*` keys from the installed
standard's config extension rules (TS-015-03-02; [templates/README.md](../templates/README.md)):

| Key | Default | Validation |
|---|---|---|
| `framework.flutter.targets` | `web,apk` | Non-empty comma-separated list of known targets (`web`, `apk`, `ios`); duplicates rejected — each known target is executed once |
| `framework.flutter.build_args` | — (optional) | Whitespace-separated `flutter build` arguments; shell metacharacters rejected |

Unknown keys under the namespace are rejected. The adopting project's
Flutter version against the standard's framework-version support scope
is a compatibility validation fact, not an assumption (see
[Compatibility](../compatibility/README.md)).

## 8. What rollback reverses — and what is irreversible

Rollback restores the previously active release, then runs the rollback
operations. **Irreversibility never blocks rollback** — the adapter
reports an informational result for irreversible phases and the
rollback proceeds:

| Phase | Rollback behavior |
|---|---|
| `pub_get` | Reversed by re-running `flutter pub get` in the restored release's working directory — resolution is idempotent and lockfile-driven, so re-running it re-establishes the previous release's locked dependency state |
| `platform_sync` | **Irreversible** — a rollback cannot undo the platform integration; the restored release's own activation re-runs its platform steps |

`platform_sync` is the irreversible surface of the Flutter lifecycle:
its effects reflect exactly one release's state and are never repaired,
only regenerated by the next activation.

## 9. Where the details live

| Topic | Document |
|---|---|
| Standard identity and capability declaration | [Manifest](../standard/README.md) |
| Contract version and framework-version support scope | [Compatibility](../compatibility/README.md) |
| Activation, rollback, and failure semantics per phase | [Lifecycle Definition](../lifecycle/README.md) |
| Verification rules | [Verification](../verification/README.md) |
| Templates and config extension | [Templates](../templates/README.md) |
| Release pipeline and trust model | [docs/release.md](release.md) |

See also: [Flutter lifecycle documentation](flutter-lifecycle.md) · [Glossary](https://github.com/maleolabs/forge-anvil-cli/blob/develop/wiki/glossary.md)
