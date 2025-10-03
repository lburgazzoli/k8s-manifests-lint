package yaml

import (
	"io"

	"gopkg.in/yaml.v3"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

type Formatter struct{}

func (f *Formatter) Format(w io.Writer, issues []linter.Issue) error {
	encoder := yaml.NewEncoder(w)
	defer encoder.Close()
	return encoder.Encode(map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}
