package gotemplate

import (
	"context"
	"fmt"
	"os"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/gotemplate"
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
	templatePath := r.source.Path
	if templatePath == "" {
		templatePath = path
	}

	values := r.source.Data
	if values == nil {
		values = make(map[string]interface{})
	}

	fs := os.DirFS(".")

	templateRenderer := gotemplate.New([]gotemplate.Data{
		{
			FS:     fs,
			Path:   templatePath,
			Values: values,
		},
	})

	objects, err := templateRenderer.Process(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to render go templates: %w", err)
	}

	return objects, nil
}