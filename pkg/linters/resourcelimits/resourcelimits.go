package resourcelimits

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
	Name        = "resource-limits"
	Description = "Ensures containers have resource requests and limits defined"
)

type Config struct {
	RequireCPULimit      bool     `mapstructure:"require-cpu-limit"`
	RequireMemoryLimit   bool     `mapstructure:"require-memory-limit"`
	RequireCPURequest    bool     `mapstructure:"require-cpu-request"`
	RequireMemoryRequest bool     `mapstructure:"require-memory-request"`
	ExcludeNamespaces    []string `mapstructure:"exclude-namespaces"`
}

func init() {
	linter.Register(&Linter{
		config: Config{
			RequireCPULimit:      true,
			RequireMemoryLimit:   true,
			RequireCPURequest:    true,
			RequireMemoryRequest: true,
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
	if !gvk.IsWorkload(obj) {
		return nil, nil
	}

	namespace := obj.GetNamespace()
	for _, ns := range l.config.ExcludeNamespaces {
		if ns == namespace {
			return nil, nil
		}
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
		resources, hasResources := containerMap["resources"].(map[string]interface{})

		if !hasResources {
			issues = append(issues, linter.Issue{
				Severity:   linter.SeverityError,
				Linter:     l.Name(),
				Message:    fmt.Sprintf("Container %q has no resource requirements", name),
				Resource:   common.ResourceRef(obj),
				Field:      fmt.Sprintf("spec.template.spec.containers[%d].resources", i),
				Suggestion: "Add resources.requests and resources.limits",
			})
			continue
		}

		limits, _ := resources["limits"].(map[string]interface{})
		requests, _ := resources["requests"].(map[string]interface{})

		if l.config.RequireCPULimit {
			if _, ok := limits["cpu"]; !ok {
				issues = append(issues, linter.Issue{
					Severity:   linter.SeverityError,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Container %q missing CPU limit", name),
					Resource:   common.ResourceRef(obj),
					Field:      fmt.Sprintf("spec.template.spec.containers[%d].resources.limits.cpu", i),
					Suggestion: "Add: resources.limits.cpu: \"1000m\"",
				})
			}
		}

		if l.config.RequireMemoryLimit {
			if _, ok := limits["memory"]; !ok {
				issues = append(issues, linter.Issue{
					Severity:   linter.SeverityError,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Container %q missing memory limit", name),
					Resource:   common.ResourceRef(obj),
					Field:      fmt.Sprintf("spec.template.spec.containers[%d].resources.limits.memory", i),
					Suggestion: "Add: resources.limits.memory: \"512Mi\"",
				})
			}
		}

		if l.config.RequireCPURequest {
			if _, ok := requests["cpu"]; !ok {
				issues = append(issues, linter.Issue{
					Severity:   linter.SeverityError,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Container %q missing CPU request", name),
					Resource:   common.ResourceRef(obj),
					Field:      fmt.Sprintf("spec.template.spec.containers[%d].resources.requests.cpu", i),
					Suggestion: "Add: resources.requests.cpu: \"100m\"",
				})
			}
		}

		if l.config.RequireMemoryRequest {
			if _, ok := requests["memory"]; !ok {
				issues = append(issues, linter.Issue{
					Severity:   linter.SeverityError,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Container %q missing memory request", name),
					Resource:   common.ResourceRef(obj),
					Field:      fmt.Sprintf("spec.template.spec.containers[%d].resources.requests.memory", i),
					Suggestion: "Add: resources.requests.memory: \"256Mi\"",
				})
			}
		}
	}

	return issues, nil
}
