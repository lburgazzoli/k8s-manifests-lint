package securitycontext

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
	Name        = "security-context"
	Description = "Validates pod and container security contexts"
)

type Config struct {
	RequireRunAsNonRoot           bool     `mapstructure:"require-run-as-non-root"`
	RequireReadOnlyRootFilesystem bool     `mapstructure:"require-read-only-root-filesystem"`
	DisallowPrivilegeEscalation   bool     `mapstructure:"disallow-privilege-escalation"`
	RequiredDroppedCapabilities   []string `mapstructure:"required-dropped-capabilities"`
}

func init() {
	linter.Register(&Linter{
		config: Config{
			RequireRunAsNonRoot:         true,
			DisallowPrivilegeEscalation: true,
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
		securityContext, _ := containerMap["securityContext"].(map[string]interface{})

		if l.config.RequireRunAsNonRoot {
			runAsNonRoot, ok := securityContext["runAsNonRoot"].(bool)
			if !ok || !runAsNonRoot {
				issues = append(issues, linter.Issue{
					Severity:   linter.SeverityError,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Container %q must set runAsNonRoot to true", name),
					Resource:   common.ResourceRef(obj),
					Field:      fmt.Sprintf("spec.template.spec.containers[%d].securityContext.runAsNonRoot", i),
					Suggestion: "Add: securityContext.runAsNonRoot: true",
				})
			}
		}

		if l.config.RequireReadOnlyRootFilesystem {
			readOnlyRootFilesystem, ok := securityContext["readOnlyRootFilesystem"].(bool)
			if !ok || !readOnlyRootFilesystem {
				issues = append(issues, linter.Issue{
					Severity:   linter.SeverityWarning,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Container %q should set readOnlyRootFilesystem to true", name),
					Resource:   common.ResourceRef(obj),
					Field:      fmt.Sprintf("spec.template.spec.containers[%d].securityContext.readOnlyRootFilesystem", i),
					Suggestion: "Add: securityContext.readOnlyRootFilesystem: true",
				})
			}
		}

		if l.config.DisallowPrivilegeEscalation {
			allowPrivilegeEscalation, ok := securityContext["allowPrivilegeEscalation"].(bool)
			if !ok || allowPrivilegeEscalation {
				issues = append(issues, linter.Issue{
					Severity:   linter.SeverityError,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Container %q must set allowPrivilegeEscalation to false", name),
					Resource:   common.ResourceRef(obj),
					Field:      fmt.Sprintf("spec.template.spec.containers[%d].securityContext.allowPrivilegeEscalation", i),
					Suggestion: "Add: securityContext.allowPrivilegeEscalation: false",
				})
			}
		}

		if len(l.config.RequiredDroppedCapabilities) > 0 {
			capabilities, _ := securityContext["capabilities"].(map[string]interface{})
			drop, _ := capabilities["drop"].([]interface{})

			droppedCaps := make(map[string]bool)
			for _, d := range drop {
				if capStr, ok := d.(string); ok {
					droppedCaps[capStr] = true
				}
			}

			for _, requiredCap := range l.config.RequiredDroppedCapabilities {
				if !droppedCaps[requiredCap] {
					issues = append(issues, linter.Issue{
						Severity:   linter.SeverityWarning,
						Linter:     l.Name(),
						Message:    fmt.Sprintf("Container %q should drop capability %q", name, requiredCap),
						Resource:   common.ResourceRef(obj),
						Field:      fmt.Sprintf("spec.template.spec.containers[%d].securityContext.capabilities.drop", i),
						Suggestion: fmt.Sprintf("Add %q to capabilities.drop", requiredCap),
					})
				}
			}
		}
	}

	return issues, nil
}
