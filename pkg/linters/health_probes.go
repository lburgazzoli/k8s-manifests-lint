package linters

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/k8s"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

func init() {
	linter.Register(&HealthProbesLinter{
		requireLiveness:  true,
		requireReadiness: true,
	})
}

type HealthProbesLinter struct {
	requireLiveness  bool
	requireReadiness bool
	excludeKinds     []string
}

func (l *HealthProbesLinter) Name() string {
	return "health-probes"
}

func (l *HealthProbesLinter) Description() string {
	return "Ensures pods have liveness and readiness probes"
}

func (l *HealthProbesLinter) Configure(settings map[string]interface{}) error {
	if v, ok := settings["require-liveness"].(bool); ok {
		l.requireLiveness = v
	}
	if v, ok := settings["require-readiness"].(bool); ok {
		l.requireReadiness = v
	}

	if v, ok := settings["exclude-kinds"].([]interface{}); ok {
		l.excludeKinds = make([]string, 0, len(v))
		for _, kind := range v {
			if kindStr, ok := kind.(string); ok {
				l.excludeKinds = append(l.excludeKinds, kindStr)
			}
		}
	}

	return nil
}

func (l *HealthProbesLinter) Lint(ctx context.Context, obj *unstructured.Unstructured) ([]linter.Issue, error) {
	kind := obj.GetKind()

	for _, excludeKind := range l.excludeKinds {
		if kind == excludeKind {
			return nil, nil
		}
	}

	if kind != "Deployment" && kind != "StatefulSet" && kind != "DaemonSet" && kind != "Pod" {
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

		if l.requireLiveness {
			if _, ok := containerMap["livenessProbe"]; !ok {
				issues = append(issues, linter.Issue{
					Severity: linter.SeverityWarning,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Container %q missing livenessProbe", name),
					Resource: resourceRef(obj),
					Field:    fmt.Sprintf("spec.template.spec.containers[%d].livenessProbe", i),
					Suggestion: "Add a livenessProbe to detect and recover from failures",
				})
			}
		}

		if l.requireReadiness {
			if _, ok := containerMap["readinessProbe"]; !ok {
				issues = append(issues, linter.Issue{
					Severity: linter.SeverityWarning,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Container %q missing readinessProbe", name),
					Resource: resourceRef(obj),
					Field:    fmt.Sprintf("spec.template.spec.containers[%d].readinessProbe", i),
					Suggestion: "Add a readinessProbe to control traffic routing",
				})
			}
		}
	}

	return issues, nil
}
