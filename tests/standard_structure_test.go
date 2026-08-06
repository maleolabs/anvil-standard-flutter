// Package tests validates the seven-part structure of this delivery
// lifecycle standard (ADR-021 §3.2, Transition Plan §5.4): every part
// exists in the repository, and the manifest is a consistent declaration
// in the registry metadata format conventions the Anvil Runtime registry
// client reads (registry-metadata.schema.json; Core internal/registry).
//
// These tests concern the standard itself — the Tests part — not an
// adopting project's release (project-facing checks are the Verification
// part).
package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

// repoRoot resolves the standard repository root (the directory
// containing go.mod) from the test package location.
func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test package directory")
		}
		dir = parent
	}
}

// semverPattern is the registry metadata semver pattern
// (registry-metadata.schema.json): major.minor.patch, no leading zeroes.
var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// manifest mirrors the authoring-time fields of the registry metadata
// document (registry-metadata.schema.json conventions; Core
// internal/registry/metadata.go). Release-time fields (distribution,
// lifecycle, trust) are populated by the release pipeline and are not
// part of the source manifest.
type manifest struct {
	ID              string `json:"id"`
	Version         string `json:"version"`
	ContractVersion string `json:"contractVersion"`
	Capability      struct {
		FrameworkVersion []string `json:"frameworkVersion"`
	} `json:"capability"`
}

// TestSevenPartStructureExists verifies that the repository carries the
// seven-part standard structure (ADR-021 §3.2, Transition Plan §5.4):
// Manifest, Lifecycle Definition, Verification, Templates, Compatibility,
// Documentation, Tests.
func TestSevenPartStructureExists(t *testing.T) {
	root := repoRoot(t)

	parts := map[string]string{
		"Manifest":             filepath.Join("standard", "manifest.json"),
		"Lifecycle Definition": filepath.Join("lifecycle", "README.md"),
		"Verification":         filepath.Join("verification", "README.md"),
		"Templates":            filepath.Join("templates", "README.md"),
		"Compatibility":        filepath.Join("compatibility", "README.md"),
		"Documentation":        filepath.Join("docs", "flutter-lifecycle.md"),
		"Tests":                filepath.Join("tests", "README.md"),
	}
	for part, rel := range parts {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("part %q missing: %s (%v)", part, rel, err)
		}
	}
}

// TestManifestDeclaresIdentity verifies the manifest declares the
// standard identity per the registry metadata format conventions: id,
// version, contract version, and framework-version support scope — all
// semver where the format requires semver.
func TestManifestDeclaresIdentity(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "standard", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}

	if m.ID != "anvil-standard-flutter" {
		t.Errorf("id = %q, want %q", m.ID, "anvil-standard-flutter")
	}
	if !semverPattern.MatchString(m.Version) {
		t.Errorf("version = %q, want semver", m.Version)
	}
	if !semverPattern.MatchString(m.ContractVersion) {
		t.Errorf("contractVersion = %q, want semver", m.ContractVersion)
	}
	if len(m.Capability.FrameworkVersion) == 0 {
		t.Fatal("capability.frameworkVersion is empty, want at least one supported framework version")
	}
	for _, v := range m.Capability.FrameworkVersion {
		if !semverPattern.MatchString(v) {
			t.Errorf("capability.frameworkVersion entry %q is not semver", v)
		}
	}
}

// TestManifestContractVersionMatchesCompatibility verifies the manifest's
// declared contract version agrees with the compatibility declaration —
// the two parts of the standard must not drift.
func TestManifestContractVersionMatchesCompatibility(t *testing.T) {
	root := repoRoot(t)

	data, err := os.ReadFile(filepath.Join(root, "standard", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}

	compat, err := os.ReadFile(filepath.Join(root, "compatibility", "README.md"))
	if err != nil {
		t.Fatalf("read compatibility declaration: %v", err)
	}
	if !regexp.MustCompile(`\Q` + m.ContractVersion + `\E`).Match(compat) {
		t.Errorf("compatibility part does not reference the manifest's declared contract version %q", m.ContractVersion)
	}
}
