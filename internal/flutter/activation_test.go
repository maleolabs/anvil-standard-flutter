// Tests for the Flutter adapter's hybrid-model activation phases
// (TS-018-02-01): the declared phase table and its order, per-phase
// failure semantics, per-phase rollback semantics, and the informational
// handling of irreversible phases — rollback is never blocked by an
// irreversible phase.
package flutter

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"maleolabs.com/anvil-standard-flutter/internal/contracts"
)

// activationRequest builds an ActivationRequest for the phase, operation,
// and working directory under test.
func activationRequest(phase string, operation contracts.PhaseOperation, workingDir string) contracts.ActivationRequest {
	return contracts.ActivationRequest{
		Phase:     phase,
		Operation: operation,
		Release: contracts.ReleaseContext{
			ProjectID:  "project-1",
			ReleaseID:  "release-1",
			WorkingDir: workingDir,
		},
	}
}

// runActivation invokes RunActivation with the given fake runners and
// request, returning the result. The fakes record their invocations for
// inspection.
func runActivation(t *testing.T, flutterRunner, podRunner *fakeRunner, req contracts.ActivationRequest) contracts.ActivationResult {
	t.Helper()
	return RunActivation(context.Background(), flutterRunner.run, podRunner.run, req)
}

// fakes returns two fresh fake runners (flutter + pod) for a request.
func fakes() (*fakeRunner, *fakeRunner) {
	return &fakeRunner{output: "ok"}, &fakeRunner{output: "ok"}
}

// TestActivationPhases_DeclaredOrder verifies the declared activation
// phase table is in declared order — pub_get (dependency resolution)
// before platform_sync (platform steps) — and that the capability
// declaration mirrors the table exactly so the two cannot drift
// (TS-018-02-01, TS-P7-20 AC-4).
func TestActivationPhases_DeclaredOrder(t *testing.T) {
	want := []string{PhasePubGet, PhasePlatformSync}
	if !reflect.DeepEqual(activationPhaseNames(), want) {
		t.Errorf("activationPhaseNames() = %v, want %v", activationPhaseNames(), want)
	}

	result := Capabilities()
	if !reflect.DeepEqual(result.Declaration.ActivationPhases, want) {
		t.Errorf("capability ActivationPhases = %v, want %v (must mirror the phase table)", result.Declaration.ActivationPhases, want)
	}
}

// TestActivationPhases_TableContent verifies the declarative phase table
// content: each phase's program, activation arguments, rollback
// arguments, and irreversibility — the per-phase rollback semantics of
// the hybrid model (TS-018-02-01).
func TestActivationPhases_TableContent(t *testing.T) {
	if len(activationPhases) != 2 {
		t.Fatalf("activationPhases length = %d, want 2 (pub_get, platform_sync)", len(activationPhases))
	}

	pubGet := activationPhases[0]
	if pubGet.name != PhasePubGet || pubGet.program != "flutter" {
		t.Errorf("phases[0] = %+v, want pub_get running the flutter program", pubGet)
	}
	if !reflect.DeepEqual(pubGet.activateArgs, []string{"pub", "get"}) {
		t.Errorf("pub_get activateArgs = %v, want [pub get]", pubGet.activateArgs)
	}
	if pubGet.irreversible {
		t.Errorf("pub_get irreversible = true, want false — dependency resolution is reversible (idempotent re-resolution)")
	}
	if !reflect.DeepEqual(pubGet.rollbackArgs, []string{"pub", "get"}) {
		t.Errorf("pub_get rollbackArgs = %v, want [pub get]", pubGet.rollbackArgs)
	}

	platformSync := activationPhases[1]
	if platformSync.name != PhasePlatformSync || platformSync.program != "pod" {
		t.Errorf("phases[1] = %+v, want platform_sync running the pod program", platformSync)
	}
	if !reflect.DeepEqual(platformSync.activateArgs, []string{"install"}) {
		t.Errorf("platform_sync activateArgs = %v, want [install]", platformSync.activateArgs)
	}
	if !platformSync.irreversible {
		t.Errorf("platform_sync irreversible = false, want true — the platform integration cannot be undone by rollback")
	}
	if platformSync.rollbackArgs != nil {
		t.Errorf("platform_sync rollbackArgs = %v, want nil (irreversible)", platformSync.rollbackArgs)
	}
	if platformSync.requiresDir != "ios" {
		t.Errorf("platform_sync requiresDir = %q, want %q", platformSync.requiresDir, "ios")
	}
	if platformSync.requiresPlatform != PlatformDarwin {
		t.Errorf("platform_sync requiresPlatform = %q, want %q", platformSync.requiresPlatform, PlatformDarwin)
	}
}

