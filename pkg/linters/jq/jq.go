package jq

import (
	"context"
	"fmt"

	"github.com/itchyny/gojq"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

type Rule struct {
	Expression string
	Message    string
	Severity   linter.Severity
	Field      string
	Suggestion string
}

type Linter struct {
	name        string
	description string
	rules       []Rule
}

type Factory struct{}

func (f *Factory) Create(name string, description string) linter.Linter {
	return &Linter{
		name:        name,
		description: description,
	}
}

func init() {
	linter.Register(&Linter{
		name:        "jq",
		description: "Evaluates custom jq expressions against Kubernetes resources",
	})
	linter.RegisterFactory("jq", &Factory{})
}

func New(name string, description string) *Linter {
	return &Linter{
		name:        name,
		description: description,
	}
}

func (l *Linter) Name() string {
	return l.name
}

func (l *Linter) Description() string {
	return l.description
}

func (l *Linter) Configure(settings map[string]interface{}) error {
	rulesData, ok := settings["rules"].([]interface{})
	if !ok {
		return fmt.Errorf("rules must be an array")
	}

	l.rules = make([]Rule, 0, len(rulesData))
	for i, ruleData := range rulesData {
		ruleMap, ok := ruleData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("rule %d must be an object", i)
		}

		rule := Rule{
			Severity: linter.SeverityError,
		}

		if expr, ok := ruleMap["expression"].(string); ok {
			rule.Expression = expr
		} else {
			return fmt.Errorf("rule %d: expression is required", i)
		}

		if msg, ok := ruleMap["message"].(string); ok {
			rule.Message = msg
		} else {
			return fmt.Errorf("rule %d: message is required", i)
		}

		if sev, ok := ruleMap["severity"].(string); ok {
			rule.Severity = linter.Severity(sev)
		}

		if field, ok := ruleMap["field"].(string); ok {
			rule.Field = field
		}

		if sugg, ok := ruleMap["suggestion"].(string); ok {
			rule.Suggestion = sugg
		}

		l.rules = append(l.rules, rule)
	}

	return nil
}

func (l *Linter) Lint(ctx context.Context, obj unstructured.Unstructured) ([]linter.Issue, error) {
	var issues []linter.Issue

	allObjects, _ := linter.AllObjectsFromContext(ctx)

	for _, rule := range l.rules {
		query, err := gojq.Parse(rule.Expression)
		if err != nil {
			return nil, fmt.Errorf("failed to parse jq expression %q: %w", rule.Expression, err)
		}

		code, err := gojq.Compile(query, gojq.WithVariables([]string{"$objects", "$object"}))
		if err != nil {
			return nil, fmt.Errorf("failed to compile jq expression %q: %w", rule.Expression, err)
		}

		allObjectsSlice := make([]interface{}, len(allObjects))
		for i, o := range allObjects {
			allObjectsSlice[i] = o.Object
		}

		iter := code.Run(nil, allObjectsSlice, obj.Object)
		for {
			result, ok := iter.Next()
			if !ok {
				break
			}

			if err, ok := result.(error); ok {
				return nil, fmt.Errorf("jq expression %q failed: %w", rule.Expression, err)
			}

			if result == nil || result == false {
				continue
			}

			issue := linter.Issue{
				Severity: rule.Severity,
				Linter:   l.Name(),
				Message:  rule.Message,
				Resource: linter.ResourceRef{
					APIVersion: obj.GetAPIVersion(),
					Kind:       obj.GetKind(),
					Namespace:  obj.GetNamespace(),
					Name:       obj.GetName(),
				},
				Field:      rule.Field,
				Suggestion: rule.Suggestion,
			}

			issues = append(issues, issue)
			break
		}
	}

	return issues, nil
}
