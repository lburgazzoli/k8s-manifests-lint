package linter

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type RunnerConfig struct {
	EnabledLinters  []string
	DisabledLinters []string
	Settings        map[string]map[string]interface{}
}

type Runner struct {
	linters []Linter
	config  *RunnerConfig
}

func NewRunner(config *RunnerConfig) (*Runner, error) {
	enabledMap := make(map[string]bool)
	for _, name := range config.EnabledLinters {
		enabledMap[name] = true
	}

	disabledMap := make(map[string]bool)
	for _, name := range config.DisabledLinters {
		disabledMap[name] = true
	}

	var linters []Linter
	for _, l := range All() {
		name := l.Name()

		if len(enabledMap) > 0 && !enabledMap[name] {
			continue
		}

		if disabledMap[name] {
			continue
		}

		if settings, ok := config.Settings[name]; ok {
			if err := l.Configure(settings); err != nil {
				return nil, fmt.Errorf("failed to configure linter %q: %w", name, err)
			}
		}

		linters = append(linters, l)
	}

	return &Runner{
		linters: linters,
		config:  config,
	}, nil
}

func (r *Runner) Run(ctx context.Context, objects []unstructured.Unstructured) ([]Issue, error) {
	var issues []Issue

	ctx = WithAllObjects(ctx, objects)

	for _, obj := range objects {
		for _, linter := range r.linters {
			objIssues, err := linter.Lint(ctx, obj)
			if err != nil {
				return issues, fmt.Errorf("linter %q failed on %s/%s: %w",
					linter.Name(), obj.GetKind(), obj.GetName(), err)
			}

			issues = append(issues, objIssues...)
		}
	}

	return issues, nil
}

func (r *Runner) Linters() []Linter {
	return r.linters
}
