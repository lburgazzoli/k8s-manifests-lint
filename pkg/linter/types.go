package linter

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type ResourceRef struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}

type Issue struct {
	Severity   Severity
	Linter     string
	Message    string
	Resource   ResourceRef
	Field      string
	Suggestion string
}

type Linter interface {
	Name() string
	Description() string
	Lint(ctx context.Context, obj *unstructured.Unstructured) ([]Issue, error)
	Configure(settings map[string]interface{}) error
}
