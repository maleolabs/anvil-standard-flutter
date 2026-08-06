// Package contracts is this standard's Go mirror of the delivery
// lifecycle specification's standard command contract payloads (the
// wire contract exchanged with the Anvil Runtime over subprocess JSON).
//
// The standard is an independent module and cannot import the Anvil
// Core's internal packages (Go internal-package rule), so it carries its
// own Go types for the wire payloads. The JSON wire shapes are the
// contract: they are defined by the delivery lifecycle specification
// (docs/specification-corpus/command-contract.md and
// command-contract.schema.json in the Core repository; ADR-029 §3 — the
// schema governs). These types mirror the Core's internal/contracts
// mirror byte-for-byte (same struct shapes, same json tags), so the
// standard's stdout parses into the Core's payload types unchanged.
//
// When the specification's command contract advances, this mirror MUST
// be updated in the same release — a standard release never breaks the
// contract it declares (ADR-021 §3.4, ADR-024).
package contracts
