package requiredlabels

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/common"
)

const (
	Name        = "required-labels"
	Description = "Ensures resources have required labels"
)

type Config struct {
	Labels       []string `mapstructure:"labels"`
	ExcludeKinds []string `mapstructure:"exclude-kinds"`
}

func init() {
	linter.Register(&Linter{})
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
	kind := obj.GetKind()
	for _, excludeKind := range l.config.ExcludeKinds {
		if kind == excludeKind {
			return nil, nil
		}
	}

	var issues []linter.Issue
	labels := obj.GetLabels()

	for _, requiredLabel := range l.config.Labels {
		if _, ok := labels[requiredLabel]; !ok {
			issues = append(issues, linter.Issue{
				Severity:   linter.SeverityWarning,
				Linter:     l.Name(),
				Message:    fmt.Sprintf("Missing required label %q", requiredLabel),
				Resource:   common.ResourceRef(obj),
				Field:      "metadata.labels",
				Suggestion: fmt.Sprintf("Add label: %s: <value>", requiredLabel),
			})
		}
	}

	return issues, nil
}
