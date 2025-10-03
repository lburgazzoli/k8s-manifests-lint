package linter

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type RunnerConfig struct {
	EnabledLinters  []string
	DisabledLinters []string
	Settings        map[string]map[string]interface{}
	Concurrency     int
}

type Runner struct {
	linters []Linter
	config  *RunnerConfig
}

func NewRunner(config *RunnerConfig) (*Runner, error) {
	if config.Concurrency <= 0 {
		config.Concurrency = 4
	}

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

func (r *Runner) Run(ctx context.Context, objects []*unstructured.Unstructured) ([]Issue, error) {
	var (
		mu     sync.Mutex
		issues []Issue
	)

	sem := make(chan struct{}, r.config.Concurrency)
	errChan := make(chan error, len(objects)*len(r.linters))
	var wg sync.WaitGroup

	for _, obj := range objects {
		for _, linter := range r.linters {
			wg.Add(1)
			go func(obj *unstructured.Unstructured, linter Linter) {
				defer wg.Done()

				sem <- struct{}{}
				defer func() { <-sem }()

				// Deep copy the object to avoid concurrent map access
				objCopy := obj.DeepCopy()

				objIssues, err := linter.Lint(ctx, objCopy)
				if err != nil {
					errChan <- fmt.Errorf("linter %q failed on %s/%s: %w",
						linter.Name(), objCopy.GetKind(), objCopy.GetName(), err)
					return
				}

				if len(objIssues) > 0 {
					mu.Lock()
					issues = append(issues, objIssues...)
					mu.Unlock()
				}
			}(obj, linter)
		}
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return issues, <-errChan
	}

	return issues, nil
}

func (r *Runner) Linters() []Linter {
	return r.linters
}
