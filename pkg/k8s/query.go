package k8s

import (
	"fmt"

	"github.com/itchyny/gojq"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Query executes a jq-style query on an unstructured object
func Query(obj *unstructured.Unstructured, query string) (interface{}, error) {
	q, err := gojq.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query %q: %w", query, err)
	}

	iter := q.Run(obj.Object)
	v, ok := iter.Next()
	if !ok {
		return nil, nil
	}

	if err, ok := v.(error); ok {
		return nil, err
	}

	return v, nil
}

// QueryArray executes a jq-style query and returns results as a slice
func QueryArray(obj *unstructured.Unstructured, query string) ([]interface{}, error) {
	q, err := gojq.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query %q: %w", query, err)
	}

	iter := q.Run(obj.Object)
	var results []interface{}

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		results = append(results, v)
	}

	return results, nil
}

// QueryString executes a jq-style query and returns the result as a string
func QueryString(obj *unstructured.Unstructured, query string) (string, bool, error) {
	v, err := Query(obj, query)
	if err != nil {
		return "", false, err
	}
	if v == nil {
		return "", false, nil
	}
	if s, ok := v.(string); ok {
		return s, true, nil
	}
	return "", false, nil
}

// QueryBool executes a jq-style query and returns the result as a bool
func QueryBool(obj *unstructured.Unstructured, query string) (bool, bool, error) {
	v, err := Query(obj, query)
	if err != nil {
		return false, false, err
	}
	if v == nil {
		return false, false, nil
	}
	if b, ok := v.(bool); ok {
		return b, true, nil
	}
	return false, false, nil
}

// QueryInt executes a jq-style query and returns the result as an int
func QueryInt(obj *unstructured.Unstructured, query string) (int, bool, error) {
	v, err := Query(obj, query)
	if err != nil {
		return 0, false, err
	}
	if v == nil {
		return 0, false, nil
	}

	switch n := v.(type) {
	case int:
		return n, true, nil
	case float64:
		return int(n), true, nil
	}

	return 0, false, nil
}

// QueryExists checks if a field exists (returns non-null value)
func QueryExists(obj *unstructured.Unstructured, query string) (bool, error) {
	v, err := Query(obj, query)
	if err != nil {
		return false, err
	}
	return v != nil, nil
}

// GetContainers is a helper to get containers from various resource types
func GetContainers(obj *unstructured.Unstructured) ([]interface{}, error) {
	kind := obj.GetKind()

	var query string
	switch kind {
	case "CronJob":
		query = ".spec.jobTemplate.spec.template.spec.containers"
	case "Pod":
		query = ".spec.containers"
	default:
		query = ".spec.template.spec.containers"
	}

	result, err := Query(obj, query)
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
