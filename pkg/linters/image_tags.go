package linters

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/k8s"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

func init() {
	linter.Register(&ImageTagsLinter{
		disallowLatest: true,
	})
}

type ImageTagsLinter struct {
	disallowLatest       bool
	requireDigest        bool
	allowedRegistries    []string
	requireVersionPattern string
	versionRegex         *regexp.Regexp
}

func (l *ImageTagsLinter) Name() string {
	return "image-tags"
}

func (l *ImageTagsLinter) Description() string {
	return "Validates container image tags"
}

func (l *ImageTagsLinter) Configure(settings map[string]interface{}) error {
	if v, ok := settings["disallow-latest"].(bool); ok {
		l.disallowLatest = v
	}
	if v, ok := settings["require-digest"].(bool); ok {
		l.requireDigest = v
	}

	if v, ok := settings["allowed-registries"].([]interface{}); ok {
		l.allowedRegistries = make([]string, 0, len(v))
		for _, reg := range v {
			if regStr, ok := reg.(string); ok {
				l.allowedRegistries = append(l.allowedRegistries, regStr)
			}
		}
	}

	if v, ok := settings["require-version-pattern"].(string); ok {
		l.requireVersionPattern = v
		var err error
		l.versionRegex, err = regexp.Compile(v)
		if err != nil {
			return fmt.Errorf("invalid version pattern: %w", err)
		}
	}

	return nil
}

func (l *ImageTagsLinter) Lint(ctx context.Context, obj *unstructured.Unstructured) ([]linter.Issue, error) {
	kind := obj.GetKind()
	if kind != "Deployment" && kind != "StatefulSet" && kind != "DaemonSet" && kind != "Job" && kind != "CronJob" && kind != "Pod" {
		return nil, nil
	}

	var issues []linter.Issue

	// Use gojq helper to get containers
	containers, err := k8s.GetContainers(obj)
	if err != nil {
		return nil, err
	}

	for i, container := range containers {
		containerMap, ok := container.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := containerMap["name"].(string)
		image, ok := containerMap["image"].(string)
		if !ok {
			continue
		}

		containerIssues := l.checkImage(obj, name, image, i)
		issues = append(issues, containerIssues...)
	}

	return issues, nil
}

func (l *ImageTagsLinter) checkImage(obj *unstructured.Unstructured, containerName string, image string, index int) []linter.Issue {
	var issues []linter.Issue

	parts := strings.Split(image, "@")
	hasDigest := len(parts) == 2

	if l.requireDigest && !hasDigest {
		issues = append(issues, linter.Issue{
			Severity: linter.SeverityWarning,
			Linter:   l.Name(),
			Message:  fmt.Sprintf("Container %q image should use digest", containerName),
			Resource: resourceRef(obj),
			Field:    fmt.Sprintf("spec.template.spec.containers[%d].image", index),
			Suggestion: "Use image with SHA256 digest: image@sha256:...",
		})
	}

	imageWithoutDigest := parts[0]
	registry := ""
	imageName := imageWithoutDigest
	tag := ""

	if strings.Contains(imageWithoutDigest, "/") {
		registryAndImage := strings.SplitN(imageWithoutDigest, "/", 2)
		if strings.Contains(registryAndImage[0], ".") || strings.Contains(registryAndImage[0], ":") {
			registry = registryAndImage[0]
			imageName = registryAndImage[1]
		}
	}

	if strings.Contains(imageName, ":") {
		parts := strings.SplitN(imageName, ":", 2)
		imageName = parts[0]
		tag = parts[1]
	}

	if len(l.allowedRegistries) > 0 && registry != "" {
		allowed := false
		for _, allowedRegistry := range l.allowedRegistries {
			if registry == allowedRegistry || strings.HasPrefix(registry, allowedRegistry+"/") {
				allowed = true
				break
			}
		}

		if !allowed {
			issues = append(issues, linter.Issue{
				Severity: linter.SeverityError,
				Linter:   l.Name(),
				Message:  fmt.Sprintf("Container %q uses disallowed registry %q", containerName, registry),
				Resource: resourceRef(obj),
				Field:    fmt.Sprintf("spec.template.spec.containers[%d].image", index),
				Suggestion: fmt.Sprintf("Use one of the allowed registries: %v", l.allowedRegistries),
			})
		}
	}

	if tag != "" {
		if l.disallowLatest && tag == "latest" {
			issues = append(issues, linter.Issue{
				Severity: linter.SeverityError,
				Linter:   l.Name(),
				Message:  fmt.Sprintf("Container %q uses 'latest' tag", containerName),
				Resource: resourceRef(obj),
				Field:    fmt.Sprintf("spec.template.spec.containers[%d].image", index),
				Suggestion: "Specify an explicit version tag",
			})
		}

		if l.versionRegex != nil && !l.versionRegex.MatchString(tag) {
			issues = append(issues, linter.Issue{
				Severity: linter.SeverityWarning,
				Linter:   l.Name(),
				Message:  fmt.Sprintf("Container %q tag %q doesn't match required pattern", containerName, tag),
				Resource: resourceRef(obj),
				Field:    fmt.Sprintf("spec.template.spec.containers[%d].image", index),
				Suggestion: fmt.Sprintf("Use tag matching pattern: %s", l.requireVersionPattern),
			})
		}
	}

	return issues
}
