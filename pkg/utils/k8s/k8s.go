package k8s

import (
	"fmt"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/utils/gvk"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/utils/jq"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GetContainers is a helper to get containers from various resource types
func GetContainers(obj unstructured.Unstructured) ([]interface{}, error) {
	var query string
	switch {
	case gvk.IsGVK(obj, gvk.CronJob):
		query = ".spec.jobTemplate.spec.template.spec.containers"
	case gvk.IsGVK(obj, gvk.Pod):
		query = ".spec.containers"
	case gvk.IsGVK(obj, gvk.Deployment):
		query = ".spec.template.spec.containers"
	default:
		return nil, fmt.Errorf(
			"unsuported type: %s:%s",
			obj.GroupVersionKind().GroupVersion(),
			obj.GroupVersionKind().Kind,
		)
	}

	result, err := jq.Query(obj, query)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	if containers, ok := result.([]interface{}); ok {
		return containers, nil
	}

	return nil, nil
}
