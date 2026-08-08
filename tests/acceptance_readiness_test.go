// Package-level acceptance-readiness tests of the Flutter delivery
// lifecycle standard (TS-018-04-02).
//
// These tests complete the Tests part of the seven-part standard
// structure (ADR-021 §3.2, Transition Plan §5.4) at the ADR-027
// acceptance bar: the standard's own tests must validate that the
// standard behaves as it declares (tests/README.md), the declaration
// must be consistent against the declared contract version, and a
// maintainer must be declared and accountable (ADR-027 §3 — structure,
// conformance, tests, maintainership).
//
// Where the unit tests under internal/flutter verify the executable
// behavior of the standard, these tests verify the CONTENT parts — the
// Lifecycle Definition, Verification, Manifest, and Compatibility
// documents — against that same executable declaration, so the
// human-readable content cannot drift from the code the runtime
// executes (007 §4: the runtime invokes only declared capability).
//
// The registry validates these bars mechanically at acceptance
// (ADR-023; EPIC-014 mechanics); this suite is the standard's own
// evidence that the content under test passes the bar before
// submission.
package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"maleolabs.com/anvil-standard-flutter/internal/flutter"
)

// readPart returns the content of a repository part document, failing
// the test when it cannot be read.
func readPart(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// containsAll reports every required substring the content is missing.
func containsAll(content string, required ...string) []string {
	var missing []string
	for _, want := range required {
		if !strings.Contains(content, want) {
			missing = append(missing, want)
		}
	}
	return missing
}

// activationSection returns the slice of a lifecycle document starting
// at its activation heading ("## Activation"). Order assertions run
// against this slice so a phase name mentioned in a table of contents
// or an unrelated section cannot influence them.
func activationSection(t *testing.T, name, doc string) string {
	t.Helper()
	loc := regexp.MustCompile(`(?m)^## Activation`).FindStringIndex(doc)
	if loc == nil {
		t.Fatalf("%s has no \"## Activation\" heading — cannot scope the activation-order assertions", name)
	}
	return doc[loc[0]:]
}

// TestLifecycleContent_DeclaresActivationPhasesAndRollback verifies the
// Lifecycle Definition part (lifecycle/README.md) and the Documentation
// part (docs/flutter-lifecycle.md) declare the activation phase content
// authored in TS-018-02-01: the hybrid model's phases in declared order
// (pub_get then platform_sync), the rollback semantics (pub_get —
// idempotent re-resolution; platform_sync — irreversible, never blocks
// rollback), and the command strings derived from the executable phase
// table (ManifestCommands). The documents are the adopters' reference;
// the Go phase table is the single source of truth — this test keeps
// the two from drifting.
func TestLifecycleContent_DeclaresActivationPhasesAndRollback(t *testing.T) {
	root := repoRoot(t)
	lifecycle := readPart(t, root, filepath.Join("lifecycle", "README.md"))
	docs := readPart(t, root, filepath.Join("docs", "flutter-lifecycle.md"))

	declared := flutter.Capabilities().Declaration.ActivationPhases
	if len(declared) != 2 || declared[0] != flutter.PhasePubGet || declared[1] != flutter.PhasePlatformSync {
		t.Fatalf("executable activation phase declaration = %v, want [pub_get platform_sync]", declared)
	}

	// The manifest command strings derived from the phase table (ADR-017):
	// activation "flutter pub get" then "pod install", rollback "flutter
	// pub get" only. Declared here so the order and command assertions
	// below compare against the executable surface.
	commands := flutter.ManifestCommands()

	for _, doc := range map[string]string{
		"lifecycle/README.md":       lifecycle,
		"docs/flutter-lifecycle.md": docs,
	} {
		if missing := containsAll(doc, declared[0], declared[1]); len(missing) > 0 {
			t.Errorf("documentation does not declare activation phase(s) %v", missing)
		}
	}

	// Declared order: pub_get (dependency resolution) before
	// platform_sync (platform steps) in both documents. The search is
	// scoped to the activation section of each document ("## Activation"
	// heading) — a bare whole-document index would false-fail on a phase
	// name mentioned in a table of contents or a note, and false-pass if
	// a phase appeared only in an unrelated section — and compares the
	// command-string positions ("flutter pub get" before "pod install",
	// the manifest command surface derived from the phase table) rather
	// than the short phase names.
	for name, doc := range map[string]string{"lifecycle/README.md": lifecycle, "docs/flutter-lifecycle.md": docs} {
		activation := activationSection(t, name, doc)
		first, second := strings.Index(activation, declared[0]), strings.Index(activation, declared[1])
		if first == -1 || second == -1 || first > second {
			t.Errorf("%s activation section does not list the phases in declared order (%s before %s)", name, declared[0], declared[1])
		}
		pubGetCmd, podCmd := commands.ActivationCommands[0], commands.ActivationCommands[1]
		pubGetPos, podPos := strings.Index(activation, pubGetCmd), strings.Index(activation, podCmd)
		if pubGetPos == -1 || podPos == -1 || pubGetPos > podPos {
			t.Errorf("%s activation section does not declare the command order (%q before %q)", name, pubGetCmd, podCmd)
		}
	}

	// Rollback semantics (TS-018-02-01, 007 §5): pub_get reversible via
	// idempotent re-resolution; platform_sync irreversible.
	if missing := containsAll(lifecycle, "idempotent re-resolution", "irreversible", "never blocks"); len(missing) > 0 {
		t.Errorf("lifecycle/README.md misses rollback semantics: %v", missing)
	}
	if missing := containsAll(docs, "irreversible", "never blocks"); len(missing) > 0 {
		t.Errorf("docs/flutter-lifecycle.md misses rollback semantics: %v", missing)
	}

	// The manifest command strings derived from the phase table must be
	// what the documents declare (ADR-017): activation "flutter pub
	// get" then "pod install", rollback "flutter pub get" only.
	for _, cmd := range commands.ActivationCommands {
		if !strings.Contains(lifecycle, cmd) {
			t.Errorf("lifecycle/README.md does not declare the activation command %q (derived from the phase table)", cmd)
		}
	}
	for _, cmd := range commands.RollbackCommands {
		if !strings.Contains(lifecycle, cmd) {
			t.Errorf("lifecycle/README.md does not declare the rollback command %q (derived from the phase table)", cmd)
		}
	}
}

// TestVerificationContent_DeclaresAllChecks verifies the Verification
// part (verification/README.md) and the Documentation part
// (docs/flutter-lifecycle.md) declare the verification rules authored
// in TS-018-03-02: the two structural checks (TS-P7-25) and the four
// lifecycle-conformity checks, matching the executable capability
// declaration — the runtime invokes only declared checks
// (TS-P7-08 AC-3), so a document that drifts from the declaration
// misleads adopters about what the standard verifies.
func TestVerificationContent_DeclaresAllChecks(t *testing.T) {
	root := repoRoot(t)
	verification := readPart(t, root, filepath.Join("verification", "README.md"))
	docs := readPart(t, root, filepath.Join("docs", "flutter-lifecycle.md"))

	declared := flutter.Capabilities().Declaration.VerificationChecks
	if len(declared) != 6 {
		t.Fatalf("executable verification declaration = %d checks, want 6 (2 structural + 4 lifecycle-conformity)", len(declared))
	}
	var checkNames []string
	for _, check := range declared {
		checkNames = append(checkNames, check.Name)
	}
	if missing := containsAll(verification, checkNames...); len(missing) > 0 {
		t.Errorf("verification/README.md does not declare check(s) %v (must match the capability declaration)", missing)
	}

	// The adopters' documentation covers the lifecycle-conformity rules
	// (the TS-018-03-02 surface); the structural checks are covered in
	// verification/README.md.
	lifecycleChecks := []string{
		flutter.CheckDependencyLockfile,
		flutter.CheckDependencyTiming,
		flutter.CheckPlatformSyncReady,
		flutter.CheckRollbackBehavior,
	}
	if missing := containsAll(docs, lifecycleChecks...); len(missing) > 0 {
		t.Errorf("docs/flutter-lifecycle.md does not document lifecycle-conformity check(s) %v", missing)
	}
}

// TestManifestPart_MirrorsCapabilityDeclaration verifies the Manifest
// part (standard/README.md) mirrors the executable capability
// declaration — the standard's complete lifecycle surface per 007 §4
// (the runtime invokes only declared capability, TS-P7-08 AC-3): the
// hybrid deployment model, the activation phases, the build phases, the
// verification checks, the config extension keys, and the template set.
// The Manifest part is the identity document adopters and the registry
// read first; if it drifts from what the executable declares, the
// standard misrepresents itself.
func TestManifestPart_MirrorsCapabilityDeclaration(t *testing.T) {
	root := repoRoot(t)
	manifestDoc := readPart(t, root, filepath.Join("standard", "README.md"))

	declaration := flutter.Capabilities().Declaration

	if missing := containsAll(manifestDoc, declaration.DeploymentModel); len(missing) > 0 {
		t.Errorf("standard/README.md does not declare the deployment model %q", declaration.DeploymentModel)
	}
	if missing := containsAll(manifestDoc, declaration.ActivationPhases...); len(missing) > 0 {
		t.Errorf("standard/README.md does not declare activation phase(s) %v", missing)
	}

	// Build phases asserted by their full command strings, not the bare
	// target names: the three-char names ("web", "apk", "ios") appear in
	// unrelated contexts, so name-only matching would pass even if the
	// build-phase declaration were removed. The command strings are the
	// executable build surface (build.go — the single source of build
	// knowledge); the Manifest part must declare them.
	for _, cmd := range []string{"flutter build web", "flutter build apk --release", "flutter build ios --release"} {
		if !strings.Contains(manifestDoc, cmd) {
			t.Errorf("standard/README.md does not declare the build-phase command %q", cmd)
		}
	}
	if len(declaration.BuildPhases) != 3 {
		t.Errorf("executable build phase declaration = %v, want the three Flutter targets (web, apk, ios)", declaration.BuildPhases)
	}
	var checkNames []string
	for _, check := range declaration.VerificationChecks {
		checkNames = append(checkNames, check.Name)
	}
	if missing := containsAll(manifestDoc, checkNames...); len(missing) > 0 {
		t.Errorf("standard/README.md does not declare verification check(s) %v", missing)
	}

	// The declared config extension keys and the template set — the
	// `extension`/`validate` and `template` command surfaces.
	for _, key := range flutter.ConfigExtension().Extension.Keys {
		if !strings.Contains(manifestDoc, key.Name) {
			t.Errorf("standard/README.md does not declare the config extension key %q", key.Name)
		}
	}
	if missing := containsAll(manifestDoc, "build.yaml", "ci.yaml"); len(missing) > 0 {
		t.Errorf("standard/README.md does not declare template(s) %v", missing)
	}
}

// TestConformance_ContractVersionDeclaredConsistently verifies the
// conformance declaration against the target contract version: the
// manifest's contractVersion (1.0.0), the Compatibility part, and the
// Manifest part must agree, and the declared contract major — the
// compatibility unit (ADR-024 §3.1) — must match the registry metadata
// format version the manifest claims. This is the standard's own
// evidence for the ADR-027 conformance bar: at registry acceptance the
// standard is validated against the contract version it declares
// (007 §8, ADR-023); the suite keeps the declaration internally
// consistent so acceptance never surprises.
func TestConformance_ContractVersionDeclaredConsistently(t *testing.T) {
	root := repoRoot(t)

	data, err := os.ReadFile(filepath.Join(root, "standard", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}

	const wantContract = "1.0.0"
	if m.ContractVersion != wantContract {
		t.Errorf("manifest contractVersion = %q, want %q (the delivery lifecycle specification version this standard targets)", m.ContractVersion, wantContract)
	}
	if !semverPattern.MatchString(m.ContractVersion) {
		t.Errorf("manifest contractVersion = %q, want well-formed semver", m.ContractVersion)
	}

	// The contract major is the compatibility unit (ADR-024 §3.1): the
	// Compatibility part must declare the same contract version, and
	// the registry metadata format the manifest claims must live on the
	// same major.
	compatibility := readPart(t, root, filepath.Join("compatibility", "README.md"))
	if !strings.Contains(compatibility, wantContract) {
		t.Errorf("compatibility/README.md does not declare the target contract version %q", wantContract)
	}
	if !strings.Contains(compatibility, "major") {
		t.Error("compatibility/README.md does not declare the contract compatibility unit (major version, ADR-024 §3.1)")
	}

	manifestDoc := readPart(t, root, filepath.Join("standard", "README.md"))
	if !strings.Contains(manifestDoc, wantContract) {
		t.Errorf("standard/README.md does not declare the target contract version %q", wantContract)
	}
}

// TestMaintainer_DeclaredAndAccountable verifies the ADR-027
// maintainership bar: the Manifest part declares a maintainer that is
// accountable for the standard — content correctness, template
// freshness, and governed deprecation (ADR-027 §3: standards are owned
// by their maintainers, not by Core; initially Maleo Labs, with
// per-framework owners as the ecosystem grows, Transition Plan §4.3).
// A standard without a declared, accountable maintainer is not accepted
// for publication.
func TestMaintainer_DeclaredAndAccountable(t *testing.T) {
	root := repoRoot(t)
	manifestDoc := readPart(t, root, filepath.Join("standard", "README.md"))

	// Scope every assertion to the Maintainer section itself: a
	// whole-document search would be satisfied by the repository or
	// identity wording appearing elsewhere (e.g. "anvil-standard-flutter"
	// in the Identity table), even if the Maintainer section were empty.
	loc := regexp.MustCompile(`(?m)^## Maintainer`).FindStringIndex(manifestDoc)
	if loc == nil {
		t.Fatal("standard/README.md has no Maintainer section — the ADR-027 maintainership bar requires a declared maintainer")
	}
	maintainer := manifestDoc[loc[0]:]
	if missing := containsAll(maintainer, "Maleo Labs", "accountable", "anvil-standard-flutter", "engineering@maleolabs.com"); len(missing) > 0 {
		t.Errorf("standard/README.md Maintainer section misses %v — the maintainer must be declared, accountable, reachable through an org contact channel, and tied to this standard's repository", missing)
	}
}
