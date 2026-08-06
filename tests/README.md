# Tests — The Standard's Own Tests

The Tests part is the standard's own tests ([Transition Plan §5.4](../../docs/planning/ANVIL_V2_TRANSITION_PLAN.md);
[ADR-021 §3.2](../../docs/adr/ADR-021-delivery-lifecycle-standard-model.md)):
the checks that validate that the standard behaves as it declares,
validated at registry acceptance.

## What is tested

| Area | Location |
|---|---|
| Standard executable behavior (command contract) | `internal/flutter/*_test.go` — the moved adapter test suite: capability declaration, build pipeline (target selection, platform filtering, strict mode), verification checks, config extension + validation, template command, exit-code semantics |
| Executable entrypoint | `internal/flutter/binary_test.go` — builds `cmd/flutter-adapter` and exercises the real binary |
| Seven-part structure + manifest | `tests/standard_structure_test.go` — the seven parts exist and the manifest is a consistent registry-metadata-format declaration |

## Running

```sh
go test -race -count=1 ./...
```

## Registry acceptance

Registry acceptance validates the standard on structure, conformance,
tests, and maintainership (ADR-027); carrying the Tests part at a
quality the registry validates is the standard's first responsibility
([007 §2](../../docs/architecture/007-delivery-lifecycle-standard-specification.md)).
The tests concern the standard itself, not an adopting project's
release — project-facing checks are the Verification part.
