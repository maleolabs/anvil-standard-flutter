// Command dispatcher of the Flutter adapter executable.
//
// The dispatcher implements the invocation shape of the adapter command
// contract (005-adapter-command-contract §5.2): the first CLI argument is
// the command name, the second is the JSON payload as a single argument.
// It prints the structured JSON result to stdout and uses the exit code
// convention of ADR-010 §8.1.
//
// Exit code semantics (documented in 005-adapter-command-contract §7):
// the adapter exits 0 whenever it produced a valid JSON result on stdout
// — the JSON result is authoritative for the operation outcome (a target
// that fails reports Success=false in its JSON result, which the Core
// reads as the phase outcome). The adapter exits non-zero only when it
// could NOT produce a JSON result: unknown command, malformed payload,
// or an internal dispatch error. This keeps the exit code and the JSON
// result in agreement (005 §7 — "the exit code is authoritative for
// process-level failure").
//
// The dispatcher implements the commands of this batch plus the
// registration scaffold — `capabilities`, `extension`, `verify`,
// `validate`, `build`, `activate`, `template`, and `manifest`. The
// `verify` command runs the Flutter verification checks (TS-P7-25) and
// the `validate` command validates Flutter config extension values
// (TS-P7-26). The `activate` command runs the hybrid deployment model's
// activation phase operations (TS-018-02-01): the Core invokes it with
// an ActivationRequest payload and receives the ActivationResult for one
// declared phase operation (pub_get, platform_sync — activate or
// rollback). The `template` command returns the adapter-owned pipeline
// definitions (TS-007-038, ADR-020 §1). The `manifest` command returns
// the activation and rollback command strings the hybrid model stores in
// the artifact manifest (TS-018-02-01, TS-P7-15, TS-P7-16).
package flutter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"maleolabs.com/anvil-standard-flutter/internal/contracts"
)

// Exit codes of the adapter executable. Zero indicates success
// (ADR-010 §8.1); non-zero values categorize the failure.
//
// Reference: 005-adapter-command-contract §7
const (
	// ExitOK is returned when the adapter produced a valid JSON result.
	ExitOK = 0

	// ExitError is returned when a command could not be dispatched:
	// malformed JSON payload, invalid arguments, or an internal error.
	ExitError = 1

	// ExitUsage is returned for an unknown command or a missing command
	// name.
	ExitUsage = 2
)

// Adapter is the Flutter adapter executable's command surface. It holds
// the injectable command runners used by the build and activation
// pipelines.
//
// Reference: TS-P7-20, TS-P7-21, TS-018-02-01, 004-review-resolutions D1
type Adapter struct {
	// buildRunner executes the build targets (`flutter build ...`). A
	// nil buildRunner means each target uses its production runner from
	// the build table (runFlutter); tests set it to a fake to execute
	// the build pipeline without the Flutter toolchain on the host
	// (TS-P7-21).
	buildRunner commandRunner

	// activationRunner executes the flutter program activation phases
	// (`flutter pub get` — pub_get). A nil activationRunner means the
	// phase uses its production runner (runFlutter); tests set it to a
	// fake to execute the activation pipeline without the Flutter
	// toolchain on the host (TS-018-02-01).
	activationRunner commandRunner

	// podRunner executes the pod program activation phases
	// (`pod install` — platform_sync). A nil podRunner means the phase
	// uses its production runner (runPod); tests set it to a fake to
	// execute the platform step without CocoaPods on the host
	// (TS-018-02-01).
	podRunner commandRunner
}

// New returns an Adapter with the production runners left nil, so build
// targets use runFlutter and activation phases use their production
// runners (runFlutter, runPod). Tests construct &Adapter{buildRunner: f,
// activationRunner: f, podRunner: f} to execute the pipelines without
// the Flutter toolchain or CocoaPods on the host.
func New() *Adapter {
	return &Adapter{}
}

// ErrUnknownCommand is returned when the adapter receives a command name
// it does not implement. It maps to ExitUsage so unknown commands are
// distinguishable from malformed payloads.
var ErrUnknownCommand = errors.New("unknown command")

