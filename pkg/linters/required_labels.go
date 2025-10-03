package linters

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

func init() {
	linter.Register(&RequiredLabelsLinter{})
}

type RequiredLabelsLinter struct {
	labels       []string
	excludeKinds []string
}

func (l *RequiredLabelsLinter) Name() string {
	return "required-labels"
}

func (l *RequiredLabelsLinter) Description() string {
	return "Ensures resources have required labels"
}

func (l *RequiredLabelsLinter) Configure(settings map[string]interface{}) error {
	if v, ok := settings["labels"].([]interface{}); ok {
		l.labels = make([]string, 0, len(v))
		for _, label := range v {
			if labelStr, ok := label.(string); ok {
				l.labels = append(l.labels, labelStr)
			}
		}
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

func (l *RequiredLabelsLinter) Lint(ctx context.Context, obj *unstructured.Unstructured) ([]linter.Issue, error) {
	kind := obj.GetKind()
	for _, excludeKind := range l.excludeKinds {
		if kind == excludeKind {
			return nil, nil
		}
	}

	var issues []linter.Issue
	labels := obj.GetLabels()

	for _, requiredLabel := range l.labels {
		if _, ok := labels[requiredLabel]; !ok {
			issues = append(issues, linter.Issue{
				Severity: linter.SeverityWarning,
				Linter:   l.Name(),
				Message:  fmt.Sprintf("Missing required label %q", requiredLabel),
				Resource: resourceRef(obj),
				Field:    "metadata.labels",
				Suggestion: fmt.Sprintf("Add label: %s: <value>", requiredLabel),
			})
		}
	}

	return issues, nil
}
