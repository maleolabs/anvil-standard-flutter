// Verification checks of the Flutter adapter (TS-P7-25, TS-018-03-02).
//
// Each check validates a Flutter-specific file or structure inside the
// artifact under verification and returns a contracts.VerificationOutcome
// (pass/fail + details), aligned with artifact.CheckResult so outcomes
// merge into the Core's verification report without transformation.
//
// The checks come in two categories (ADR-033 §3, 007 §6): the structural
// checks (the preserved v1.x surface — files and directories exist) and
// the lifecycle-conformity checks (TS-018-03-02) — shared-resource
// wiring, dependency-resolution timing relative to promotion, the
// platform step at its declared lifecycle point, and rollback behavior
// (Review 19 §3.3, adapted to the hybrid deployment model, ADR-016). The
// lifecycle-conformity checks are standard-supplied content against the
// verification contract (verification-contract.md, specification
// corpus): gate semantics and evidence requirements belong to the
// contract; the rules below are the Flutter standard's content, additive
// only — gates are never weakened.
//
// The artifact path may be either a directory (the extracted artifact,
// the common case in tests) or an Anvil artifact archive (tar.gz). For
// archives, entries are scanned directly — no full extraction is
// performed (known-risk decision carried over from the pre-split
// adapter, docs/sessions in the Core repository §Known Risks).
package flutter

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"maleolabs.com/anvil-standard-flutter/internal/contracts"
)

// Verification check names declared in the capability declaration
// (Capabilities). Check names are part of the adapter's contract surface:
// the Core invokes only declared checks (TS-P7-08 AC-3).
//
// Reference: TS-P7-25 AC-1..AC-3, TS-018-03-02
const (
	// CheckPubspecYaml validates that pubspec.yaml exists in the
	// project root — the Flutter project manifest.
	CheckPubspecYaml = "pubspec_yaml"

	// CheckLibDirectory validates that the lib/ directory exists in
	// the project root — the Flutter application source directory.
	CheckLibDirectory = "lib_directory"

	// CheckDependencyLockfile validates the shared-resource wiring of
	// the release: the locked dependency set (pubspec.lock) must be
	// present in the release so activation's pub_get re-resolves
	// exactly the set the artifact was built from (TS-018-03-02:
	// shared-resource wiring, Review 19 §3.3; the hybrid analog of
	// Laravel's .env/storage wiring).
	CheckDependencyLockfile = "dependency_lockfile"

	// CheckDependencyTiming validates the dependency-resolution timing
	// evidence of the release: the declared pre-promotion pub_get phase
	// (TS-018-02-01 — pub_get runs BEFORE promotion) needs the locked
	// set that matches the declared manifest, so resolution before
	// promotion reproduces the built set (TS-018-03-02: dependency-
	// resolution timing relative to promotion, Review 19 §3.3; the
	// hybrid analog of Laravel's migration timing).
	CheckDependencyTiming = "dependency_timing"

	// CheckPlatformSyncReady validates the platform-step evidence of
	// the release: when the release contains an ios/ directory (the
	// platform_sync phase's requiresDir), the platform integration the
	// phase finalizes must be present — ios/Podfile, the pod install
	// input; without ios/ the phase is an informational no-op and the
	// check passes with that note (TS-018-03-02: lifecycle-point
	// conformity — the platform step at its declared point, Review 19
	// §3.3; the hybrid analog of queue restart, which the hybrid model
	// does not declare — there is no server-side queue to restart).
	CheckPlatformSyncReady = "platform_sync_ready"

	// CheckRollbackBehavior validates that rollback produces the
	// declared state: every activation phase declares its rollback
	// coverage (a rollback command when reversible, the irreversible
	// marker when not), and the manifest rollback metadata matches the
	// executable phase table (TS-018-03-02: rollback behavior, Review
	// 19 §3.3; TS-018-02-01 — pub_get rollback is the idempotent
	// re-resolution, platform_sync is irreversible and never blocks
	// rollback).
	CheckRollbackBehavior = "rollback_behavior"
)

