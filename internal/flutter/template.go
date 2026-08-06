// Adapter-owned pipeline template of the Flutter adapter (TS-007-038).
//
// The build pipeline definition the adapter owns is returned through the
// `template` command (contracts.CommandTemplate) and written by the Core
// to .anvil/pipelines/build.yaml at generation time, replacing the
// Core-embedded template function execution.FlutterBuildPipeline (ADR-020
// §1: framework knowledge moves OUT of the Core binary INTO the adapter
// binaries).
//
// The definition mirrors the targets the adapter's build pipeline
// executes (internal/flutter/build.go — the single source of build
// knowledge): `flutter build web`, `flutter build apk --release`, and
// `flutter build ios --release`. Each task preserves the ADR-018 platform
// metadata — the platforms that support the target and the target name —
// so the local engine keeps its platform-aware execution (skip
// unsupported targets with a warning; --target selection) on the
// adapter-owned template. The explicit timeouts (10m web, 15m apk) are
// preserved from the pre-ADR-020 Core template: Flutter builds (first
// Gradle run in particular) routinely exceed the engine's 5-minute
// default (found by E2E verification, ST-007-005).
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
// Reference: TS-007-038, ADR-020 §1, ADR-018, MVP-002 §3.5
package flutter

import (
	"maleolabs.com/anvil-standard-flutter/internal/contracts"
	"maleolabs.com/anvil-standard-flutter/internal/pipeline"
)

// Template returns the pipeline definitions the Flutter adapter owns:
// the build pipeline with its platform metadata and the CI scaffold. The
// Core validates them through the pipeline loader and writes them to
// .anvil/pipelines/ at generation time (ADR-020 §1).
//
// Reference: TS-007-038, ADR-020 §1, ADR-018
func Template() contracts.TemplateResult {
	return contracts.TemplateResult{
		Build: &pipeline.PipelineDefinition{
			Pipeline: pipeline.Pipeline{
				Name: "build",
				Stages: []pipeline.PipelineStage{
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
