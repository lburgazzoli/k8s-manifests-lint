package linters

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/k8s"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

func init() {
	linter.Register(&ClusterRoleBindingSecurityLinter{
		disallowedGroups: []string{
			"system:authenticated",
			"system:unauthenticated",
			"system:serviceaccounts",
		},
		warnNamespaceGroups: true,
	})
}

type ClusterRoleBindingSecurityLinter struct {
	disallowedGroups        []string
	warnNamespaceGroups     bool
	allowedRolesForBroadGroups []string
	criticalRoles           []string
}

func (l *ClusterRoleBindingSecurityLinter) Name() string {
	return "cluster-role-binding-security"
}

func (l *ClusterRoleBindingSecurityLinter) Description() string {
	return "Validates ClusterRoleBindings for overly permissive group assignments"
}

func (l *ClusterRoleBindingSecurityLinter) Configure(settings map[string]interface{}) error {
	if v, ok := settings["disallowed-groups"].([]interface{}); ok {
		l.disallowedGroups = make([]string, 0, len(v))
		for _, group := range v {
			if groupStr, ok := group.(string); ok {
				l.disallowedGroups = append(l.disallowedGroups, groupStr)
			}
		}
	}

	if v, ok := settings["warn-namespace-groups"].(bool); ok {
		l.warnNamespaceGroups = v
	}

	if v, ok := settings["allowed-roles-for-broad-groups"].([]interface{}); ok {
		l.allowedRolesForBroadGroups = make([]string, 0, len(v))
		for _, role := range v {
			if roleStr, ok := role.(string); ok {
				l.allowedRolesForBroadGroups = append(l.allowedRolesForBroadGroups, roleStr)
			}
		}
	}

	if v, ok := settings["critical-roles"].([]interface{}); ok {
		l.criticalRoles = make([]string, 0, len(v))
		for _, role := range v {
			if roleStr, ok := role.(string); ok {
				l.criticalRoles = append(l.criticalRoles, roleStr)
			}
		}
	}

	return nil
}

func (l *ClusterRoleBindingSecurityLinter) Lint(ctx context.Context, obj *unstructured.Unstructured) ([]linter.Issue, error) {
	if obj.GetKind() != "ClusterRoleBinding" {
		return nil, nil
	}

	var issues []linter.Issue

	// Get role name using gojq
	roleName, _, err := k8s.QueryString(obj, ".roleRef.name")
	if err != nil {
		return nil, err
	}

	// Get all Group subjects using gojq
	groupSubjects, err := k8s.QueryArray(obj, `.subjects[] | select(.kind == "Group") | .name`)
	if err != nil {
		return nil, err
	}

	for _, subject := range groupSubjects {
		name, ok := subject.(string)
		if !ok {
			continue
		}

		// Check disallowed groups
		for _, disallowedGroup := range l.disallowedGroups {
			if name == disallowedGroup {
				severity := linter.SeverityError
				if l.isRoleAllowed(roleName) {
					severity = linter.SeverityWarning
				}

				issues = append(issues, linter.Issue{
					Severity: severity,
					Linter:   l.Name(),
					Message:  fmt.Sprintf("Binds to dangerous group %q (role: %s)", name, roleName),
					Resource: resourceRef(obj),
					Field:    "subjects",
					Suggestion: "Use specific ServiceAccounts or Users instead of broad groups",
				})
			}
		}

		// Check namespace-wide service account groups
		if l.warnNamespaceGroups && strings.HasPrefix(name, "system:serviceaccounts:") {
			severity := linter.SeverityWarning
			if l.isCriticalRole(roleName) {
				severity = linter.SeverityError
			}

			namespace := strings.TrimPrefix(name, "system:serviceaccounts:")
			issues = append(issues, linter.Issue{
				Severity: severity,
				Linter:   l.Name(),
				Message:  fmt.Sprintf("Binds to all ServiceAccounts in namespace %q (role: %s)", namespace, roleName),
				Resource: resourceRef(obj),
				Field:    "subjects",
				Suggestion: "Use specific ServiceAccount instead of namespace-wide group",
			})
		}
	}

	return issues, nil
}

func (l *ClusterRoleBindingSecurityLinter) isRoleAllowed(roleName string) bool {
	for _, allowed := range l.allowedRolesForBroadGroups {
		if roleName == allowed {
			return true
		}
	}
	return false
}

func (l *ClusterRoleBindingSecurityLinter) isCriticalRole(roleName string) bool {
	for _, critical := range l.criticalRoles {
		if roleName == critical {
			return true
		}
	}
	return false
}
