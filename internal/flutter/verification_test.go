// Tests for the Flutter adapter verification checks (TS-P7-25,
// TS-018-03-02). Checks run against temp artifact-like directories and
// against real tar.gz archive fixtures, so the directory and archive
// access paths are both covered.
package flutter

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"maleolabs.com/anvil-standard-flutter/internal/contracts"
)

// writeArtifactDir creates a temp directory artifact containing the given
// relative files (empty contents) and returns its path.
func writeArtifactDir(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, rel := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

// writeArtifactContents creates a temp directory artifact containing the
// given relative files with the given contents and returns its path.
func writeArtifactContents(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

// writeArtifactStructure creates a temp directory artifact containing the
// given relative files (empty contents) and directories, and returns its
// path. Directories are created first, so files nested inside them are
// placed into the real directories.
func writeArtifactStructure(t *testing.T, files, dirs []string) string {
	t.Helper()
	dir := t.TempDir()
	for _, rel := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, rel), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
	}
	for _, rel := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

// writeArtifactArchive creates a tar.gz archive containing the given
// relative files under the "app/" deployable-content prefix (the Anvil
// artifact convention) and returns its path.
func writeArtifactArchive(t *testing.T, files ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artifact.tar.gz")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()

	gzr := gzip.NewWriter(f)
	tr := tar.NewWriter(gzr)
	for _, rel := range files {
		content := []byte("fixture")
		if err := tr.WriteHeader(&tar.Header{
			Name: filepath.ToSlash(filepath.Join("app", rel)),
			Mode: 0644,
			Size: int64(len(content)),
		}); err != nil {
			t.Fatalf("write header for %s: %v", rel, err)
		}
		if _, err := tr.Write(content); err != nil {
			t.Fatalf("write content for %s: %v", rel, err)
		}
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzr.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return path
}

// writeArtifactArchiveWithDirs creates a tar.gz archive containing the
// given relative files and directories under the "app/" deployable-content
// prefix and returns its path. Directories are written as tar.TypeDir
// entries with a trailing slash, as most tar writers emit them.
func writeArtifactArchiveWithDirs(t *testing.T, files, dirs []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artifact.tar.gz")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()

	gzr := gzip.NewWriter(f)
	tr := tar.NewWriter(gzr)
	for _, rel := range dirs {
		if err := tr.WriteHeader(&tar.Header{
			Name:     filepath.ToSlash(filepath.Join("app", rel)) + "/",
			Mode:     0755,
			Typeflag: tar.TypeDir,
		}); err != nil {
			t.Fatalf("write dir header for %s: %v", rel, err)
		}
	}
	for _, rel := range files {
		content := []byte("fixture")
		if err := tr.WriteHeader(&tar.Header{
			Name: filepath.ToSlash(filepath.Join("app", rel)),
			Mode: 0644,
			Size: int64(len(content)),
		}); err != nil {
			t.Fatalf("write header for %s: %v", rel, err)
		}
		if _, err := tr.Write(content); err != nil {
			t.Fatalf("write content for %s: %v", rel, err)
		}
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzr.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return path
}

// writeArtifactArchiveContents creates a tar.gz archive containing the
// given relative files with the given contents under the "app/"
// deployable-content prefix and returns its path.
func writeArtifactArchiveContents(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artifact.tar.gz")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()

	gzr := gzip.NewWriter(f)
	tr := tar.NewWriter(gzr)
	for rel, content := range files {
		if err := tr.WriteHeader(&tar.Header{
			Name: filepath.ToSlash(filepath.Join("app", rel)),
			Mode: 0644,
			Size: int64(len(content)),
		}); err != nil {
			t.Fatalf("write header for %s: %v", rel, err)
		}
		if _, err := tr.Write([]byte(content)); err != nil {
			t.Fatalf("write content for %s: %v", rel, err)
		}
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzr.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return path
}

// TestRunVerification_PubspecYaml verifies the pubspec_yaml check passes
// when pubspec.yaml exists in the artifact directory and fails with a
// descriptive detail when it is absent (TS-P7-25 AC-1, AC-3).
func TestRunVerification_PubspecYaml(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckPubspecYaml,
			ArtifactPath: writeArtifactDir(t, "pubspec.yaml"),
		})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
		if outcome.Name != CheckPubspecYaml {
			t.Errorf("Name = %q, want %q", outcome.Name, CheckPubspecYaml)
		}
		if outcome.Details == "" {
			t.Error("Details = empty, want a description of what was validated")
		}
	})

	t.Run("fail", func(t *testing.T) {
		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckPubspecYaml,
			ArtifactPath: writeArtifactDir(t, "lib/main.dart"),
		})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		if !strings.Contains(outcome.Details, "pubspec.yaml") {
			t.Errorf("Details = %q, want mention of the missing pubspec.yaml", outcome.Details)
		}
	})
}

