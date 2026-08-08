# Tests — The Standard's Own Tests

The Tests part is the standard's own tests ([Transition Plan §5.4](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/planning/ANVIL_V2_TRANSITION_PLAN.md);
[ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/adr/ADR-021-delivery-lifecycle-standard-model.md)):
the checks that validate that the standard behaves as it declares,
validated at registry acceptance.

## What is tested

| Area | Location |
|---|---|
| Standard executable behavior (command contract) | `internal/flutter/*_test.go` — the moved adapter test suite: capability declaration, build pipeline (target selection, platform filtering, strict mode), verification checks, config extension + validation, template command, exit-code semantics |
| Activation phases + rollback (lifecycle content, TS-018-02-01) | `internal/flutter/activation_test.go` — declared phase order (pub_get before platform_sync), per-phase activation/failure/rollback semantics, irreversible-phase informational handling, platform-aware execution (ios/ directory, darwin host), manifest command strings |
| Lifecycle-conformity verification rules (TS-018-03-02) | `internal/flutter/verification_test.go` — the four lifecycle checks (dependency_lockfile, dependency_timing, platform_sync_ready, rollback_behavior) over directory and archive artifacts, plus the two structural checks (TS-P7-25) |
| Template + config extension validity (TS-018-02-02) | `internal/flutter/template_test.go` + `config_test.go` — pipeline loader validation of the build/CI definitions, build-stage mirror of the target table, config keys + validation rules |
| Executable entrypoint | `internal/flutter/binary_test.go` — builds `cmd/flutter-adapter` and exercises the real binary |
| Source manifest content (007 §4, §8) | `internal/release/source_manifest_test.go` — pins the source manifest's declared identity, target contract version, and framework-version support scope, and verifies the release pipeline's derived document preserves them (TS-018-02-03) |
| Seven-part structure + manifest | `tests/standard_structure_test.go` — the seven parts exist with content and the manifest is a consistent registry-metadata-format declaration |
| Acceptance readiness (ADR-027) | `tests/acceptance_readiness_test.go` — the standard's content parts validated against the executable declaration: lifecycle content (phases, order, rollback), verification rules, Manifest-part mirror of the capability declaration, conformance against the declared contract version, maintainer declaration |

## Running

```sh
go test -race -count=1 ./...
```

## Registry acceptance

Registry acceptance validates the standard on structure, conformance,
tests, and maintainership (ADR-027); carrying the Tests part at a
quality the registry validates is the standard's first responsibility
([007 §2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/architecture/007-delivery-lifecycle-standard-specification.md)).
The tests concern the standard itself, not an adopting project's
release — project-facing checks are the Verification part.

### Acceptance evidence per ADR-027

| Bar (ADR-027 §3) | Evidence in this repository |
|---|---|
| **Structure** — all seven parts present | `tests/standard_structure_test.go` (`TestSevenPartStructureExists`) — every part exists with non-trivial content (Manifest, Lifecycle Definition, Verification, Templates, Compatibility, Documentation, Tests) |
| **Conformance** — validated against the declared contract version | `tests/acceptance_readiness_test.go` (`TestConformance_ContractVersionDeclaredConsistently`) + `internal/release/source_manifest_test.go` (pins `contractVersion` `1.0.0`) + `tests/standard_structure_test.go` (`TestManifestContractVersionMatchesCompatibility`) — manifest, Compatibility part, and Manifest part agree on contract version `1.0.0`; registry validation mechanics belong to the registry (EPIC-014), this suite keeps the declaration internally consistent before acceptance |
| **Tests** — the standard's own tests pass, registry-validated | This suite, run by CI on every push/PR (`go test -race -count=1 ./...`, `.github/workflows/ci.yml`); the Tests part is the suite the registry validates at acceptance |
| **Maintainership** — maintainer declared and accountable | `standard/README.md` §Maintainer (Maleo Labs, declared and accountable) — pinned by `tests/acceptance_readiness_test.go` (`TestMaintainer_DeclaredAndAccountable`) |

The Tests part keeps the standard ready for the Draft → Published
transition: content parts cannot drift from the executable declaration
(the runtime invokes only declared capability, TS-P7-08 AC-3), and the
four acceptance bars have a passing test each.
