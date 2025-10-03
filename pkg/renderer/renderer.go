package renderer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Renderer interface {
	Render(ctx context.Context, path string) ([]*unstructured.Unstructured, error)
}

type YAMLRenderer struct{}

func NewYAMLRenderer() *YAMLRenderer {
	return &YAMLRenderer{}
}

func (r *YAMLRenderer) Render(ctx context.Context, path string) ([]*unstructured.Unstructured, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
	}

	var files []string
	if info.IsDir() {
		files, err = r.findYAMLFiles(path)
		if err != nil {
			return nil, err
		}
	} else {
		files = []string{path}
	}

	var objects []*unstructured.Unstructured
	for _, file := range files {
		fileObjs, err := r.renderFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to render file %q: %w", file, err)
		}
		objects = append(objects, fileObjs...)
	}

	return objects, nil
}

func (r *YAMLRenderer) findYAMLFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func (r *YAMLRenderer) renderFile(file string) ([]*unstructured.Unstructured, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var objects []*unstructured.Unstructured

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	for {
		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}

		if obj == nil {
			continue
		}

		kind, ok := obj["kind"].(string)
		if !ok || kind == "" {
			continue
		}

		objJSON, err := json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal object: %w", err)
		}

		var normalized map[string]interface{}
		if err := json.Unmarshal(objJSON, &normalized); err != nil {
			return nil, fmt.Errorf("failed to unmarshal object: %w", err)
		}

		u := &unstructured.Unstructured{Object: normalized}
		objects = append(objects, u)
	}

	return objects, nil
}
