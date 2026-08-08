// Command flutter-adapter is the Flutter framework adapter executable
// (004-review-resolutions D1: adapters are standalone executables invoked
// by the Core as `<adapter-executable> <command> <json-payload>`).
//
// The binary name convention is `anvil-adapter-flutter` — the Core
// resolves it via exec.LookPath("anvil-adapter-" + framework) when a
// project selects the "flutter" adapter (005-adapter-command-contract
// §10).
//
// Supported commands: capabilities, extension, verify, validate, build,
// activate, template, manifest (005-adapter-command-contract §5.2, §6.2;
// the template command returns the adapter-owned pipeline definitions,
// ADR-020 §1). The `activate` command runs the hybrid deployment model's
// activation phase operations — pub_get and platform_sync, with per-phase
// failure and rollback semantics (TS-018-02-01, ADR-016, EPIC-007 §7.3).
// JSON result on stdout; exit 0 on a produced result, non-zero on
// dispatch failure (ADR-010 §8.1).
//
// Reference: TS-P7-20, TS-P7-21, TS-P7-22, TS-P7-25, TS-P7-26,
// TS-018-02-01, TS-007-038, ADR-016, ADR-020, 004-review-resolutions D1
package main

import (
	"os"

	"maleolabs.com/anvil-standard-flutter/internal/flutter"
)

func main() {
	os.Exit(flutter.New().Run(os.Args[1:], os.Stdout, os.Stderr))
}