// RunVerification executes one verification check against the artifact
// path and returns the pass/fail outcome.
//
// Reference: TS-P7-25 AC-1..AC-3, TS-018-03-02
func RunVerification(req contracts.VerificationRequest) contracts.VerificationOutcome {
	switch req.Check {
	case CheckPubspecYaml:
		return checkFiles(req.ArtifactPath, CheckPubspecYaml, "pubspec.yaml")
	case CheckLibDirectory:
		return checkDirectory(req.ArtifactPath, CheckLibDirectory, "lib")
	case CheckDependencyLockfile:
		return checkDependencyLockfile(req.ArtifactPath)
	case CheckDependencyTiming:
		return checkDependencyTiming(req.ArtifactPath)
	case CheckPlatformSyncReady:
		return checkPlatformSyncReady(req.ArtifactPath)
	case CheckRollbackBehavior:
		return checkRollbackBehavior(req.ArtifactPath)
	default:
		return contracts.VerificationOutcome{
			Name:    req.Check,
			Passed:  false,
			Details: fmt.Sprintf("unknown verification check %q", req.Check),
		}
	}
}

// checkFiles verifies that every required relative path exists in the
// artifact (directory or archive) and reports a single outcome for the
// check.
func checkFiles(artifactPath, checkName string, required ...string) contracts.VerificationOutcome {
	var missing []string
	for _, rel := range required {
		found, err := artifactContains(artifactPath, rel)
		if err != nil {
			return contracts.VerificationOutcome{
				Name:    checkName,
				Passed:  false,
				Details: fmt.Sprintf("cannot inspect artifact %q: %v", artifactPath, err),
			}
		}
		if !found {
			missing = append(missing, rel)
		}
	}

	if len(missing) > 0 {
		return contracts.VerificationOutcome{
			Name:    checkName,
			Passed:  false,
			Details: fmt.Sprintf("missing required file(s): %s", strings.Join(missing, ", ")),
		}
	}
	return contracts.VerificationOutcome{
		Name:    checkName,
		Passed:  true,
		Details: fmt.Sprintf("required file(s) found: %s", strings.Join(required, ", ")),
	}
}

// checkDirectory verifies that a required directory exists inside the
// artifact (directory or archive) and reports a single outcome for the
// check.
func checkDirectory(artifactPath, checkName, requiredDir string) contracts.VerificationOutcome {
	found, err := artifactContainsDir(artifactPath, requiredDir)
	if err != nil {
		return contracts.VerificationOutcome{
			Name:    checkName,
			Passed:  false,
			Details: fmt.Sprintf("cannot inspect artifact %q: %v", artifactPath, err),
		}
	}
	if !found {
		return contracts.VerificationOutcome{
			Name:    checkName,
			Passed:  false,
			Details: fmt.Sprintf("missing required directory: %s", requiredDir),
		}
	}
	return contracts.VerificationOutcome{
		Name:    checkName,
		Passed:  true,
		Details: fmt.Sprintf("required directory found: %s", requiredDir),
	}
}

// artifactContains reports whether relPath exists inside the artifact at
// artifactPath. When artifactPath is a directory, the path is resolved
// directly; otherwise the path is treated as a tar.gz archive and its
// entries are scanned. Anvil artifact archives store deployable content
// under the "app/" prefix (artifact.DeployableContentDir); both prefixed
// and unprefixed entries are accepted so plain directories and archives
// behave consistently.
func artifactContains(artifactPath, relPath string) (bool, error) {
	info, err := os.Stat(artifactPath)
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		if _, err := os.Stat(filepath.Join(artifactPath, relPath)); err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	return archiveContains(artifactPath, relPath)
}

