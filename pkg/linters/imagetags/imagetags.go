package imagetags

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/utils/gvk"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/utils/k8s"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linters/common"
)

const (
	Name        = "image-tags"
	Description = "Validates container image tags"
)

type Config struct {
	DisallowLatest        bool     `mapstructure:"disallow-latest"`
	RequireDigest         bool     `mapstructure:"require-digest"`
	AllowedRegistries     []string `mapstructure:"allowed-registries"`
	RequireVersionPattern string   `mapstructure:"require-version-pattern"`
}

func init() {
	linter.Register(&Linter{
		config: Config{
			DisallowLatest: true,
		},
	})
}

type Linter struct {
	config       Config
	versionRegex *regexp.Regexp
}

func (l *Linter) Name() string {
	return Name
}

func (l *Linter) Description() string {
	return Description
}

func (l *Linter) Configure(settings map[string]interface{}) error {
	if err := mapstructure.Decode(settings, &l.config); err != nil {
		return err
	}

	if l.config.RequireVersionPattern != "" {
		var err error
		l.versionRegex, err = regexp.Compile(l.config.RequireVersionPattern)
		if err != nil {
			return fmt.Errorf("invalid version pattern: %w", err)
		}
	}

	return nil
}

func (l *Linter) Lint(ctx context.Context, obj unstructured.Unstructured) ([]linter.Issue, error) {
	if !gvk.IsWorkloadOrPod(obj) {
		return nil, nil
	}

	var issues []linter.Issue

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

func (l *Linter) checkImage(obj unstructured.Unstructured, containerName string, image string, index int) []linter.Issue {
	var issues []linter.Issue

	parts := strings.Split(image, "@")
	hasDigest := len(parts) == 2

	if l.config.RequireDigest && !hasDigest {
		issues = append(issues, linter.Issue{
			Severity:   linter.SeverityWarning,
			Linter:     l.Name(),
			Message:    fmt.Sprintf("Container %q image should use digest", containerName),
			Resource:   common.ResourceRef(obj),
			Field:      fmt.Sprintf("spec.template.spec.containers[%d].image", index),
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

	if len(l.config.AllowedRegistries) > 0 && registry != "" {
		allowed := false
		for _, allowedRegistry := range l.config.AllowedRegistries {
			if registry == allowedRegistry || strings.HasPrefix(registry, allowedRegistry+"/") {
				allowed = true
				break
			}
		}

		if !allowed {
			issues = append(issues, linter.Issue{
				Severity:   linter.SeverityError,
				Linter:     l.Name(),
				Message:    fmt.Sprintf("Container %q uses disallowed registry %q", containerName, registry),
				Resource:   common.ResourceRef(obj),
				Field:      fmt.Sprintf("spec.template.spec.containers[%d].image", index),
				Suggestion: fmt.Sprintf("Use one of the allowed registries: %v", l.config.AllowedRegistries),
			})
		}
	}

	if tag != "" {
		if l.config.DisallowLatest && tag == "latest" {
			issues = append(issues, linter.Issue{
				Severity:   linter.SeverityError,
				Linter:     l.Name(),
				Message:    fmt.Sprintf("Container %q uses 'latest' tag", containerName),
				Resource:   common.ResourceRef(obj),
				Field:      fmt.Sprintf("spec.template.spec.containers[%d].image", index),
				Suggestion: "Specify an explicit version tag",
			})
		}

		if l.versionRegex != nil && !l.versionRegex.MatchString(tag) {
			issues = append(issues, linter.Issue{
				Severity:   linter.SeverityWarning,
				Linter:     l.Name(),
				Message:    fmt.Sprintf("Container %q tag %q doesn't match required pattern", containerName, tag),
				Resource:   common.ResourceRef(obj),
				Field:      fmt.Sprintf("spec.template.spec.containers[%d].image", index),
				Suggestion: fmt.Sprintf("Use tag matching pattern: %s", l.config.RequireVersionPattern),
			})
		}
	}

	return issues
}