// TestRunActivation_PubGetActivate verifies the pub_get activate
// operation runs `flutter pub get` in the release working directory and
// reports a successful result — dependency resolution before promotion
// (TS-018-02-01).
func TestRunActivation_PubGetActivate(t *testing.T) {
	f, p := fakes()
	req := activationRequest(PhasePubGet, contracts.PhaseOperationActivate, "/var/lib/anvil/projects/acme-app/releases/rel-2")

	result := runActivation(t, f, p, req)

	if !result.Success {
		t.Fatalf("Success = false, want true (result: %#v)", result)
	}
	if result.Error != "" {
		t.Errorf("Error = %q, want empty", result.Error)
	}
	if len(f.args) != 1 {
		t.Fatalf("runner invocations = %d, want 1", len(f.args))
	}
	if !reflect.DeepEqual(f.args[0], []string{"pub", "get"}) {
		t.Errorf("runner args = %v, want [pub get]", f.args[0])
	}
	if f.dirs[0] != req.Release.WorkingDir {
		t.Errorf("runner dir = %q, want the release working directory", f.dirs[0])
	}
	if len(p.args) != 0 {
		t.Errorf("pod runner invocations = %d, want 0 — pub_get runs the flutter program only", len(p.args))
	}
}

// TestRunActivation_PubGetActivateFailure verifies the pub_get failure
// semantics: a failed dependency resolution fails activation with the
// runner's error details — a release with an unresolvable dependency set
// cannot be activated (TS-018-02-01).
func TestRunActivation_PubGetActivateFailure(t *testing.T) {
	f, p := fakes()
	f.output = "stderr detail"
	f.err = errFake("flutter pub get failed: network unreachable")

	result := runActivation(t, f, p, activationRequest(PhasePubGet, contracts.PhaseOperationActivate, ""))

	if result.Success {
		t.Fatalf("Success = true, want false — an unresolvable dependency set fails activation")
	}
	if !strings.Contains(result.Error, "network unreachable") {
		t.Errorf("Error = %q, want the runner failure details", result.Error)
	}
}

// TestRunActivation_PubGetRollback verifies the pub_get rollback
// operation re-runs `flutter pub get` in the restored release's working
// directory — the phase is reversible via idempotent, lockfile-driven
// re-resolution (TS-018-02-01).
func TestRunActivation_PubGetRollback(t *testing.T) {
	f, p := fakes()

	result := runActivation(t, f, p, activationRequest(PhasePubGet, contracts.PhaseOperationRollback, "/var/lib/anvil/projects/acme-app/releases/rel-1"))

	if !result.Success {
		t.Fatalf("Success = false, want true (result: %#v)", result)
	}
	if len(f.args) != 1 {
		t.Fatalf("runner invocations = %d, want 1", len(f.args))
	}
	if !reflect.DeepEqual(f.args[0], []string{"pub", "get"}) {
		t.Errorf("rollback args = %v, want [pub get]", f.args[0])
	}
}

// TestRunActivation_PubGetRollbackFailure verifies a failing rollback
// operation reports the failure — the rollback operation is a real
// operation with real failure semantics; it is not silently swallowed.
func TestRunActivation_PubGetRollbackFailure(t *testing.T) {
	f, p := fakes()
	f.err = errFake("flutter pub get failed: disk full")

	result := runActivation(t, f, p, activationRequest(PhasePubGet, contracts.PhaseOperationRollback, ""))

	if result.Success {
		t.Fatalf("Success = true, want false — a failed rollback operation reports the failure")
	}
	if !strings.Contains(result.Error, "disk full") {
		t.Errorf("Error = %q, want the rollback failure details", result.Error)
	}
}