// artifactContainsDir reports whether the directory relPath exists inside
// the artifact at artifactPath (directory or tar.gz archive). Unlike
// artifactContains, a matching entry must be a directory, not a file.
func artifactContainsDir(artifactPath, relPath string) (bool, error) {
	info, err := os.Stat(artifactPath)
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		dirInfo, err := os.Stat(filepath.Join(artifactPath, relPath))
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		return dirInfo.IsDir(), nil
	}
	return archiveContainsDir(artifactPath, relPath)
}

// scanArchive opens the tar.gz archive and calls match for each entry —
// with the optional "app/" deployable-content prefix stripped from the
// entry name — until match returns true. The returned bool reports
// whether any entry matched.
func scanArchive(archivePath string, match func(name string, hdr *tar.Header) bool) (bool, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return false, fmt.Errorf("not a gzip archive: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, err
		}
		if match(strings.TrimPrefix(hdr.Name, "app/"), hdr) {
			return true, nil
		}
	}
	return false, nil
}

// archiveContains scans the entries of a tar.gz archive for relPath,
// accepting an optional "app/" prefix on entry names. Only regular file
// entries (tar.TypeReg) count.
func archiveContains(archivePath, relPath string) (bool, error) {
	return scanArchive(archivePath, func(name string, hdr *tar.Header) bool {
		return name == relPath && hdr.Typeflag == tar.TypeReg
	})
}

// archiveContainsDir scans the entries of a tar.gz archive for evidence
// of the directory relPath. Two forms count as evidence:
//
//  1. An explicit directory entry: tar.TypeDir entries, or regular file
//     entries whose name ends with "/" (some tar writers store
//     directories that way).
//  2. A regular entry living beneath the directory — Anvil artifact
//     archives never contain directory entries (packaging only stores
//     regular files), so a directory is present whenever an entry is
//     stored under its path.
//
// The optional "app/" deployable-content prefix is stripped before
// matching, so prefixed and unprefixed archives behave consistently.
func archiveContainsDir(archivePath, relPath string) (bool, error) {
	return scanArchive(archivePath, func(name string, hdr *tar.Header) bool {
		isDirMarker := hdr.Typeflag == tar.TypeDir ||
			(hdr.Typeflag == tar.TypeReg && strings.HasSuffix(name, "/"))
		if isDirMarker && strings.TrimSuffix(name, "/") == relPath {
			return true
		}
		return strings.HasPrefix(name, relPath+"/")
	})
}

// ---------------------------------------------------------------------------
// Lifecycle-conformity checks (TS-018-03-02, ADR-033 §3, Review 19 §3.3).
//
// The lifecycle-conformity checks are standard-supplied content against
// the verification contract (verification-contract.md, specification
// corpus): gate semantics and evidence requirements are the contract's;
// the rules below are the Flutter standard's, adapted to the hybrid
// deployment model (ADR-016). Evidence is re-checkable — it is embedded
// in the release artifact (the locked dependency set, the manifest/lock
// pair, the platform integration input, the declared phase table) so any
// consumer can re-run the check and derive the same outcome, and
// outcomes merge into the runtime's verification report in the standard
// outcome shape (name, passed, details).
//
// The hybrid model declares no server-side queue, so the queue-restart
// item of Review 19 §3.3 does not apply; its lifecycle-point position is
// carried by the platform step (platform_sync) at its declared point
// (TS-018-02-01; lifecycle/README.md §Semantics — "Convention depth
// where applicable").
// ---------------------------------------------------------------------------

// passOutcome and failOutcome build the standard outcome shape with the
// check name and a details string.
func passOutcome(checkName, details string) contracts.VerificationOutcome {
	return contracts.VerificationOutcome{Name: checkName, Passed: true, Details: details}
}

func failOutcome(checkName, details string) contracts.VerificationOutcome {
	return contracts.VerificationOutcome{Name: checkName, Passed: false, Details: details}
}

