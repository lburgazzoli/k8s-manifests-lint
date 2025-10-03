package clusterrolebindingsecurity

import (
	"context"
	"fmt"
	"strings"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/utils/gvk"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/utils/jq"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/common"
)

const (
	Name        = "cluster-role-binding-security"
	Description = "Validates ClusterRoleBindings for overly permissive group assignments"
)

type Config struct {
	DisallowedGroups           []string `mapstructure:"disallowed-groups"`
	WarnNamespaceGroups        bool     `mapstructure:"warn-namespace-groups"`
	AllowedRolesForBroadGroups []string `mapstructure:"allowed-roles-for-broad-groups"`
	CriticalRoles              []string `mapstructure:"critical-roles"`
}

func init() {
	linter.Register(&Linter{
		config: Config{
			DisallowedGroups: []string{
				"system:authenticated",
				"system:unauthenticated",
				"system:serviceaccounts",
			},
			WarnNamespaceGroups: true,
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
	if !gvk.IsGVK(obj, gvk.ClusterRoleBinding) {
		return nil, nil
	}

	var issues []linter.Issue

	roleName, _, err := jq.QueryString(obj, ".roleRef.name")
	if err != nil {
		return nil, err
	}

	groupSubjects, err := jq.QueryArray(obj, `.subjects[] | select(.kind == "Group") | .name`)
	if err != nil {
		return nil, err
	}

	for _, subject := range groupSubjects {
		name, ok := subject.(string)
		if !ok {
			continue
		}

		for _, disallowedGroup := range l.config.DisallowedGroups {
			if name == disallowedGroup {
				severity := linter.SeverityError
				if l.isRoleAllowed(roleName) {
					severity = linter.SeverityWarning
				}

				issues = append(issues, linter.Issue{
					Severity:   severity,
					Linter:     l.Name(),
					Message:    fmt.Sprintf("Binds to dangerous group %q (role: %s)", name, roleName),
					Resource:   common.ResourceRef(obj),
					Field:      "subjects",
					Suggestion: "Use specific ServiceAccounts or Users instead of broad groups",
				})
			}
		}

		if l.config.WarnNamespaceGroups && strings.HasPrefix(name, "system:serviceaccounts:") {
			severity := linter.SeverityWarning
			if l.isCriticalRole(roleName) {
				severity = linter.SeverityError
			}

			namespace := strings.TrimPrefix(name, "system:serviceaccounts:")
			issues = append(issues, linter.Issue{
				Severity:   severity,
				Linter:     l.Name(),
				Message:    fmt.Sprintf("Binds to all ServiceAccounts in namespace %q (role: %s)", namespace, roleName),
				Resource:   common.ResourceRef(obj),
				Field:      "subjects",
				Suggestion: "Use specific ServiceAccount instead of namespace-wide group",
			})
		}
	}

	return issues, nil
}

func (l *Linter) isRoleAllowed(roleName string) bool {
	for _, allowed := range l.config.AllowedRolesForBroadGroups {
		if roleName == allowed {
			return true
		}
	}
	return false
}

func (l *Linter) isCriticalRole(roleName string) bool {
	for _, critical := range l.config.CriticalRoles {
		if roleName == critical {
			return true
		}
	}
	return false
}
