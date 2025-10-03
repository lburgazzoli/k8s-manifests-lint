package output

import (
	"fmt"
	"io"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/output/githubactions"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/output/json"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/output/sarif"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/output/text"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/output/yaml"
)

type Formatter interface {
	Format(w io.Writer, issues []linter.Issue) error
}

func NewFormatter(format string, useColor bool) (Formatter, error) {
	switch format {
	case "text":
		return &text.Formatter{UseColor: useColor}, nil
	case "json":
		return &json.Formatter{}, nil
	case "yaml":
		return &yaml.Formatter{}, nil
	case "github-actions":
		return &githubactions.Formatter{}, nil
	case "sarif":
		return &sarif.Formatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}