// TestRunVerification_LibDirectory verifies the lib_directory check
// passes when the lib/ directory exists in the artifact directory and
// fails with a descriptive detail when it is absent (TS-P7-25 AC-2,
// AC-3).
func TestRunVerification_LibDirectory(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckLibDirectory,
			ArtifactPath: writeArtifactStructure(t, []string{"lib/main.dart"}, []string{"lib"}),
		})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
		if outcome.Name != CheckLibDirectory {
			t.Errorf("Name = %q, want %q", outcome.Name, CheckLibDirectory)
		}
		if outcome.Details == "" {
			t.Error("Details = empty, want a description of what was validated")
		}
	})

	t.Run("fail", func(t *testing.T) {
		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckLibDirectory,
			ArtifactPath: writeArtifactDir(t, "pubspec.yaml"),
		})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		if !strings.Contains(outcome.Details, "missing required directory: lib") {
			t.Errorf("Details = %q, want the descriptive missing message", outcome.Details)
		}
	})
}

// TestRunVerification_LibDirectory_FileNotDirectory verifies that a
// regular file named "lib" does not satisfy the lib_directory check —
// only a real directory counts (TS-P7-25 AC-2).
func TestRunVerification_LibDirectory_FileNotDirectory(t *testing.T) {
	t.Run("directory_artifact", func(t *testing.T) {
		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckLibDirectory,
			ArtifactPath: writeArtifactDir(t, "lib"),
		})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
	})

	t.Run("archive", func(t *testing.T) {
		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckLibDirectory,
			ArtifactPath: writeArtifactArchive(t, "lib"),
		})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
	})
}

// TestRunVerification_Archive verifies that both checks also pass against
// a real tar.gz artifact archive with the "app/" deployable-content
// prefix, and fail when the entries are absent — no full extraction is
// performed (TS-P7-25 AC-1..AC-3).
func TestRunVerification_Archive(t *testing.T) {
	t.Run("pass_existing", func(t *testing.T) {
		// The archive contains pubspec.yaml and explicit TypeDir
		// entries for lib/, as most tar writers emit them.
		artifactPath := writeArtifactArchiveWithDirs(t,
			[]string{"pubspec.yaml", "lib/main.dart"},
			[]string{"lib"},
		)
		for _, check := range []string{CheckPubspecYaml, CheckLibDirectory} {
			outcome := RunVerification(contracts.VerificationRequest{Check: check, ArtifactPath: artifactPath})
			if !outcome.Passed {
				t.Errorf("%s: Passed = false, want true (outcome: %#v)", check, outcome)
			}
		}
	})

	t.Run("pass_no_dir_entries", func(t *testing.T) {
		// Anvil packaging stores only regular files, never directory
		// entries: a directory still counts when entries live beneath
		// it.
		artifactPath := writeArtifactArchive(t, "pubspec.yaml", "lib/main.dart")
		for _, check := range []string{CheckPubspecYaml, CheckLibDirectory} {
			outcome := RunVerification(contracts.VerificationRequest{Check: check, ArtifactPath: artifactPath})
			if !outcome.Passed {
				t.Errorf("%s: Passed = false, want true (outcome: %#v)", check, outcome)
			}
		}
	})

	t.Run("fail", func(t *testing.T) {
		// The archive has no pubspec.yaml and no lib/ entries.
		artifactPath := writeArtifactArchive(t, "README.md")
		tests := []struct {
			check         string
			missingDetail string
		}{
			{check: CheckPubspecYaml, missingDetail: "pubspec.yaml"},
			{check: CheckLibDirectory, missingDetail: "missing required directory: lib"},
		}
		for _, tt := range tests {
			outcome := RunVerification(contracts.VerificationRequest{Check: tt.check, ArtifactPath: artifactPath})
			if outcome.Passed {
				t.Errorf("%s: Passed = true, want false (outcome: %#v)", tt.check, outcome)
			}
			if !strings.Contains(outcome.Details, tt.missingDetail) {
				t.Errorf("%s: Details = %q, want it to contain %q", tt.check, outcome.Details, tt.missingDetail)
			}
		}
	})
}

