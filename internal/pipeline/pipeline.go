// Package pipeline mirrors the Anvil Core's pipeline definition types
// (the Core's internal/execution PipelineDefinition family) for the
// `template` command of the standard command contract.
//
// The `template` command returns the standard-owned build and CI
// pipeline definitions as JSON on stdout. The Anvil Runtime parses that
// JSON into its own pipeline definition types and validates/writes them
// to .anvil/pipelines/ (ADR-020 §1). The JSON keys this package emits
// are the Go field names of the definition structs (the Core's types
// carry yaml tags only, so encoding/json uses field names on both
// sides); the struct shapes here are byte-identical to the Core's, so
// the wire output is unchanged from the pre-split adapter.
//
// Reference: TS-007-038, ADR-020 §1; Core internal/execution/pipeline.go
package pipeline

import (
	"errors"
	"fmt"
)

// PipelineDefinition is the top-level YAML structure for pipeline files.
type PipelineDefinition struct {
	Pipeline Pipeline `yaml:"pipeline"`
}

// Pipeline represents a complete workflow with one or more PipelineStages.
type Pipeline struct {
	Name   string            `yaml:"name"`
	Stages []PipelineStage   `yaml:"stages"`
	Env    map[string]string `yaml:"env,omitempty"`
}

// PipelineStage groups Tasks that share an execution context or dependency boundary.
type PipelineStage struct {
	Name     string `yaml:"name"`
	Parallel bool   `yaml:"parallel,omitempty"`
	Tasks    []Task `yaml:"tasks"`
}

// Task is the atomic unit of execution.
type Task struct {
	Name       string            `yaml:"name"`
	Command    string            `yaml:"command"`
	Args       []string          `yaml:"args,omitempty"`
	WorkingDir string            `yaml:"working_dir,omitempty"`
	Env        map[string]string `yaml:"env,omitempty"`
	Timeout    string            `yaml:"timeout,omitempty"` // duration string like "30s"

	// Environments holds environment-aware overrides keyed by environment name.
	Environments map[string]TaskOverride `yaml:"environments,omitempty"`

	// Metadata holds optional platform-aware execution metadata
	// (ADR-018): the platforms that support the task and the build
	// target it produces. The pipeline engine uses it for
	// platform-aware execution and --target selection.
	Metadata *TaskMetadata `yaml:"metadata,omitempty"`
}

// TaskMetadata declares platform-aware execution metadata for a Task
// (ADR-018). Platforms lists the canonical platform identifiers
// ("linux", "darwin", "windows") that support the task; Target names the
// build target the task produces.
type TaskMetadata struct {
	Platforms []string `yaml:"platforms,omitempty"`
	Target    string   `yaml:"target,omitempty"`
}

// TaskOverride allows environment-specific overrides for a Task.
type TaskOverride struct {
	Command    string            `yaml:"command,omitempty"`
	Args       []string          `yaml:"args,omitempty"`
	WorkingDir string            `yaml:"working_dir,omitempty"`
	Env        map[string]string `yaml:"env,omitempty"`
	Timeout    string            `yaml:"timeout,omitempty"`
}

// ValidationError reports one pipeline definition validation failure.
type ValidationError struct {
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return e.Message
}

// Unwrap returns nil so that errors.Is can match against ValidationError
// sentinels directly. Each ValidationError is a leaf error.
func (e *ValidationError) Unwrap() error {
	return nil
}

// Validate checks that the PipelineDefinition has all required fields
// filled in:
//
//   - Pipeline name must be non-empty
//   - At least one stage must be defined
//   - Each stage must have a non-empty name
//   - Each stage must have at least one task
//   - Each task must have a non-empty name and command
//
// When multiple fields are invalid, all errors are collected and returned
// together via errors.Join.
func (pd PipelineDefinition) Validate() error {
	var errs []error

	if pd.Pipeline.Name == "" {
		errs = append(errs, &ValidationError{
			Message: "pipeline name is required",
		})
	}

	if len(pd.Pipeline.Stages) == 0 {
		errs = append(errs, &ValidationError{
			Message: "pipeline must have at least one stage",
		})
	}

	for i, stage := range pd.Pipeline.Stages {
		if stage.Name == "" {
			errs = append(errs, &ValidationError{
				Message: fmt.Sprintf("stage %d: name is required", i),
			})
		}
		if len(stage.Tasks) == 0 {
			name := stage.Name
			if name == "" {
				name = fmt.Sprintf("%d", i)
			}
			errs = append(errs, &ValidationError{
				Message: fmt.Sprintf("stage %q: must have at least one task", name),
			})
		}
		for j, task := range stage.Tasks {
			if task.Name == "" {
				errs = append(errs, &ValidationError{
					Message: fmt.Sprintf("stage %q task %d: name is required", stage.Name, j),
				})
			}
			if task.Command == "" {
				errs = append(errs, &ValidationError{
					Message: fmt.Sprintf("stage %q task %q: command is required", stage.Name, task.Name),
				})
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