// TestRunActivation_PlatformSyncActivate verifies the platform_sync
// activate operation runs `pod install` in the release working directory
// when the release contains an ios/ directory and the host is macOS —
// the platform steps of the hybrid model's native targets
// (TS-018-02-01, ADR-018 platform-aware execution).
func TestRunActivation_PlatformSyncActivate(t *testing.T) {
	dir := releaseDirWithIOS(t)
	withPlatform(t, PlatformDarwin)
	f, p := fakes()

	result := runActivation(t, f, p, activationRequest(PhasePlatformSync, contracts.PhaseOperationActivate, dir))

	if !result.Success {
		t.Fatalf("Success = false, want true (result: %#v)", result)
	}
	if len(p.args) != 1 {
		t.Fatalf("pod runner invocations = %d, want 1", len(p.args))
	}
	if !reflect.DeepEqual(p.args[0], []string{"install"}) {
		t.Errorf("pod args = %v, want [install]", p.args[0])
	}
	if p.dirs[0] != dir {
		t.Errorf("pod dir = %q, want the release working directory", p.dirs[0])
	}
	if len(f.args) != 0 {
		t.Errorf("flutter runner invocations = %d, want 0 — platform_sync runs the pod program only", len(f.args))
	}
}

// TestRunActivation_PlatformSyncActivateFailure verifies the
// platform_sync failure semantics: a failed platform step fails
// activation with the runner's error details — the iOS platform
// integration is broken and the artifact cannot serve its native target.
func TestRunActivation_PlatformSyncActivateFailure(t *testing.T) {
	dir := releaseDirWithIOS(t)
	withPlatform(t, PlatformDarwin)
	f, p := fakes()
	p.output = "stderr detail"
	p.err = errFake("pod install failed: Podfile.lock conflict")

	result := runActivation(t, f, p, activationRequest(PhasePlatformSync, contracts.PhaseOperationActivate, dir))

	if result.Success {
		t.Fatalf("Success = true, want false — a broken platform step fails activation")
	}
	if !strings.Contains(result.Error, "Podfile.lock conflict") {
		t.Errorf("Error = %q, want the platform step failure details", result.Error)
	}
}

// TestRunActivation_PlatformSyncNoIOSDir verifies the platform_sync
// conditional applicability: without an ios/ directory in the release
// working directory there is no CocoaPods platform state to sync — the
// phase reports an informational no-op with Success=true and runs no
// command (TS-018-02-01).
func TestRunActivation_PlatformSyncNoIOSDir(t *testing.T) {
	withPlatform(t, PlatformDarwin)
	f, p := fakes()

	result := runActivation(t, f, p, activationRequest(PhasePlatformSync, contracts.PhaseOperationActivate, t.TempDir()))

	if !result.Success {
		t.Fatalf("Success = false, want true — no platform state is not a failure")
	}
	if len(p.args) != 0 {
		t.Errorf("pod runner invocations = %d, want 0 — no command runs without an ios/ directory", len(p.args))
	}
	if !strings.Contains(result.Output, "skipped") {
		t.Errorf("Output = %q, want an informational skip message", result.Output)
	}
}

// TestRunActivation_PlatformSyncNonDarwin verifies the platform_sync
// platform-aware execution: CocoaPods is a macOS tool, so on a non-darwin
// host the phase reports an informational skip — the iOS platform step
// belongs to the darwin build host (TS-018-02-01, ADR-018).
func TestRunActivation_PlatformSyncNonDarwin(t *testing.T) {
	dir := releaseDirWithIOS(t)
	withPlatform(t, PlatformLinux)
	f, p := fakes()

	result := runActivation(t, f, p, activationRequest(PhasePlatformSync, contracts.PhaseOperationActivate, dir))

	if !result.Success {
		t.Fatalf("Success = false, want true — an unsupported host platform is a skip, not a failure")
	}
	if len(p.args) != 0 {
		t.Errorf("pod runner invocations = %d, want 0 — no command runs on a non-darwin host", len(p.args))
	}
	if !strings.Contains(result.Output, "skipped") {
		t.Errorf("Output = %q, want an informational skip message", result.Output)
	}
}

