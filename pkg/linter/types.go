package linter

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type contextKey int

const (
	allObjectsKey contextKey = iota
)

type Severity string

const (
	SeverityFatal   Severity = "fatal"
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type ResourceRef struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
	Namespace  string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Name       string `json:"name" yaml:"name"`
}

type Issue struct {
	Severity   Severity    `json:"severity" yaml:"severity"`
	Linter     string      `json:"linter" yaml:"linter"`
	Message    string      `json:"message" yaml:"message"`
	Resource   ResourceRef `json:"resource" yaml:"resource"`
	Field      string      `json:"field,omitempty" yaml:"field,omitempty"`
	Suggestion string      `json:"suggestion,omitempty" yaml:"suggestion,omitempty"`
}

type Linter interface {
	Name() string
	Description() string
	Lint(ctx context.Context, obj unstructured.Unstructured) ([]Issue, error)
	Configure(settings map[string]interface{}) error
}

// WithAllObjects adds all objects to the context
func WithAllObjects(ctx context.Context, objects []unstructured.Unstructured) context.Context {
	return context.WithValue(ctx, allObjectsKey, objects)
}

// AllObjectsFromContext retrieves all objects from the context
func AllObjectsFromContext(ctx context.Context) ([]unstructured.Unstructured, bool) {
	objects, ok := ctx.Value(allObjectsKey).([]unstructured.Unstructured)
	return objects, ok
}