// pubspecEntryPattern matches an entry line of a pubspec.yaml
// dependencies/dev_dependencies section or a pubspec.lock packages
// section — two-space-indented, the dependency/package name followed by
// ":" (e.g. "  http: ^1.0.0", the multi-line "  flutter:" form, or the
// lock's "  async:"). The canonical shapes `pub get` and `flutter create`
// write are two-space-indented, so the pattern is pinned to that shape.
var pubspecEntryPattern = regexp.MustCompile(`^  ([a-zA-Z0-9_\-]+):`)

// pubspecDeclaredDependencies returns the dependency names declared in
// the dependencies and dev_dependencies sections of a pubspec.yaml
// content. The scan is line-oriented: within each section, every
// two-space-indented "name:" entry counts. The parser is deliberately
// minimal — it reads the declared dependency names of the canonical
// Flutter pubspec shape, not the full YAML grammar (nested keys such as
// "sdk: flutter" are indented deeper and do not match).
func pubspecDeclaredDependencies(data []byte) []string {
	var deps []string
	inDeps := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !startsWithIndent(line) {
			// A top-level key: it opens the dependencies/
			// dev_dependencies sections or ends the current section.
			inDeps = strings.HasPrefix(line, "dependencies:") ||
				strings.HasPrefix(line, "dev_dependencies:")
			continue
		}
		if !inDeps {
			continue
		}
		if m := pubspecEntryPattern.FindStringSubmatch(line); m != nil {
			deps = append(deps, m[1])
		}
	}
	return deps
}

// pubspecLockPackages returns the package names pinned in the packages
// section of a pubspec.lock content — the names `pub get` wrote. The
// scan is line-oriented: within the packages section, every
// two-space-indented "name:" entry counts; the trailing sdks section is
// a top-level key that ends the scan.
func pubspecLockPackages(data []byte) []string {
	var pkgs []string
	inPackages := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !startsWithIndent(line) {
			// A top-level key: "packages:" opens the section, any
			// other (e.g. "sdks:") ends it.
			inPackages = strings.HasPrefix(line, "packages:")
			continue
		}
		if !inPackages {
			continue
		}
		if m := pubspecEntryPattern.FindStringSubmatch(line); m != nil {
			pkgs = append(pkgs, m[1])
		}
	}
	return pkgs
}

// startsWithIndent reports whether the line begins with a space or tab —
// an indented (nested) line in the canonical YAML shape, as opposed to a
// top-level key.
func startsWithIndent(line string) bool {
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}

// checkDependencyLockfile implements CheckDependencyLockfile: the
// release's locked dependency set — the hybrid model's shared resource —
// must be present in the artifact. Activation's pub_get phase re-resolves
// from pubspec.lock before promotion; a release without its lockfile
// would resolve an unlocked, drifting set, so the release would not serve
// what it was built from.
//
// Reference: TS-018-03-02, Review 19 §3.3, TS-018-02-01
func checkDependencyLockfile(artifactPath string) contracts.VerificationOutcome {
	found, err := artifactContains(artifactPath, "pubspec.lock")
	if err != nil {
		return failOutcome(CheckDependencyLockfile,
			fmt.Sprintf("cannot inspect artifact %q: %v", artifactPath, err))
	}
	if !found {
		return failOutcome(CheckDependencyLockfile,
			"missing pubspec.lock: the release's locked dependency set is not wired — activation's pub_get would resolve an unlocked set, so the release would not serve exactly what it was built from")
	}
	return passOutcome(CheckDependencyLockfile,
		"locked dependency set wired for the release: pubspec.lock present — activation's pub_get re-resolves exactly the set the artifact was built from")
}

