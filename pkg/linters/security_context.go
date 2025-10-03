package linters

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/k8s"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

func init() {
	linter.Register(&SecurityContextLinter{
		requireRunAsNonRoot:       true,
		disallowPrivilegeEscalation: true,
	})
}

type SecurityContextLinter struct {
	requireRunAsNonRoot         bool
	requireReadOnlyRootFilesystem bool
	disallowPrivilegeEscalation bool
	requiredDroppedCapabilities []string
}

func (l *SecurityContextLinter) Name() string {
	return "security-context"
}

func (l *SecurityContextLinter) Description() string {
	return "Validates pod and container security contexts"
}

func (l *SecurityContextLinter) Configure(settings map[string]interface{}) error {
	if v, ok := settings["require-run-as-non-root"].(bool); ok {
		l.requireRunAsNonRoot = v
	}
	if v, ok := settings["require-read-only-root-filesystem"].(bool); ok {
		l.requireReadOnlyRootFilesystem = v
	}
	if v, ok := settings["disallow-privilege-escalation"].(bool); ok {
		l.disallowPrivilegeEscalation = v
	}

	if v, ok := settings["required-dropped-capabilities"].([]interface{}); ok {
		l.requiredDroppedCapabilities = make([]string, 0, len(v))
		for _, cap := range v {
			if capStr, ok := cap.(string); ok {
				l.requiredDroppedCapabilities = append(l.requiredDroppedCapabilities, capStr)
			}
		}
	}

	return nil
}

func (l *SecurityContextLinter) Lint(ctx context.Context, obj *unstructured.Unstructured) ([]linter.Issue, error) {
	kind := obj.GetKind()
	if kind != "Deployment" && kind != "StatefulSet" && kind != "DaemonSet" && kind != "Job" && kind != "CronJob" && kind != "Pod" {
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
		securityContext, _ := containerMap["securityContext"].(map[string]interface{})

		if l.requireRunAsNonRoot {
			runAsNonRoot, ok := securityContext["runAsNonRoot"].(bool)
			if !ok || !runAsNonRoot {
				issues = append(issues, linter.Issue{
					Severity: linter.SeverityError,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Container %q must set runAsNonRoot to true", name),
					Resource: resourceRef(obj),
					Field:    fmt.Sprintf("spec.template.spec.containers[%d].securityContext.runAsNonRoot", i),
					Suggestion: "Add: securityContext.runAsNonRoot: true",
				})
			}
		}

		if l.requireReadOnlyRootFilesystem {
			readOnlyRootFilesystem, ok := securityContext["readOnlyRootFilesystem"].(bool)
			if !ok || !readOnlyRootFilesystem {
				issues = append(issues, linter.Issue{
					Severity: linter.SeverityWarning,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Container %q should set readOnlyRootFilesystem to true", name),
					Resource: resourceRef(obj),
					Field:    fmt.Sprintf("spec.template.spec.containers[%d].securityContext.readOnlyRootFilesystem", i),
					Suggestion: "Add: securityContext.readOnlyRootFilesystem: true",
				})
			}
		}

		if l.disallowPrivilegeEscalation {
			allowPrivilegeEscalation, ok := securityContext["allowPrivilegeEscalation"].(bool)
			if !ok || allowPrivilegeEscalation {
				issues = append(issues, linter.Issue{
					Severity: linter.SeverityError,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Container %q must set allowPrivilegeEscalation to false", name),
					Resource: resourceRef(obj),
					Field:    fmt.Sprintf("spec.template.spec.containers[%d].securityContext.allowPrivilegeEscalation", i),
					Suggestion: "Add: securityContext.allowPrivilegeEscalation: false",
				})
			}
		}

		if len(l.requiredDroppedCapabilities) > 0 {
			capabilities, _ := securityContext["capabilities"].(map[string]interface{})
			drop, _ := capabilities["drop"].([]interface{})

			droppedCaps := make(map[string]bool)
			for _, d := range drop {
				if capStr, ok := d.(string); ok {
					droppedCaps[capStr] = true
				}
			}

			for _, requiredCap := range l.requiredDroppedCapabilities {
				if !droppedCaps[requiredCap] {
					issues = append(issues, linter.Issue{
						Severity: linter.SeverityWarning,
						Linter:   l.Name(),
						Message:  fmt.Sprintf("Container %q should drop capability %q", name, requiredCap),
						Resource: resourceRef(obj),
						Field:    fmt.Sprintf("spec.template.spec.containers[%d].securityContext.capabilities.drop", i),
						Suggestion: fmt.Sprintf("Add %q to capabilities.drop", requiredCap),
					})
				}
			}
		}
	}

	return issues, nil
}