// TestRunVerification_UnknownCheck verifies that an undeclared check
// yields a failing outcome with descriptive details, not a panic.
func TestRunVerification_UnknownCheck(t *testing.T) {
	outcome := RunVerification(contracts.VerificationRequest{Check: "unknown_check", ArtifactPath: t.TempDir()})
	if outcome.Passed {
		t.Error("Passed = true, want false")
	}
	if !strings.Contains(outcome.Details, `unknown verification check "unknown_check"`) {
		t.Errorf("Details = %q, want mention of the unknown check", outcome.Details)
	}
}

// TestRunVerification_MissingArtifact verifies that an unreadable artifact
// path yields a failing outcome with an actionable detail.
func TestRunVerification_MissingArtifact(t *testing.T) {
	outcome := RunVerification(contracts.VerificationRequest{
		Check:        CheckPubspecYaml,
		ArtifactPath: filepath.Join(t.TempDir(), "does-not-exist"),
	})
	if outcome.Passed {
		t.Error("Passed = true, want false")
	}
	if !strings.Contains(outcome.Details, "cannot inspect artifact") {
		t.Errorf("Details = %q, want mention of the inspection failure", outcome.Details)
	}
}

// ---------------------------------------------------------------------------
// Lifecycle-conformity checks (TS-018-03-02).
// ---------------------------------------------------------------------------

// pubspecFixture is a realistic Flutter pubspec.yaml: a dependencies
// section with a hosted dependency and the SDK dependency, plus a
// dev_dependencies section with the SDK test dependency — the canonical
// shape `flutter create` writes.
const pubspecFixture = `name: my_app
description: "A Flutter application"
publish_to: 'none'
version: 1.0.0+1

environment:
  sdk: '>=3.0.0 <4.0.0'

dependencies:
  flutter:
    sdk: flutter
  http: ^1.0.0

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^3.0.0
`

// lockFixture is a realistic pubspec.lock covering the pubspecFixture
// dependencies: the packages section lists the SDK packages (flutter,
// flutter_test) and the hosted ones (http, flutter_lints), followed by
// the sdks section — the canonical shape `pub get` writes.
const lockFixture = `# Generated by pub on 2026-08-01 10:00:00.000000Z
packages:
  async:
    dependency: transitive
    description:
      name: async
      url: "https://pub.dev"
    source: hosted
    version: "2.11.0"
  flutter:
    dependency: "direct main"
    description: flutter
    source: sdk
    version: "0.0.0"
  flutter_lints:
    dependency: "direct dev"
    description:
      name: flutter_lints
      url: "https://pub.dev"
    source: hosted
    version: "3.0.2"
  flutter_test:
    dependency: "direct dev"
    description: flutter
    source: sdk
    version: "0.0.0"
  http:
    dependency: "direct main"
    description:
      name: http
      url: "https://pub.dev"
    source: hosted
    version: "1.2.0"
sdks:
  dart: ">=3.0.0 <4.0.0"
  flutter: ">=3.10.0"
`

