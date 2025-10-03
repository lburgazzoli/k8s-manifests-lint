package linters

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/k8s"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

func init() {
	linter.Register(&ResourceLimitsLinter{
		requireCPULimit:    true,
		requireMemoryLimit: true,
		requireCPURequest:  true,
		requireMemoryRequest: true,
	})
}

type ResourceLimitsLinter struct {
	requireCPULimit      bool
	requireMemoryLimit   bool
	requireCPURequest    bool
	requireMemoryRequest bool
	excludeNamespaces    []string
}

func (l *ResourceLimitsLinter) Name() string {
	return "resource-limits"
}

func (l *ResourceLimitsLinter) Description() string {
	return "Ensures containers have resource requests and limits defined"
}

func (l *ResourceLimitsLinter) Configure(settings map[string]interface{}) error {
	if v, ok := settings["require-cpu-limit"].(bool); ok {
		l.requireCPULimit = v
	}
	if v, ok := settings["require-memory-limit"].(bool); ok {
		l.requireMemoryLimit = v
	}
	if v, ok := settings["require-cpu-request"].(bool); ok {
		l.requireCPURequest = v
	}
	if v, ok := settings["require-memory-request"].(bool); ok {
		l.requireMemoryRequest = v
	}

	if v, ok := settings["exclude-namespaces"].([]interface{}); ok {
		l.excludeNamespaces = make([]string, 0, len(v))
		for _, ns := range v {
			if nsStr, ok := ns.(string); ok {
				l.excludeNamespaces = append(l.excludeNamespaces, nsStr)
			}
		}
	}

	return nil
}

func (l *ResourceLimitsLinter) Lint(ctx context.Context, obj *unstructured.Unstructured) ([]linter.Issue, error) {
	kind := obj.GetKind()
	if kind != "Deployment" && kind != "StatefulSet" && kind != "DaemonSet" && kind != "Job" && kind != "CronJob" {
		return nil, nil
	}

	namespace := obj.GetNamespace()
	for _, ns := range l.excludeNamespaces {
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
				Severity: linter.SeverityError,
				Linter:   l.Name(),
				Message:  fmt.Sprintf("Container %q has no resource requirements", name),
				Resource: resourceRef(obj),
				Field:    fmt.Sprintf("spec.template.spec.containers[%d].resources", i),
				Suggestion: "Add resources.requests and resources.limits",
			})
			continue
		}

		limits, _ := resources["limits"].(map[string]interface{})
		requests, _ := resources["requests"].(map[string]interface{})

		if l.requireCPULimit {
			if _, ok := limits["cpu"]; !ok {
				issues = append(issues, linter.Issue{
					Severity: linter.SeverityError,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Container %q missing CPU limit", name),
					Resource: resourceRef(obj),
					Field:    fmt.Sprintf("spec.template.spec.containers[%d].resources.limits.cpu", i),
					Suggestion: "Add: resources.limits.cpu: \"1000m\"",
				})
			}
		}

		if l.requireMemoryLimit {
			if _, ok := limits["memory"]; !ok {
				issues = append(issues, linter.Issue{
					Severity: linter.SeverityError,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Container %q missing memory limit", name),
					Resource: resourceRef(obj),
					Field:    fmt.Sprintf("spec.template.spec.containers[%d].resources.limits.memory", i),
					Suggestion: "Add: resources.limits.memory: \"512Mi\"",
				})
			}
		}

		if l.requireCPURequest {
			if _, ok := requests["cpu"]; !ok {
				issues = append(issues, linter.Issue{
					Severity: linter.SeverityError,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Container %q missing CPU request", name),
					Resource: resourceRef(obj),
					Field:    fmt.Sprintf("spec.template.spec.containers[%d].resources.requests.cpu", i),
					Suggestion: "Add: resources.requests.cpu: \"100m\"",
				})
			}
		}

		if l.requireMemoryRequest {
			if _, ok := requests["memory"]; !ok {
				issues = append(issues, linter.Issue{
					Severity: linter.SeverityError,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Container %q missing memory request", name),
					Resource: resourceRef(obj),
					Field:    fmt.Sprintf("spec.template.spec.containers[%d].resources.requests.memory", i),
					Suggestion: "Add: resources.requests.memory: \"256Mi\"",
				})
			}
		}
	}

	return issues, nil
}

func (l *ResourceLimitsLinter) getContainers(obj *unstructured.Unstructured) ([]interface{}, error) {
	kind := obj.GetKind()

	var path []string
	if kind == "CronJob" {
		path = []string{"spec", "jobTemplate", "spec", "template", "spec", "containers"}
	} else {
		path = []string{"spec", "template", "spec", "containers"}
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, path...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	return containers, nil
}

func (l *ResourceLimitsLinter) resourceRef(obj *unstructured.Unstructured) linter.ResourceRef {
	return linter.ResourceRef{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Namespace:  obj.GetNamespace(),
		Name:       obj.GetName(),
	}
}
