# Compatibility — Declared, Validated, Recorded

The Compatibility part makes the standard's validity explicit: declared,
validated, and recorded — not assumed ([ADR-021 §3.2](../../docs/architecture/006-b-seven-part-standard-structure.md);
`006-b §3.5`).

## Declared compatibility

| Declaration | Value | Meaning |
|---|---|---|
| **Target contract version** | `1.0.0` | The delivery lifecycle specification version this release is valid against. The contract major (`1`) is the compatibility unit (ADR-024 §3.1): the runtime supports the declared major, and a standard release never breaks the contract it declares. Mirrors the specification corpus version line (`docs/specification-corpus/version-line.md`, supported contract majors `{1}`) |
| **Framework versions supported** | Flutter stable releases 3.0.0–3.32.0 | The framework-version support scope of this release, declared in the manifest (`standard/manifest.json`, `capability.frameworkVersion`). The standard's content (build targets, verification, config surface) applies to the declared Flutter stable line |

## Validation

- **At adoption** — the registry validates the declared contract version
  and framework-version scope before the standard becomes installable
  (ADR-023); a release that violates the specification's contracts is
  rejected, not patched.
- **At runtime** — the runtime re-verifies compatibility when it
  executes the standard (ADR-024 §3.6).

## Maintenance

Compatibility is per-release: every release re-declares its target
contract version and framework-version support scope. Framework updates
ship as new standard versions through the registry — never through an
Anvil Runtime release (ADR-021 §3.4). When the contract major advances,
this standard's contractVersion declaration is updated in the same
release that implements the new contract (within the deprecation window
defined by ADR-024 §3.4).
