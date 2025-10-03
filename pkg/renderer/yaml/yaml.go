package yaml

import (
	"context"
	"fmt"
	"os"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
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
	searchPath := path
	if r.source.Path != "" {
		searchPath = r.source.Path
	}

	info, err := os.Stat(searchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %q: %w", searchPath, err)
	}

	pattern := searchPath
	if info.IsDir() {
		pattern = searchPath + "/**/*.{yaml,yml}"
	}

	fs := os.DirFS(".")

	yamlRenderer := yaml.New([]yaml.Data{
		{
			FS:   fs,
			Path: pattern,
		},
	})

	objects, err := yamlRenderer.Process(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to render YAML: %w", err)
	}

	return objects, nil
}