// checkDependencyTiming implements CheckDependencyTiming: re-checkable
// evidence that dependency resolution can run at the declared timing.
// The standard declares pub_get as the first activation phase, at
// pre-promotion timing (TS-018-02-01 — dependency resolution before
// promotion, the hybrid analog of Laravel's migration timing), so the
// artifact must carry both the manifest (pubspec.yaml) and the locked
// set (pubspec.lock), and the locked set must cover the declared
// dependencies — resolution before promotion must reproduce the built
// set, not resolve a different one.
//
// Reference: TS-018-03-02, Review 19 §3.3, TS-018-02-01
func checkDependencyTiming(artifactPath string) contracts.VerificationOutcome {
	manifestData, found, err := artifactReadFile(artifactPath, "pubspec.yaml")
	if err != nil {
		return failOutcome(CheckDependencyTiming,
			fmt.Sprintf("cannot inspect pubspec.yaml: %v", err))
	}
	if !found {
		return failOutcome(CheckDependencyTiming,
			"no pubspec.yaml in the release: the declared dependency manifest is absent, so the pre-promotion resolution timing has no re-checkable evidence")
	}

	lockData, found, err := artifactReadFile(artifactPath, "pubspec.lock")
	if err != nil {
		return failOutcome(CheckDependencyTiming,
			fmt.Sprintf("cannot inspect pubspec.lock: %v", err))
	}
	if !found {
		return failOutcome(CheckDependencyTiming,
			"no pubspec.lock in the release: the pre-promotion pub_get phase would resolve an unlocked set — dependency resolution cannot reproduce the built set at the declared timing")
	}

	declared := pubspecDeclaredDependencies(manifestData)
	locked := pubspecLockPackages(lockData)

	// A canonical Flutter app manifest always declares the flutter SDK
	// dependency; an empty extraction means the manifest does not match
	// the canonical two-space shape, so the timing evidence cannot be
	// re-checked and the check fails closed.
	if len(declared) == 0 {
		return failOutcome(CheckDependencyTiming,
			"no dependency entries found in pubspec.yaml in the canonical two-space shape: the declared dependency set cannot be re-checked")
	}

	var unlocked []string
	for _, name := range declared {
		if !slices.Contains(locked, name) {
			unlocked = append(unlocked, name)
		}
	}
	if len(unlocked) > 0 {
		return failOutcome(CheckDependencyTiming, fmt.Sprintf(
			"declared dependency(ies) not locked: %s — the locked set does not cover the manifest, so pre-promotion resolution would change the dependency set instead of reproducing the built one",
			strings.Join(unlocked, ", ")))
	}

	return passOutcome(CheckDependencyTiming, fmt.Sprintf(
		"pre-promotion dependency-resolution timing evidence present: pubspec.yaml and pubspec.lock in the release, %d declared dependency(ies) covered by the locked set; declared timing: %s",
		len(declared), "pub_get before promotion"))
}

// checkPlatformSyncReady implements CheckPlatformSyncReady: the platform
// step (platform_sync) can run at its declared lifecycle point. The
// phase applies when the release working directory contains an ios/
// directory (TS-018-02-01 requiresDir); the pod install input for that
// platform state is ios/Podfile, so a release carrying ios/ without its
// Podfile cannot finalize the platform integration. A release without
// ios/ matches the phase's informational no-op and passes with that
// note.
//
// Reference: TS-018-03-02, Review 19 §3.3, TS-018-02-01
func checkPlatformSyncReady(artifactPath string) contracts.VerificationOutcome {
	hasIOS, err := artifactContainsDir(artifactPath, "ios")
	if err != nil {
		return failOutcome(CheckPlatformSyncReady,
			fmt.Sprintf("cannot inspect artifact %q: %v", artifactPath, err))
	}
	if !hasIOS {
		return passOutcome(CheckPlatformSyncReady,
			"no ios/ directory in the release: platform_sync is an informational no-op (no platform state to sync) — nothing to verify")
	}

	podfileFound, err := artifactContains(artifactPath, "ios/Podfile")
	if err != nil {
		return failOutcome(CheckPlatformSyncReady,
			fmt.Sprintf("cannot inspect ios/Podfile: %v", err))
	}
	if !podfileFound {
		return failOutcome(CheckPlatformSyncReady,
			"ios/ directory present but ios/Podfile missing: the platform_sync phase (pod install) has no platform integration input to finalize at its declared point")
	}
	return passOutcome(CheckPlatformSyncReady,
		"platform step evidence present: ios/ directory with ios/Podfile — platform_sync can run at its declared lifecycle point")
}

