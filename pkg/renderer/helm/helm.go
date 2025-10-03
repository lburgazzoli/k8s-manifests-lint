package helm

import (
	"context"
	"fmt"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
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
	chartSource := r.source.Chart
	if chartSource == "" {
		chartSource = path
	}

	values := make(map[string]any)
	if r.source.Data != nil {
		values = r.source.Data
	}

	namespace := "default"
	if ns, ok := values["namespace"].(string); ok {
		namespace = ns
	}

	releaseName := "release"
	if name, ok := values["releaseName"].(string); ok {
		releaseName = name
	}

	helmRenderer, err := helm.New([]helm.Data{
		{
			ChartSource: chartSource,
			ReleaseName: releaseName,
			Namespace:   namespace,
			Values:      values,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create helm renderer: %w", err)
	}

	objects, err := helmRenderer.Process(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to render helm chart: %w", err)
	}

	return objects, nil
}