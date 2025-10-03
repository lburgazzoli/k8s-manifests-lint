package renderer

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/config"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/renderer/gotemplate"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/renderer/kustomize"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/renderer/yaml"
)

type Renderer interface {
	Render(ctx context.Context, path string) ([]unstructured.Unstructured, error)
}

func NewFromSource(source config.Source) (Renderer, error) {
	switch source.Type {
	case config.SourceTypeYAML, "":
		return yaml.New(source), nil
	case config.SourceTypeHelm:
		return helm.New(source), nil
	case config.SourceTypeKustomize:
		return kustomize.New(source), nil
	case config.SourceTypeGoTemplate, config.SourceTypeTemplate:
		return gotemplate.New(source), nil
	default:
		return nil, fmt.Errorf("unsupported source type: %s", source.Type)
	}
}