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

## Documentation

- [Adopter documentation](docs/flutter-lifecycle.md) — the Flutter lifecycle for adopters
- [Lifecycle Definition](lifecycle/) — activation and rollback semantics
- [Verification](verification/) — the Flutter structural checks
- [Templates](templates/) — build pipeline and configuration extension content
- [Compatibility](compatibility/) — declared contract version and framework-version support scope

## License

Apache-2.0 — see [LICENSE](LICENSE).
