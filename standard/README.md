# Manifest — Standard Identity

The Manifest is the standard's identity part ([ADR-021 §3.2](https://github.com/maleolabs/forge-anvil-cli/blob/develop/docs/planning/ANVIL_V2_TRANSITION_PLAN.md)
in the Core repository; `006-b-seven-part-standard-structure.md §3.1`):
who the standard is, what contract version it targets, and what it
provides.

## `standard/manifest.json`

The manifest follows the registry metadata format conventions the Anvil
Runtime registry client reads (`registry-metadata.schema.json`,
`internal/registry/metadata.go` in the Core repository): the same field
names, semver patterns, and capability shape.

| Field | Value | Meaning |
|---|---|---|
| `id` | `anvil-standard-flutter` | Standard identity, stable across releases (first-party naming per ADR-025 §3.1) |
| `version` | `1.0.0` | This release's version on the standard's independent semver line (ADR-021 §3.4) |
| `contractVersion` | `1.0.0` | The delivery lifecycle specification contract version this release targets; the major version is the compatibility unit (ADR-024 §3.1) |
| `capability.frameworkVersion` | Flutter stable releases | The framework-version support scope: the Flutter versions this release supports (ADR-021 §3.2) |

## Release-time fields

The registry metadata format additionally carries distribution,
lifecycle, and trust sections (ADR-030, ADR-022). Those fields describe
a **published release** — the release asset location, the governed
lifecycle state, and the content digests/attestation — and are populated
by the standard's release pipeline when a versioned artifact is
published to the registry (EPIC-016 scope, TS-016-03-02).

> **Parseability note.** Because this source manifest intentionally
> carries authoring-time fields only, it is **NOT a valid registry
> metadata document** for the Core registry client's strict parser
> (`registry-metadata.schema.json` requires distribution, lifecycle, and
> trust). Do not attempt to parse `standard/manifest.json` with the
> registry parser — it becomes parseable only after TS-016-03-02 fills
> the release-time fields at publication time. This README is the single
> source of truth for the source manifest's semantics; the published
> registry metadata document is generated from this manifest plus the
> release-time fields.

## Manifest maintenance

A manifest change is a release event: bump `version` for any content
change, and re-declare `contractVersion`/`capability` whenever the
standard's compatibility surface changes (a standard release never
breaks the contract it declares — ADR-021 §3.4).
