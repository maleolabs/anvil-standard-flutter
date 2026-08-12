// Adapter-owned pipeline template of the Flutter adapter (TS-007-038,
// TS-018-02-02).
//
// The build pipeline definition the adapter owns is returned through the
// `template` command (contracts.CommandTemplate) and written by the Core
// to .anvil/pipelines/build.yaml at generation time, replacing the
// Core-embedded template function execution.FlutterBuildPipeline (ADR-020
// §1: framework knowledge moves OUT of the Core binary INTO the adapter
// binaries).
//
// The definition covers the Flutter hybrid build steps at the same
// convention depth as the Laravel template (Review 19 §3.3; TS-018-02-02):
// a "dependencies" stage first — `flutter pub get` resolves the package
// graph before any build can run, the same position Laravel's
// composer install holds — followed by the "build" stage with the
// targets the adapter's build pipeline executes (internal/flutter/build.go
// — the single source of build knowledge): `flutter build web`,
// `flutter build apk --release`, and `flutter build ios --release`.
//
// Each build task preserves the ADR-018 platform metadata — the platforms
// that support the target and the target name — so the local engine keeps
// its platform-aware execution (skip unsupported targets with a warning;
// --target selection) on the adapter-owned template. The explicit
// timeouts (10m web, 15m apk) are preserved from the pre-ADR-020 Core
// template: Flutter builds (first Gradle run in particular) routinely
// exceed the engine's 5-minute default (found by E2E verification,
// ST-007-005). The pub-get task carries no target metadata — it is a
// dependency step, not a build target — and no timeout: `flutter pub get`
// is fast and platform-independent.
//
// The CI definition mirrors the generic CI scaffold the Core used to own
// (build + test placeholder stages): the CI pipeline is generic
// placeholder data, not framework knowledge, and supplying it here keeps
// the ci.yaml output of framework initializations complete. The Core no
// longer owns default pipeline template data and no longer falls back to
// a Core-owned CI pipeline when the adapter omits the CI definition
// (TS-015-01-02, ADR-026 decision 1) — the adapter supplies the full
// template set.
//
// Template freshness — tracking Flutter framework version updates against
// the template's build surface — is a standard maintenance responsibility
// (007 §7, Transition Plan §4.7); see templates/README.md.
//
// Reference: TS-007-038, TS-018-02-02, ADR-020 §1, ADR-018, MVP-002 §3.5
package flutter

import (
	"maleolabs.com/anvil-standard-flutter/internal/contracts"
	"maleolabs.com/anvil-standard-flutter/internal/pipeline"
)

// Template returns the pipeline definitions the Flutter adapter owns:
// the build pipeline — the dependencies stage (`flutter pub get`) and the
// build stage with its platform metadata — and the CI scaffold. The Core
// validates them through the pipeline loader and writes them to
// .anvil/pipelines/ at generation time (ADR-020 §1).
//
// Reference: TS-007-038, TS-018-02-02, ADR-020 §1, ADR-018
func Template() contracts.TemplateResult {
	return contracts.TemplateResult{
		Build: &pipeline.PipelineDefinition{
			Pipeline: pipeline.Pipeline{
				Name: "build",
				Stages: []pipeline.PipelineStage{
					{
						Name: "dependencies",
						Tasks: []pipeline.Task{
							{
								Name:    "flutter-pub-get",
								Command: "flutter",
								Args:    []string{"pub", "get"},
							},
						},
					},
					{
						Name: "build",
						Tasks: []pipeline.Task{
							{
								Name:    "flutter-web",
								Command: "flutter",
								Args:    []string{"build", "web"},
								Timeout: "10m",
								Metadata: &pipeline.TaskMetadata{
									Platforms: []string{PlatformLinux, PlatformDarwin, PlatformWindows},
									Target:    TargetWeb,
								},
							},
							{
								Name:    "flutter-apk",
								Command: "flutter",
								Args:    []string{"build", "apk", "--release"},
								Timeout: "15m",
								Metadata: &pipeline.TaskMetadata{
									Platforms: []string{PlatformLinux, PlatformDarwin, PlatformWindows},
									Target:    TargetApk,
								},
							},
							{
								Name:    "flutter-ios",
								Command: "flutter",
								Args:    []string{"build", "ios", "--release"},
								Metadata: &pipeline.TaskMetadata{
									Platforms: []string{PlatformDarwin},
									Target:    TargetIos,
								},
							},
						},
					},
				},
			},
		},
		CI: &pipeline.PipelineDefinition{
			Pipeline: pipeline.Pipeline{
				Name: "ci",
				Stages: []pipeline.PipelineStage{
					{
						Name: "build",
						Tasks: []pipeline.Task{
							{
								Name:    "build",
								Command: "echo",
								Args:    []string{"building..."},
							},
						},
					},
					{
						Name: "test",
						Tasks: []pipeline.Task{
							{
								Name:    "unit-tests",
								Command: "echo",
								Args:    []string{"running unit tests..."},
							},
							{
								Name:    "static-analysis",
								Command: "echo",
								Args:    []string{"running static analysis..."},
							},
							{
								Name:    "linting",
								Command: "echo",
								Args:    []string{"running linter..."},
							},
						},
					},
				},
			},
		},
	}
}
