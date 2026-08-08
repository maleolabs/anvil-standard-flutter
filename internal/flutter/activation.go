// Activation phase operations of the Flutter adapter (TS-018-02-01).
//
// The Flutter standard implements the hybrid deployment model (ADR-016):
// releases are built on a build host and deployed to targets (web bundle,
// APK, iOS app). Release activation runs the declared activation phases
// in declaration order from the release's working directory (the release
// context working_dir passed by the runtime), reflecting that model —
// the build artifact's dependency set and the platform steps of the
// native targets — rather than server-side steps (there is no server to
// activate on; EPIC-007 §7.3).
//
// The rollback operation (PhaseOperationRollback) reverses reversible
// phases (pub_get → idempotent re-resolution) and treats irreversible
// phases — platform_sync, whose effects cannot be undone — as
// informational successes that do not block rollback (007 §5: a phase
// may be irreversible; irreversibility never blocks rollback).
//
// The phase table is the single source of truth for the declared
// activation phase set: the capability declaration (Capabilities) and the
// manifest command strings (ManifestCommands) derive from it, so the
// three cannot drift.
package flutter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"maleolabs.com/anvil-standard-flutter/internal/contracts"
)

// Activation phase names declared in the capability declaration
// (Capabilities). Phase names are part of the adapter's contract surface:
// the Core invokes only declared phases (TS-P7-08 AC-3).
//
// Reference: TS-018-02-01, TS-P7-07
const (
	// PhasePubGet resolves the release's locked dependency set
	// (`flutter pub get`). The artifact was built from this dependency
	// set on the build host; activation re-resolves it in the release
	// working directory so the release serves exactly what its
	// pubspec.lock declares. Timing relative to promotion: pub_get runs
	// BEFORE the artifact is promoted — the release's dependency set is
	// the hybrid analog of Laravel's migration timing (Review 19 §3.3);
	// promoting an artifact with an unresolvable dependency set breaks
	// the release. Reversible: `flutter pub get` is idempotent and
	// lockfile-driven, so the rollback operation re-runs it in the
	// restored release's working directory to re-establish that
	// release's locked state.
	PhasePubGet = "pub_get"

	// PhasePlatformSync runs the platform steps for the release's
	// native targets. In the hybrid model the platform steps finalize
	// the platform integration of the built artifact: CocoaPods
	// (`pod install`) for the iOS target — the Podfile.lock must match
	// the release's dependency set before the iOS artifact serves.
	// Platform-aware execution mirrors the build side (ADR-018): the
	// phase applies only when the release working directory contains an
	// `ios/` directory (no CocoaPods state otherwise) and runs on macOS
	// hosts only (CocoaPods is a macOS tool; iOS finalization happens on
	// the darwin build host); otherwise it reports an informational
	// no-op. Irreversible: the platform integration cannot be undone by
	// a rollback — the previous release's own activation re-runs its
	// platform sync; a rollback request returns an informational success
	// that does not block rollback.
	PhasePlatformSync = "platform_sync"
)

// activationPhase defines one activation phase: the program and arguments
// it runs during activation, its rollback arguments when reversible, and
// whether the phase is irreversible (no rollback operation — the phase
// reports an informational result instead of blocking rollback).
//
// Reference: TS-018-02-01, TS-P7-10
type activationPhase struct {
	// name is the phase identifier (Phase* constants).
	name string

	// program is the executable that runs the phase: "flutter" or
	// "pod". The production runner resolves it via os/exec; tests
	// inject fakes per program.
	program string

	// activateArgs are the arguments for the activate operation
	// (e.g. {"pub", "get"}).
	activateArgs []string

	// rollbackArgs are the arguments for the rollback operation.
	// Nil when the phase has no rollback operation.
	rollbackArgs []string

	// irreversible reports that the phase's effects cannot be undone.
	// Rollback requests then return an informational success instead of
	// running a command (TS-P7-10 AC-2).
	irreversible bool

	// requiresDir names the release-directory entry the phase needs
	// (e.g. "ios"). When set and the release working directory does not
	// contain the entry, the phase reports an informational no-op —
	// there is no platform state for it to sync.
	requiresDir string

	// requiresPlatform is the host platform the phase runs on (Platform*
	// constants, ADR-018). When set and the host platform differs, the
	// phase reports an informational skip — the step belongs to that
	// platform's build host (platform-aware execution, ADR-018).
	requiresPlatform string
}

