// Package platform mirrors the Anvil Core's canonical platform
// identifiers and deterministic GOOS detection (the Core's
// internal/platform package, ADR-018).
//
// The standard is an independent module and cannot import the Core's
// internal packages, so it carries these values locally. The identifiers
// and detection semantics are stable contract values (ADR-018): "linux",
// "darwin", "windows", unknown GOOS passes through unchanged. The build
// target table and the pipeline template metadata consume them so the
// runtime's platform-aware execution (skip unsupported targets with a
// warning; --target selection; --strict failure) keeps working against
// this standard unchanged.
//
// Reference: TS-P7-22, TS-P7-23, ADR-018; Core internal/platform
package platform

import "runtime"

// Canonical platform identifiers (ADR-018 detection values). The values
// are the platform names the pipeline task metadata declares in its
// platforms list and the values Detect returns.
const (
	// PlatformLinux identifies the Linux operating system (GOOS
	// "linux").
	PlatformLinux = "linux"

	// PlatformDarwin identifies macOS (GOOS "darwin"). The Flutter iOS
	// build target is the only target that requires it — iOS builds
	// need Xcode, which exists on macOS only (ADR-018).
	PlatformDarwin = "darwin"

	// PlatformWindows identifies the Windows operating system (GOOS
	// "windows").
	PlatformWindows = "windows"
)

// Detect maps a GOOS value to its canonical platform identifier
// (PlatformLinux, PlatformDarwin, or PlatformWindows). It is a pure
// function — no runtime state — so tests exercise every platform
// deterministically regardless of the host.
//
// Unknown GOOS values are passed through unchanged; the runtime treats
// anything outside the known set as unsupported for every target.
func Detect(goos string) string {
	switch goos {
	case PlatformLinux, PlatformDarwin, PlatformWindows:
		return goos
	default:
		return goos
	}
}

// Current returns the platform identifier of the host the standard runs
// on, using runtime.GOOS as the detection source.
func Current() string {
	return Detect(runtime.GOOS)
}
