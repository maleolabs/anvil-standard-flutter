// Tests for the Flutter adapter-owned pipeline template (TS-007-038,
// TS-018-02-02): the build pipeline definition — the dependencies stage
// (`flutter pub get`) and the build stage mirroring the build target
// table — and the CI scaffold, as returned through the `template`
// command. The definitions must pass the pipeline loader validation
// (pipeline.Validate, the mirror of the Core loader) and must not drift
// from the executed build knowledge (build.go).
package flutter

import (
	"testing"
)

// TestTemplate_Valid verifies that both returned definitions are
// well-formed pipeline definitions: non-empty pipeline name, stages with
// names and tasks, tasks with names and commands — the pipeline loader
// validation of the Anvil Runtime (TS-007-038; the mirror's Validate).
func TestTemplate_Valid(t *testing.T) {
	result := Template()

	if result.Build == nil {
		t.Fatal("Build = nil, want the build pipeline definition")
	}
	if err := result.Build.Validate(); err != nil {
		t.Errorf("Build definition failed pipeline validation: %v", err)
	}
	if result.Build.Pipeline.Name != "build" {
		t.Errorf("Build.Pipeline.Name = %q, want %q", result.Build.Pipeline.Name, "build")
	}

	if result.CI == nil {
		t.Fatal("CI = nil, want the CI scaffold definition")
	}
	if err := result.CI.Validate(); err != nil {
		t.Errorf("CI definition failed pipeline validation: %v", err)
	}
	if result.CI.Pipeline.Name != "ci" {
		t.Errorf("CI.Pipeline.Name = %q, want %q", result.CI.Pipeline.Name, "ci")
	}
}

// TestTemplate_BuildPipelineDepth verifies the build pipeline covers the
// Flutter hybrid build steps at the same convention depth as the Laravel
// template (TS-018-02-02, Review 19 §3.3): a "dependencies" stage first —
// `flutter pub get` resolves the package graph before any build runs, the
// same position Laravel's composer install holds — followed by the
// "build" stage with the build targets.
func TestTemplate_BuildPipelineDepth(t *testing.T) {
	result := Template()
	stages := result.Build.Pipeline.Stages

	if len(stages) != 2 {
		t.Fatalf("Build stages = %d, want 2 (dependencies, build)", len(stages))
	}

	dependencies := stages[0]
	if dependencies.Name != "dependencies" {
		t.Errorf("stage[0].Name = %q, want %q", dependencies.Name, "dependencies")
	}
	if len(dependencies.Tasks) != 1 {
		t.Fatalf("dependencies stage tasks = %d, want 1", len(dependencies.Tasks))
	}
	pubGet := dependencies.Tasks[0]
	if pubGet.Name != "flutter-pub-get" {
		t.Errorf("dependencies task name = %q, want %q", pubGet.Name, "flutter-pub-get")
	}
	if pubGet.Command != "flutter" {
		t.Errorf("dependencies task command = %q, want %q", pubGet.Command, "flutter")
	}
	if len(pubGet.Args) != 2 || pubGet.Args[0] != "pub" || pubGet.Args[1] != "get" {
		t.Errorf("dependencies task args = %v, want [pub get]", pubGet.Args)
	}
	// The dependency step is not a build target: no ADR-018 target
	// metadata, and no timeout — `flutter pub get` is fast and
	// platform-independent.
	if pubGet.Metadata != nil {
		t.Errorf("dependencies task Metadata = %v, want nil (not a build target)", pubGet.Metadata)
	}
	if pubGet.Timeout != "" {
		t.Errorf("dependencies task Timeout = %q, want empty", pubGet.Timeout)
	}

	buildStage := stages[1]
	if buildStage.Name != "build" {
		t.Errorf("stage[1].Name = %q, want %q", buildStage.Name, "build")
	}
	if len(buildStage.Tasks) != len(buildTargets) {
		t.Errorf("build stage tasks = %d, want %d", len(buildStage.Tasks), len(buildTargets))
	}
}