// activationPhases is the adapter's activation phase table, in capability
// declaration order: pub_get first (dependency resolution before
// promotion), then platform_sync (platform steps on the resolved
// dependency set). The order is the fixed order the capability
// declaration's ActivationPhases and the manifest command strings use.
//
// Reference: TS-018-02-01, TS-P7-20
var activationPhases = []activationPhase{
	{
		name:         PhasePubGet,
		program:      "flutter",
		activateArgs: []string{"pub", "get"},
		rollbackArgs: []string{"pub", "get"},
	},
	{
		name:             PhasePlatformSync,
		program:          "pod",
		activateArgs:     []string{"install"},
		irreversible:     true,
		requiresDir:      "ios",
		requiresPlatform: PlatformDarwin,
	},
}

// activationPhaseNames returns the activation phase names in table order —
// the ActivationPhases declaration mirrors the phase table exactly, so
// the two cannot drift.
//
// Reference: TS-018-02-01, TS-P7-20 AC-4
func activationPhaseNames() []string {
	names := make([]string, 0, len(activationPhases))
	for _, p := range activationPhases {
		names = append(names, p.name)
	}
	return names
}

// runProgram executes `program <args...>` in dir (empty dir = current
// working directory) and returns the command output. It is the shared
// production runner for the activation phase programs (flutter, pod) and
// the build pipeline's runFlutter, which delegates here.
//
// On failure the error carries the program's stderr (or the exit error
// when stderr is empty) so phase failures report actionable details
// (TS-P7-09 AC-6, TS-P7-21).
//
// Reference: TS-018-02-01, TS-P7-21, 004-review-resolutions D1
func runProgram(ctx context.Context, program, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, program, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return stdout.String(), fmt.Errorf("%s %s failed: %s", program, strings.Join(args, " "), detail)
	}
	return stdout.String(), nil
}

// runPod executes `pod <args...>` via os/exec — the CocoaPods command
// line tool used by the platform_sync phase. CocoaPods is a macOS tool,
// which the phase table's requiresPlatform (PlatformDarwin) enforces
// before this runner is reached.
//
// Reference: TS-018-02-01
func runPod(ctx context.Context, dir string, args ...string) (string, error) {
	return runProgram(ctx, "pod", dir, args...)
}

// RunActivation executes one activation phase operation and returns the
// contract result. The JSON result is authoritative for the phase
// outcome: Success=false with Error details reports a phase failure,
// while the process-level exit code reports whether the adapter itself
// produced a result (005-adapter-command-contract §7).
//
// The runners are injectable so tests can exercise the phase operations
// without the Flutter toolchain or CocoaPods on the host: flutterRunner
// executes the flutter program phases (pub_get), podRunner executes the
// pod program phases (platform_sync); nil runners default to the
// production runners (runFlutter, runPod).
//
// Platform-aware execution (ADR-018): a phase whose requiresDir is not
// present in the release working directory, or whose requiresPlatform
// does not match the host, reports an informational no-op with
// Success=true — the phase has no state to act on, which is not a
// failure and does not stop activation.
//
// Reference: TS-018-02-01, TS-P7-09 AC-1..AC-6, TS-P7-10 AC-1, AC-2,
// ADR-018
func RunActivation(ctx context.Context, flutterRunner, podRunner commandRunner, req contracts.ActivationRequest) contracts.ActivationResult {
	p, ok := lookupActivationPhase(req.Phase)
	if !ok {
		return contracts.ActivationResult{
			Success: false,
			Error:   fmt.Sprintf("unknown activation phase %q", req.Phase),
		}
	}

	switch req.Operation {
	case contracts.PhaseOperationActivate:
		return runActivationPhase(ctx, flutterRunner, podRunner, req, p, p.activateArgs)
	case contracts.PhaseOperationRollback:
		if p.irreversible {
			return irreversibleActivationRollbackResult(p)
		}
		return runActivationPhase(ctx, flutterRunner, podRunner, req, p, p.rollbackArgs)
	default:
		return contracts.ActivationResult{
			Success: false,
			Error:   fmt.Sprintf("unknown activation operation %q", req.Operation),
		}
	}
}

// lookupActivationPhase returns the phase definition for the given name.
func lookupActivationPhase(name string) (activationPhase, bool) {
	for _, p := range activationPhases {
		if p.name == name {
			return p, true
		}
	}
	return activationPhase{}, false
}

// activationRunnerFor returns the injectable runner for the phase's
// program: the flutter runner for flutter program phases (pub_get), the
// pod runner for pod program phases (platform_sync). A nil runner falls
// back to the production runner for the program.
func activationRunnerFor(flutterRunner, podRunner commandRunner, p activationPhase) commandRunner {
	if p.program == "pod" {
		if podRunner != nil {
			return podRunner
		}
		return runPod
	}
	if flutterRunner != nil {
		return flutterRunner
	}
	return runFlutter
}

