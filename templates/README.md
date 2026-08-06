# Templates — Generated Content

The Templates part carries the generated content a project receives from
the standard at init time ([ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/architecture/006-b-seven-part-standard-structure.md);
`006-b §3.4`).

## Build pipeline template

[`build.yaml`](build.yaml) is the committed copy of the build pipeline
definition the standard returns through the `template` command: the
Flutter build targets in execution order with their ADR-018 platform
metadata. The runtime validates it through the pipeline loader and
writes it to `.anvil/pipelines/build.yaml` at generation time.

## CI pipeline scaffold

[`ci.yaml`](ci.yaml) is the committed copy of the CI scaffold (build +
test placeholder stages) the standard returns alongside the build
definition.

## Configuration extension

The standard declares its framework-specific configuration keys through
the `extension` command and validates values through the `validate`
command. The keys live under the `framework.flutter.` namespace
(ADR-005 §4.4):

| Key | Default | Rule |
|---|---|---|
| `framework.flutter.targets` | `web,apk` | Non-empty comma-separated list of known targets (`web`, `apk`, `ios`) |
| `framework.flutter.build_args` | *(none)* | Optional whitespace-separated `flutter build` arguments; shell metacharacters rejected |

The runtime enforces namespace isolation; the standard owns the value
validation rules (`internal/flutter/config.go`).

## Maintenance

Template freshness — tracking Flutter framework version updates — is a
standard maintenance responsibility: when the framework's build surface
changes, the template, the build target table, and the config extension
rules are updated together in one standard release.