// checkRollbackBehavior implements CheckRollbackBehavior: rollback
// produces the declared state. Every activation phase must declare its
// rollback coverage — a reversible phase carries a rollback command, an
// irreversible phase is marked irreversible with no command (the adapter
// reports an informational success that never blocks rollback, TS-P7-10
// AC-2); and the manifest rollback metadata must match the executable
// phase table (only the reversible pub_get phase's re-resolution).
//
// Reference: TS-018-03-02, Review 19 §3.3, TS-018-02-01
func checkRollbackBehavior(artifactPath string) contracts.VerificationOutcome {
	if _, err := os.Stat(artifactPath); err != nil {
		return failOutcome(CheckRollbackBehavior,
			fmt.Sprintf("cannot inspect artifact %q: %v", artifactPath, err))
	}

	for _, p := range activationPhases {
		if p.irreversible {
			if len(p.rollbackArgs) > 0 {
				return failOutcome(CheckRollbackBehavior, fmt.Sprintf(
					"phase %q is marked irreversible but declares a rollback command (%s): rollback semantics are incoherent",
					p.name, strings.Join(p.rollbackArgs, " ")))
			}
			continue
		}
		if len(p.rollbackArgs) == 0 {
			return failOutcome(CheckRollbackBehavior, fmt.Sprintf(
				"phase %q declares no rollback command and is not marked irreversible: rollback coverage is missing",
				p.name))
		}
	}

	manifestRollback := ManifestCommands().RollbackCommands
	wantRollback := []string{commandString("flutter", []string{"pub", "get"})}
	if !slices.Equal(manifestRollback, wantRollback) {
		return failOutcome(CheckRollbackBehavior, fmt.Sprintf(
			"manifest rollback metadata (%s) drifts from the executable phase table (%s): only the reversible phase's re-resolution may appear in rollback — the surfaces must not diverge",
			strings.Join(manifestRollback, "; "), strings.Join(wantRollback, "; ")))
	}

	var coverage []string
	for _, p := range activationPhases {
		if p.irreversible {
			coverage = append(coverage, fmt.Sprintf("%s: informational (irreversible, rollback never blocks)", p.name))
		} else {
			coverage = append(coverage, fmt.Sprintf("%s: %s", p.name, strings.Join(p.rollbackArgs, " ")))
		}
	}
	return passOutcome(CheckRollbackBehavior, fmt.Sprintf(
		"rollback produces the declared state: %s; manifest rollback metadata matches the phase table",
		strings.Join(coverage, "; ")))
}

// artifactReadFile returns the content of relPath inside the artifact
// (directory or tar.gz archive). The bool reports whether the path
// exists; an unreadable artifact is an error.
func artifactReadFile(artifactPath, relPath string) ([]byte, bool, error) {
	info, err := os.Stat(artifactPath)
	if err != nil {
		return nil, false, err
	}
	if info.IsDir() {
		data, err := os.ReadFile(filepath.Join(artifactPath, relPath))
		if err != nil {
			if os.IsNotExist(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		return data, true, nil
	}
	return readArchiveEntry(artifactPath, relPath)
}

// readArchiveEntry scans a tar.gz archive for relPath (accepting the
// optional "app/" deployable-content prefix) and returns the content of
// the first matching regular-file entry.
func readArchiveEntry(archivePath, relPath string) ([]byte, bool, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, false, fmt.Errorf("not a gzip archive: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, false, err
		}
		name := strings.TrimPrefix(hdr.Name, "app/")
		if name == relPath && hdr.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, false, err
			}
			return data, true, nil
		}
	}
	return nil, false, nil
}