// runActivationPhase applies the phase's conditional applicability
// (requiresDir, requiresPlatform — ADR-018 platform-aware execution),
// then executes the phase arguments via the runner and maps the outcome
// to an ActivationResult. The working directory from the release context
// is passed through so the phase runs inside the release directory
// (005-adapter-command-contract §3.3).
func runActivationPhase(ctx context.Context, flutterRunner, podRunner commandRunner, req contracts.ActivationRequest, p activationPhase, args []string) contracts.ActivationResult {
	dir := req.Release.WorkingDir

	if p.requiresDir != "" && !dirHasEntry(dir, p.requiresDir) {
		return contracts.ActivationResult{
			Success: true,
			Output: fmt.Sprintf(
				"phase %q skipped: no %q directory in the release working directory — no platform state to sync",
				p.name, p.requiresDir,
			),
		}
	}

	if p.requiresPlatform != "" {
		current := platformDetector()
		if current != p.requiresPlatform {
			return contracts.ActivationResult{
				Success: true,
				Output: fmt.Sprintf(
					"phase %q skipped: %s runs on %s only (current platform %q); the platform step belongs to the %s build host",
					p.name, p.program, p.requiresPlatform, current, p.requiresPlatform,
				),
			}
		}
	}

	runner := activationRunnerFor(flutterRunner, podRunner, p)
	output, err := runner(ctx, dir, args...)
	if err != nil {
		return contracts.ActivationResult{
			Success: false,
			Output:  output,
			Error:   err.Error(),
		}
	}
	return contracts.ActivationResult{
		Success: true,
		Output:  output,
	}
}

// dirHasEntry reports whether the directory contains the named
// DIRECTORY entry. An empty dir refers to the current working directory.
// The check is structural — the same approach as the standard's
// verification checks (pubspec.yaml, lib/ presence) — so the phase's
// applicability follows from the release contents, not from contract
// fields (the release context is generic and carries no target
// information). A file with the same name does not satisfy the check:
// the declared semantics is "contains an ios/ directory", not "contains
// an entry named ios".
func dirHasEntry(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && info.IsDir()
}

// irreversibleActivationRollbackResult reports an informational success
// for a rollback request on an irreversible phase. The operation cannot
// be undone, so the adapter documents the limitation in the result and
// does NOT block the rollback (007 §5 — irreversibility never blocks
// rollback; TS-P7-10 AC-2): the Core treats Success=true as "rollback of
// this phase completed without error". The previous release's own
// activation re-runs the phase for its state.
func irreversibleActivationRollbackResult(p activationPhase) contracts.ActivationResult {
	return contracts.ActivationResult{
		Success: true,
		Output: fmt.Sprintf(
			"phase %q is irreversible: %s %s cannot be undone; rollback proceeds without undoing this operation — the previous release's own activation re-runs its platform steps",
			p.name, p.program, strings.Join(p.activateArgs, " "),
		),
	}
}

// ManifestCommands returns the activation and rollback command strings
// stored in the artifact manifest at packaging time (ADR-017) and
// executed by the orchestrator during release activation and rollback.
// The strings derive from the activation phase table in table order —
// activation carries every phase's command (join(program, activateArgs)),
// rollback carries only the phases with a rollback operation
// (rollbackArgs != nil, i.e. the non-irreversible phases) — so the
// manifest surface and the executable phase table cannot drift.
//
// The platform_sync entry is conditional by nature — it applies when the
// release working directory contains an ios/ directory and the host is
// macOS (ADR-018 platform-aware execution); the metadata form records
// the command, the executable phase table carries the conditions.
//
// Reference: TS-018-02-01, TS-P7-15, TS-P7-16, ADR-017
func ManifestCommands() contracts.ManifestCommandResult {
	activation := make([]string, 0, len(activationPhases))
	rollback := make([]string, 0, len(activationPhases))
	for _, p := range activationPhases {
		activation = append(activation, commandString(p.program, p.activateArgs))
		if p.rollbackArgs != nil {
			rollback = append(rollback, commandString(p.program, p.rollbackArgs))
		}
	}
	return contracts.ManifestCommandResult{
		ActivationCommands: activation,
		RollbackCommands:   rollback,
	}
}

// commandString renders one phase operation as its full command string —
// the program followed by the space-joined arguments (e.g. "flutter pub
// get"). It is the string form the artifact manifest stores (ADR-017).
func commandString(program string, args []string) string {
	return strings.Join(append([]string{program}, args...), " ")
}
