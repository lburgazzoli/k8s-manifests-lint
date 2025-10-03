package healthprobes

import (
	"context"
	"fmt"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/utils/gvk"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/utils/k8s"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/common"
)

const (
	Name        = "health-probes"
	Description = "Ensures pods have liveness and readiness probes"
)

type Config struct {
	RequireLiveness  bool     `mapstructure:"require-liveness"`
	RequireReadiness bool     `mapstructure:"require-readiness"`
	ExcludeKinds     []string `mapstructure:"exclude-kinds"`
}

func init() {
	linter.Register(&Linter{
		config: Config{
			RequireLiveness:  true,
			RequireReadiness: true,
		},
	})
}

type Linter struct {
	config Config
}

func (l *Linter) Name() string {
	return Name
}

func (l *Linter) Description() string {
	return Description
}

func (l *Linter) Configure(settings map[string]interface{}) error {
	return mapstructure.Decode(settings, &l.config)
}

func (l *Linter) Lint(ctx context.Context, obj unstructured.Unstructured) ([]linter.Issue, error) {
	kind := obj.GetKind()

	for _, excludeKind := range l.config.ExcludeKinds {
		if kind == excludeKind {
			return nil, nil
		}
	}

	if !gvk.IsWorkloadOrPod(obj) {
		return nil, nil
	}

	var issues []linter.Issue

	containers, err := k8s.GetContainers(obj)
	if err != nil {
		return nil, err
	}

	for i, container := range containers {
		containerMap, ok := container.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := containerMap["name"].(string)

		if l.config.RequireLiveness {
			if _, ok := containerMap["livenessProbe"]; !ok {
				issues = append(issues, linter.Issue{
					Severity:   linter.SeverityWarning,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Container %q missing livenessProbe", name),
					Resource:   common.ResourceRef(obj),
					Field:      fmt.Sprintf("spec.template.spec.containers[%d].livenessProbe", i),
					Suggestion: "Add a livenessProbe to detect and recover from failures",
				})
			}
		}

		if l.config.RequireReadiness {
			if _, ok := containerMap["readinessProbe"]; !ok {
				issues = append(issues, linter.Issue{
					Severity:   linter.SeverityWarning,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Container %q missing readinessProbe", name),
					Resource:   common.ResourceRef(obj),
					Field:      fmt.Sprintf("spec.template.spec.containers[%d].readinessProbe", i),
					Suggestion: "Add a readinessProbe to control traffic routing",
				})
			}
		}
	}

	return issues, nil
}