// TestPubspecParsers pins the line-oriented pubspec parsers: the
// declared dependencies of a canonical pubspec.yaml and the pinned
// packages of a canonical pubspec.lock.
func TestPubspecParsers(t *testing.T) {
	t.Run("declared_dependencies", func(t *testing.T) {
		deps := pubspecDeclaredDependencies([]byte(pubspecFixture))
		for _, want := range []string{"flutter", "http", "flutter_test", "flutter_lints"} {
			if !slices.Contains(deps, want) {
				t.Errorf("declared dependencies = %v, want it to contain %q", deps, want)
			}
		}
		if len(deps) != 4 {
			t.Errorf("declared dependencies length = %d, want 4 (got %v)", len(deps), deps)
		}
	})

	t.Run("locked_packages", func(t *testing.T) {
		pkgs := pubspecLockPackages([]byte(lockFixture))
		for _, want := range []string{"async", "flutter", "flutter_lints", "flutter_test", "http"} {
			if !slices.Contains(pkgs, want) {
				t.Errorf("locked packages = %v, want it to contain %q", pkgs, want)
			}
		}
	})
}

// TestRunVerification_DependencyLockfile verifies the shared-resource
// wiring check (TS-018-03-02): the release's locked dependency set must
// be present.
func TestRunVerification_DependencyLockfile(t *testing.T) {
	t.Run("pass_lockfile_present", func(t *testing.T) {
		artifactPath := writeArtifactContents(t, map[string]string{
			"pubspec.yaml":  pubspecFixture,
			"pubspec.lock":  lockFixture,
			"lib/main.dart": "void main() {}",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyLockfile, ArtifactPath: artifactPath})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
		for _, want := range []string{"pubspec.lock", "wired", "pub_get"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("fail_lockfile_missing", func(t *testing.T) {
		artifactPath := writeArtifactContents(t, map[string]string{
			"pubspec.yaml":  pubspecFixture,
			"lib/main.dart": "void main() {}",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyLockfile, ArtifactPath: artifactPath})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		for _, want := range []string{"pubspec.lock", "not wired"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("archive_pass", func(t *testing.T) {
		artifactPath := writeArtifactArchiveContents(t, map[string]string{
			"pubspec.yaml":  pubspecFixture,
			"pubspec.lock":  lockFixture,
			"lib/main.dart": "void main() {}",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyLockfile, ArtifactPath: artifactPath})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
	})

	t.Run("archive_fail", func(t *testing.T) {
		artifactPath := writeArtifactArchiveContents(t, map[string]string{
			"pubspec.yaml":  pubspecFixture,
			"lib/main.dart": "void main() {}",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyLockfile, ArtifactPath: artifactPath})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
	})
}

// TestRunVerification_DependencyTiming verifies the dependency-resolution
// timing check (TS-018-03-02): the release must carry the manifest and
// the locked set, and the locked set must cover the declared
// dependencies — pre-promotion resolution reproduces the built set.
func TestRunVerification_DependencyTiming(t *testing.T) {
	t.Run("pass_manifest_and_lock_consistent", func(t *testing.T) {
		artifactPath := writeArtifactContents(t, map[string]string{
			"pubspec.yaml": pubspecFixture,
			"pubspec.lock": lockFixture,
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyTiming, ArtifactPath: artifactPath})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
		for _, want := range []string{"pre-promotion", "pubspec.lock", "4 declared dependency"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("fail_no_manifest", func(t *testing.T) {
		artifactPath := writeArtifactContents(t, map[string]string{
			"pubspec.lock":  lockFixture,
			"lib/main.dart": "void main() {}",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyTiming, ArtifactPath: artifactPath})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		for _, want := range []string{"pubspec.yaml", "no re-checkable evidence"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("fail_no_lockfile", func(t *testing.T) {
		artifactPath := writeArtifactContents(t, map[string]string{
			"pubspec.yaml":  pubspecFixture,
			"lib/main.dart": "void main() {}",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyTiming, ArtifactPath: artifactPath})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		for _, want := range []string{"pubspec.lock", "unlocked set"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("fail_declared_dependency_not_locked", func(t *testing.T) {
		// The manifest declares "http", the lock does not pin it.
		artifactPath := writeArtifactContents(t, map[string]string{
			"pubspec.yaml": pubspecFixture,
			"pubspec.lock": strings.Replace(lockFixture, "  http:\n", "", 1),
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyTiming, ArtifactPath: artifactPath})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		for _, want := range []string{"http", "not locked"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("fail_non_canonical_manifest_fails_closed", func(t *testing.T) {
		// A manifest that does not match the canonical two-space shape
		// (here: four-space indentation) yields no extractable declared
		// dependencies — the evidence cannot be re-checked, so the
		// check fails closed rather than passing vacuously.
		artifactPath := writeArtifactContents(t, map[string]string{
			"pubspec.yaml": "name: my_app\n\ndependencies:\n    http: ^1.0.0\n",
			"pubspec.lock": lockFixture,
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyTiming, ArtifactPath: artifactPath})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		if !strings.Contains(outcome.Details, "cannot be re-checked") {
			t.Errorf("Details = %q, want it to report the un-re-checkable evidence", outcome.Details)
		}
	})

	t.Run("archive_pass", func(t *testing.T) {
		artifactPath := writeArtifactArchiveContents(t, map[string]string{
			"pubspec.yaml": pubspecFixture,
			"pubspec.lock": lockFixture,
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyTiming, ArtifactPath: artifactPath})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
	})

	t.Run("archive_fail", func(t *testing.T) {
		artifactPath := writeArtifactArchiveContents(t, map[string]string{
			"pubspec.yaml": pubspecFixture,
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckDependencyTiming, ArtifactPath: artifactPath})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
	})
}

// TestRunVerification_PlatformSyncReady verifies the platform-step check
// (TS-018-03-02): a release carrying ios/ must carry ios/Podfile — the
// platform_sync input; a release without ios/ matches the phase's
// informational no-op.
func TestRunVerification_PlatformSyncReady(t *testing.T) {
	t.Run("pass_ios_with_podfile", func(t *testing.T) {
		artifactPath := writeArtifactContents(t, map[string]string{
			"ios/Podfile": "platform :ios, '12.0'\n",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckPlatformSyncReady, ArtifactPath: artifactPath})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
		for _, want := range []string{"ios/Podfile", "platform_sync"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("pass_no_ios_directory", func(t *testing.T) {
		artifactPath := writeArtifactContents(t, map[string]string{
			"pubspec.yaml":  pubspecFixture,
			"lib/main.dart": "void main() {}",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckPlatformSyncReady, ArtifactPath: artifactPath})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
		for _, want := range []string{"no ios/ directory", "informational no-op"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("fail_ios_without_podfile", func(t *testing.T) {
		artifactPath := writeArtifactContents(t, map[string]string{
			"ios/Podfile.lock": "PODS:\n",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckPlatformSyncReady, ArtifactPath: artifactPath})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		for _, want := range []string{"ios/Podfile", "missing"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("archive_pass", func(t *testing.T) {
		artifactPath := writeArtifactArchiveContents(t, map[string]string{
			"ios/Podfile": "platform :ios, '12.0'\n",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckPlatformSyncReady, ArtifactPath: artifactPath})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
	})

	t.Run("archive_fail", func(t *testing.T) {
		artifactPath := writeArtifactArchiveContents(t, map[string]string{
			"ios/Podfile.lock": "PODS:\n",
		})
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckPlatformSyncReady, ArtifactPath: artifactPath})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
	})
}

// TestRunVerification_RollbackBehavior verifies the rollback-behavior
// check (TS-018-03-02): rollback produces the declared state — every
// phase declares rollback coverage, and the manifest rollback metadata
// matches the phase table.
func TestRunVerification_RollbackBehavior(t *testing.T) {
	t.Run("pass_declared_semantics", func(t *testing.T) {
		artifactPath := writeArtifactDir(t, "pubspec.yaml")
		outcome := RunVerification(contracts.VerificationRequest{Check: CheckRollbackBehavior, ArtifactPath: artifactPath})
		if !outcome.Passed {
			t.Errorf("Passed = false, want true (outcome: %#v)", outcome)
		}
		for _, want := range []string{
			"rollback produces the declared state",
			"pub_get: pub get",
			"informational (irreversible, rollback never blocks)",
			"manifest rollback metadata matches",
		} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("fail_missing_artifact", func(t *testing.T) {
		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckRollbackBehavior,
			ArtifactPath: filepath.Join(t.TempDir(), "does-not-exist"),
		})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		if !strings.Contains(outcome.Details, "cannot inspect artifact") {
			t.Errorf("Details = %q, want it to mention the inspection failure", outcome.Details)
		}
	})

	t.Run("fail_irreversible_phase_with_command", func(t *testing.T) {
		original := activationPhases
		activationPhases = []activationPhase{
			{name: "pub_get", program: "flutter", activateArgs: []string{"pub", "get"}, rollbackArgs: []string{"pub", "get"}},
			{name: "platform_sync", program: "pod", activateArgs: []string{"install"}, irreversible: true, rollbackArgs: []string{"install"}},
		}
		defer func() { activationPhases = original }()

		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckRollbackBehavior,
			ArtifactPath: t.TempDir(),
		})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		for _, want := range []string{"platform_sync", "irreversible", "rollback command"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("fail_reversible_phase_without_command", func(t *testing.T) {
		original := activationPhases
		activationPhases = []activationPhase{
			{name: "pub_get", program: "flutter", activateArgs: []string{"pub", "get"}},
		}
		defer func() { activationPhases = original }()

		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckRollbackBehavior,
			ArtifactPath: t.TempDir(),
		})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		for _, want := range []string{"pub_get", "rollback coverage is missing"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})

	t.Run("fail_manifest_rollback_drift", func(t *testing.T) {
		// The phase table declares a second reversible phase, so the
		// manifest rollback surface would drift from the declared
		// single re-resolution.
		original := activationPhases
		activationPhases = []activationPhase{
			{name: "pub_get", program: "flutter", activateArgs: []string{"pub", "get"}, rollbackArgs: []string{"pub", "get"}},
			{name: "pub_upgrade", program: "flutter", activateArgs: []string{"pub", "upgrade"}, rollbackArgs: []string{"pub", "get"}},
		}
		defer func() { activationPhases = original }()

		outcome := RunVerification(contracts.VerificationRequest{
			Check:        CheckRollbackBehavior,
			ArtifactPath: t.TempDir(),
		})
		if outcome.Passed {
			t.Errorf("Passed = true, want false (outcome: %#v)", outcome)
		}
		for _, want := range []string{"manifest rollback metadata", "drifts from the executable phase table"} {
			if !strings.Contains(outcome.Details, want) {
				t.Errorf("Details = %q, want it to contain %q", outcome.Details, want)
			}
		}
	})
}

// TestCapabilities_DeclaresChecks verifies that the capability declaration
// lists exactly the six Flutter verification checks — the two structural
// checks (pubspec_yaml and lib_directory, TS-P7-25) and the four
// lifecycle-conformity checks (TS-018-03-02) — with descriptions
// (TS-P7-25 DoD, TS-018-03-02 DoD, TS-P7-08 AC-3).
func TestCapabilities_DeclaresChecks(t *testing.T) {
	result := Capabilities()
	checks := result.Declaration.VerificationChecks
	if len(checks) != 6 {
		t.Fatalf("VerificationChecks length = %d, want 6", len(checks))
	}
	want := []string{
		CheckPubspecYaml,
		CheckLibDirectory,
		CheckDependencyLockfile,
		CheckDependencyTiming,
		CheckPlatformSyncReady,
		CheckRollbackBehavior,
	}
	for i, name := range want {
		if checks[i].Name != name {
			t.Errorf("VerificationChecks[%d].Name = %q, want %q", i, checks[i].Name, name)
		}
		if checks[i].Description == "" {
			t.Errorf("VerificationChecks[%d].Description = empty, want a description", i)
		}
	}
}
