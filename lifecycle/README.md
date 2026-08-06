# Lifecycle Definition — Flutter

The Lifecycle Definition is the framework's lifecycle content: what a
legal lifecycle *contains* for one framework ([ADR-021 §3.2](../docs/architecture/006-b-seven-part-standard-structure.md);
`006-b §3.2`).

## Deployment model: hybrid

The Flutter standard implements the **hybrid deployment model**
(ADR-016): releases are built and packaged for distribution (web, APK,
iOS) rather than deployed to a server and activated in place (EPIC-007
§7.3). The hybrid model has **no server activation phases**.

## Activation semantics

- **Declared activation phases: none.** The capability declaration
  intentionally declares an empty activation phase list (TS-P7-20 AC-5).
  The runtime proceeds with its generic lifecycle for the positions the
  standard does not cover; the standard covers the build and package
  positions instead.
- **`activate` command: intentionally absent.** Invoking `activate`
  against this standard reports an unknown command (exit 2) — there is
  no server activation to run. This is declared behavior, not a defect.

## Rollback semantics

- **Declared rollback phases: none.** With no activation phases there is
  nothing to reverse: the hybrid model's releases are distribution
  artifacts, and rollback at the server level does not apply.
- **`manifest` command: empty result.** The artifact manifest stores
  activation/rollback command strings as deployment metadata (ADR-017);
  this standard returns empty slices, which the packaging layer omits
  from the manifest (`omitempty`, backward compatible with old
  artifacts).

## What the lifecycle contains for Flutter

| Lifecycle position | Content |
|---|---|
| Build | `flutter build web`, `flutter build apk --release`, `flutter build ios --release` in declared order (web → apk → ios), with platform metadata (ios is darwin-only) |
| Verification | Structural checks of the built artifact: `pubspec.yaml` present, `lib/` directory present |
| Configuration | `framework.flutter.*` keys (targets, build_args) with value validation |
| Templates | The build and CI pipeline definitions generated at project init |

The stage model, phase sequence, and transition rules are owned by the
delivery lifecycle specification; this standard supplies content within
the defined lifecycle — it does not invent a lifecycle.
