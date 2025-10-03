package kustomize

import (
	"context"
	"fmt"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/config"
)

type Renderer struct {
	source config.Source
}

func New(source config.Source) *Renderer {
	return &Renderer{source: source}
}

func (r *Renderer) Render(ctx context.Context, path string) ([]unstructured.Unstructured, error) {
	basePath := r.source.Path
	if basePath == "" {
		basePath = path
	}

	kustomizeRenderer := kustomize.New(basePath)

	objects, err := kustomizeRenderer.Process(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to render kustomize: %w", err)
	}

	return objects, nil
}