// TestTemplate_BuildStageMirrorsBuildTargets verifies the single-source
// requirement (TS-007-038): the build stage tasks mirror the commands of
// the adapter's build target table — the same framework knowledge must
// not drift between the template and the executed build phases. Each task
// preserves the ADR-018 platform metadata (platforms + target name) and
// the execution order (web → apk → ios) of the table.
func TestTemplate_BuildStageMirrorsBuildTargets(t *testing.T) {
	result := Template()
	tasks := result.Build.Pipeline.Stages[1].Tasks

	if len(tasks) != len(buildTargets) {
		t.Fatalf("build stage tasks = %d, want %d", len(tasks), len(buildTargets))
	}
	for i, target := range buildTargets {
		task := tasks[i]
		if task.Name != "flutter-"+target.Name {
			t.Errorf("task %d name = %q, want %q", i, task.Name, "flutter-"+target.Name)
		}
		if task.Command != "flutter" {
			t.Errorf("task %q command = %q, want %q", task.Name, task.Command, "flutter")
		}
		if len(task.Args) != len(target.Args) {
			t.Errorf("task %q args = %v, want %v (build target table)", task.Name, task.Args, target.Args)
			continue
		}
		for j, arg := range target.Args {
			if task.Args[j] != arg {
				t.Errorf("task %q arg %d = %q, want %q (build target table)", task.Name, j, task.Args[j], arg)
			}
		}
		if task.Metadata == nil {
			t.Errorf("task %q Metadata = nil, want platform metadata (ADR-018)", task.Name)
			continue
		}
		if task.Metadata.Target != target.Name {
			t.Errorf("task %q Metadata.Target = %q, want %q", task.Name, task.Metadata.Target, target.Name)
		}
		if len(task.Metadata.Platforms) != len(target.Platforms) {
			t.Errorf("task %q Metadata.Platforms = %v, want %v", task.Name, task.Metadata.Platforms, target.Platforms)
			continue
		}
		for j, platform := range target.Platforms {
			if task.Metadata.Platforms[j] != platform {
				t.Errorf("task %q Metadata.Platforms[%d] = %q, want %q", task.Name, j, task.Metadata.Platforms[j], platform)
			}
		}
	}
}

// TestTemplate_BuildStageTimeouts verifies the explicit build timeouts
// are preserved on the adapter-owned template: Flutter builds (first
// Gradle run in particular) routinely exceed the engine's 5-minute
// default (ST-007-005). The web task keeps 10m, the apk task 15m; the
// ios task keeps no explicit timeout, matching the build target table.
func TestTemplate_BuildStageTimeouts(t *testing.T) {
	result := Template()
	tasks := result.Build.Pipeline.Stages[1].Tasks

	timeouts := map[string]string{}
	for _, task := range tasks {
		timeouts[task.Name] = task.Timeout
	}

	want := map[string]string{
		"flutter-web": "10m",
		"flutter-apk": "15m",
		"flutter-ios": "",
	}
	for taskName, wantTimeout := range want {
		if timeouts[taskName] != wantTimeout {
			t.Errorf("task %q Timeout = %q, want %q", taskName, timeouts[taskName], wantTimeout)
		}
	}
}

// TestTemplate_CIScaffold verifies the CI definition supplies the generic
// scaffold (build + test placeholder stages) so the ci.yaml output of
// framework initializations stays complete (TS-015-01-02, ADR-026
// decision 1). The CI pipeline is generic placeholder data, not framework
// knowledge.
func TestTemplate_CIScaffold(t *testing.T) {
	result := Template()
	stages := result.CI.Pipeline.Stages

	if len(stages) != 2 {
		t.Fatalf("CI stages = %d, want 2 (build, test)", len(stages))
	}
	if stages[0].Name != "build" {
		t.Errorf("CI stage[0].Name = %q, want %q", stages[0].Name, "build")
	}
	if stages[1].Name != "test" {
		t.Errorf("CI stage[1].Name = %q, want %q", stages[1].Name, "test")
	}
	if len(stages[0].Tasks) < 1 {
		t.Error("CI build stage has no tasks")
	}
	if len(stages[1].Tasks) < 1 {
		t.Error("CI test stage has no tasks")
	}
}

// TestTemplate_NoEnvironmentOverrides verifies the template definitions
// carry no environment-specific overrides: the definitions are the same
// for every environment, keeping the generated build.yaml deterministic
// (TS-018-02-02).
func TestTemplate_NoEnvironmentOverrides(t *testing.T) {
	result := Template()

	for _, stage := range result.Build.Pipeline.Stages {
		for _, task := range stage.Tasks {
			if task.Environments != nil {
				t.Errorf("build task %q declares Environments overrides, want none", task.Name)
			}
		}
	}
	for _, stage := range result.CI.Pipeline.Stages {
		for _, task := range stage.Tasks {
			if task.Environments != nil {
				t.Errorf("CI task %q declares Environments overrides, want none", task.Name)
			}
		}
	}
}
