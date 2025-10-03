package json

import (
	"encoding/json"
	"io"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

type Formatter struct{}

func (f *Formatter) Format(w io.Writer, issues []linter.Issue) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}
