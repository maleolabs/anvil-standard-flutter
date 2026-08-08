# Anvil Standard — Flutter

The Flutter delivery lifecycle standard for [Anvil](https://github.com/maleolabs/forge-anvil-cli):
the distributable unit of Flutter framework lifecycle knowledge for the
Anvil Runtime. It implements the delivery lifecycle specification's
standard command contract for the Flutter framework (hybrid deployment
model) and is resolved by the Anvil Runtime as the standalone executable
`anvil-adapter-flutter`.

Per the Anvil repository model ([ADR-025](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/adr/ADR-025-repository-split-core-vs-standards.md)),
this repository is one of the first-party delivery lifecycle standards:

| Repository | Standard |
|---|---|
| `maleolabs/anvil` | Anvil Core (runtime + specification + governance) |
| `maleolabs/anvil-standard-laravel` | Laravel delivery lifecycle standard |
| `maleolabs/anvil-standard-flutter` | **Flutter delivery lifecycle standard (this repository)** |

## What this repository contains

A delivery lifecycle standard is conceptually composed of seven parts
([ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/adr/ADR-021-delivery-lifecycle-standard-model.md),
[Transition Plan §5.4](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/planning/ANVIL_V2_TRANSITION_PLAN.md)):

| Part | Location |
|---|---|
| Manifest | [`standard/manifest.json`](standard/manifest.json) |
| Lifecycle Definition | [`lifecycle/`](lifecycle/) |
| Verification | [`verification/`](verification/) |
| Templates | [`templates/`](templates/) |
| Compatibility | [`compatibility/`](compatibility/) |
| Documentation | [`docs/`](docs/) |
| Tests | [`tests/`](tests/) + the Go test suite under [`internal/`](internal/) |

The standard executable (`cmd/flutter-adapter`) implements the standard
command contract — `capabilities`, `build`, `verify`, `extension`,
`validate`, `template`, `manifest` — with the JSON wire shapes defined by
the delivery lifecycle specification's command contract. The binary name
convention is `anvil-adapter-<framework>`; the Core resolves this standard
as `anvil-adapter-flutter`.

## Building

```sh
go build -o anvil-adapter-flutter ./cmd/flutter-adapter
```

The module is self-contained (stdlib only): it carries its own Go mirror
of the specification's wire-contract payloads and builds from this
repository alone — no dependency on the Core repository.

## Testing

```sh
go test -race -count=1 ./...
```

## CI

Every push and pull request against `develop` and `main` runs the
repository's CI pipeline ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)):

1. builds the standard executable (`go build -o anvil-adapter-flutter ./cmd/flutter-adapter`);
2. runs the standard's test suite (`go test -race -count=1 ./...`), gated by
   `go build ./...`, `go vet ./...`, and a `gofmt` check;
3. validates the source manifest ([`standard/manifest.json`](standard/manifest.json))
   for internal consistency via [`scripts/validate-manifest.sh`](scripts/validate-manifest.sh):
   the manifest must parse as JSON and the declared contract version
   (`contractVersion`) must be well-formed semver. Registry parseability is
   deliberately out of scope for the source manifest — the release-time
   fields (`distribution`, `lifecycle`, `trust`) are populated at publication
   by the release pipeline (TS-016-03-02), which is why the source manifest
   is not a strict-parser registry document by design (see the parseability
   note in [standard/README.md](standard/README.md)).

**Release gate.** A failing pipeline blocks release production: `develop` and
`main` are protected branches that require this CI to pass before merge, and
release candidates are produced only from a green integration branch (the
release publication pipeline itself is TS-016-03-02 scope).

## Releases

Versioned releases are produced and published from this repository alone —
a standard release never requires a Core release (ADR-025 §3.5, §4.7). A
tag `v<version>` on `main` triggers the release pipeline
([`.github/workflows/release.yml`](.github/workflows/release.yml)): it runs
the CI green gate, builds the standard executable for the release platforms,
packages the release archive, derives and signs the registry metadata
document (real content digests + Ed25519 publisher attestation, ADR-022),
and publishes the GitHub Release with the registry metadata document. Each
release declares the contract version it targets and its framework-version
support scope, and is discoverable/installable through the registry flow
(`anvil standard list|inspect|install`).

The version line lives in the manifest `version` field
([`standard/manifest.json`](standard/manifest.json)). Releases are cut from
`main`: merge `develop` into `main`, then tag and push `v<version>`. Tags
with a `-test`/`-pre` suffix create GitHub pre-releases (e.g.
`v1.1.0-test`). Full mechanics, trust model, and the local pipeline
(`scripts/release.sh`) are documented in [docs/release.md](docs/release.md).

## Documentation

- [Adoption guide](docs/adopt.md) — how a Flutter project adopts the lifecycle and what it does (007 §9)
- [Flutter lifecycle documentation](docs/flutter-lifecycle.md) — the Flutter lifecycle for adopters
- [Lifecycle Definition](lifecycle/) — activation and rollback semantics
- [Verification](verification/) — the Flutter structural checks
- [Templates](templates/) — build pipeline and configuration extension content
- [Compatibility](compatibility/) — declared contract version and framework-version support scope

## License

Apache-2.0 — see [LICENSE](LICENSE).