// Run handles one adapter invocation and returns the process exit code.
// args[0] is the command name; args[1] is the JSON payload (the Process
// Runner has no stdin channel, so payloads are always passed as a single
// trailing argument). The JSON result is written to stdout; diagnostics
// are written to stderr.
//
// Reference: 005-adapter-command-contract §2, §5.2, §7
func (a *Adapter) Run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "flutter-adapter: usage: flutter-adapter <command> [<json-payload>]")
		return ExitUsage
	}
	if len(args) > 2 {
		fmt.Fprintf(stderr, "flutter-adapter: too many arguments for command %q (expected <command> [<json-payload>])\n", args[0])
		return ExitUsage
	}

	command := args[0]
	var payload []byte
	if len(args) == 2 {
		payload = []byte(args[1])
	}

	result, err := a.handle(context.Background(), command, payload)
	if err != nil {
		fmt.Fprintf(stderr, "flutter-adapter: %v\n", err)
		if errors.Is(err, ErrUnknownCommand) {
			return ExitUsage
		}
		return ExitError
	}

	data, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(stderr, "flutter-adapter: marshal result for command %q: %v\n", command, err)
		return ExitError
	}
	fmt.Fprintln(stdout, string(data))
	return ExitOK
}

// handle dispatches one command to its handler and returns the contract
// payload to serialize. An error means no JSON result can be produced —
// the caller maps it to a non-zero exit.
func (a *Adapter) handle(ctx context.Context, command string, payload []byte) (any, error) {
	switch command {
	case contracts.CommandCapabilities:
		var req contracts.CapabilityRequest
		if err := parsePayload(command, payload, &req); err != nil {
			return nil, err
		}
		return Capabilities(), nil

	case contracts.CommandConfigExtension:
		var req contracts.ConfigExtensionRequest
		if err := parsePayload(command, payload, &req); err != nil {
			return nil, err
		}
		return ConfigExtension(), nil

	case contracts.CommandVerification:
		var req contracts.VerificationRequest
		if err := parsePayload(command, payload, &req); err != nil {
			return nil, err
		}
		return RunVerification(req), nil

	case contracts.CommandConfigValidation:
		var req contracts.ConfigValidationRequest
		if err := parsePayload(command, payload, &req); err != nil {
			return nil, err
		}
		return ValidateConfigValues(req), nil

	case contracts.CommandBuild:
		var req contracts.BuildRequest
		if err := parsePayload(command, payload, &req); err != nil {
			return nil, err
		}
		return RunBuild(ctx, a.buildRunner, req), nil

	case contracts.CommandActivation:
		var req contracts.ActivationRequest
		if err := parsePayload(command, payload, &req); err != nil {
			return nil, err
		}
		return RunActivation(ctx, a.activationRunner, a.podRunner, req), nil

	case contracts.CommandTemplate:
		var req contracts.TemplateRequest
		if err := parsePayload(command, payload, &req); err != nil {
			return nil, err
		}
		return Template(), nil

	case contracts.CommandManifest:
		// The hybrid deployment model's activation and rollback command
		// strings are stored in the artifact manifest at packaging time
		// (ADR-017, TS-018-02-01). The strings derive from the declared
		// activation phase table: activation runs `flutter pub get`
		// then `pod install` (platform step, conditional on the ios/
		// directory and macOS host); rollback re-runs `flutter pub get`
		// (the only reversible phase — platform_sync is irreversible
		// and reverse nothing).
		return ManifestCommands(), nil

	default:
		return nil, fmt.Errorf("%w %q", ErrUnknownCommand, command)
	}
}

// parsePayload decodes the single JSON payload argument into req. The
// payload is required — the Core always sends it; a malformed payload is
// a contract violation reported as a process failure (non-zero exit).
func parsePayload(command string, payload []byte, req any) error {
	if len(payload) == 0 {
		return fmt.Errorf("command %q requires a JSON payload argument", command)
	}
	if err := json.Unmarshal(payload, req); err != nil {
		return fmt.Errorf("command %q: invalid JSON payload: %v", command, err)
	}
	return nil
}