// TestRunActivation_PlatformSyncRollbackIrreversible verifies the
// irreversible phase rollback semantics: a rollback request on
// platform_sync returns an informational success, runs NO command, and
// never blocks the rollback (TS-018-02-01, 007 §5 — irreversibility
// never blocks rollback).
func TestRunActivation_PlatformSyncRollbackIrreversible(t *testing.T) {
	dir := releaseDirWithIOS(t)
	withPlatform(t, PlatformDarwin)
	f, p := fakes()

	result := runActivation(t, f, p, activationRequest(PhasePlatformSync, contracts.PhaseOperationRollback, dir))

	if !result.Success {
		t.Fatalf("Success = false, want true — an irreversible phase never blocks rollback")
	}
	if len(p.args) != 0 {
		t.Errorf("pod runner invocations = %d, want 0 — no rollback command exists for an irreversible phase", len(p.args))
	}
	if len(f.args) != 0 {
		t.Errorf("flutter runner invocations = %d, want 0 — no command runs for an irreversible rollback", len(f.args))
	}
	if !strings.Contains(result.Output, "irreversible") {
		t.Errorf("Output = %q, want the informational irreversibility message", result.Output)
	}
}

// TestRunActivation_UnknownPhase verifies an undeclared phase name is
// rejected: the Core invokes only declared phases, and an unknown phase
// reports Success=false with an error describing the name
// (TS-P7-08 AC-3).
func TestRunActivation_UnknownPhase(t *testing.T) {
	f, p := fakes()

	result := runActivation(t, f, p, activationRequest("frobnicate", contracts.PhaseOperationActivate, ""))

	if result.Success {
		t.Fatalf("Success = true, want false for an unknown phase")
	}
	if !strings.Contains(result.Error, `"frobnicate"`) {
		t.Errorf("Error = %q, want the unknown phase name", result.Error)
	}
}

// TestRunActivation_UnknownOperation verifies an undeclared operation is
// rejected with Success=false — the contract supports activate and
// rollback only.
func TestRunActivation_UnknownOperation(t *testing.T) {
	f, p := fakes()

	result := runActivation(t, f, p, activationRequest(PhasePubGet, "explode", ""))

	if result.Success {
		t.Fatalf("Success = true, want false for an unknown operation")
	}
}

// TestManifestCommands verifies the manifest command strings derive from
// the declared phase table: activation runs `flutter pub get` then
// `pod install` (the platform step); rollback carries only the
// reversible phase's operation (pub_get) — irreversible phases reverse
// nothing (TS-018-02-01, ADR-017).
func TestManifestCommands(t *testing.T) {
	result := ManifestCommands()

	wantActivation := []string{"flutter pub get", "pod install"}
	if !reflect.DeepEqual(result.ActivationCommands, wantActivation) {
		t.Errorf("ActivationCommands = %v, want %v", result.ActivationCommands, wantActivation)
	}
	wantRollback := []string{"flutter pub get"}
	if !reflect.DeepEqual(result.RollbackCommands, wantRollback) {
		t.Errorf("RollbackCommands = %v, want %v", result.RollbackCommands, wantRollback)
	}
}

// releaseDirWithIOS returns a temporary directory containing an ios/
// entry, simulating a Flutter release working directory with native iOS
// platform state.
func releaseDirWithIOS(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "ios"), 0o755); err != nil {
		t.Fatalf("create ios/ directory: %v", err)
	}
	return dir
}

// errFake is a plain error value for runner fakes.
type errFake string

func (e errFake) Error() string { return string(e) }
