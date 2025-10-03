package githubactions

import (
	"fmt"
	"io"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

type Formatter struct{}

func (f *Formatter) Format(w io.Writer, issues []linter.Issue) error {
	for _, issue := range issues {
		resource := fmt.Sprintf("%s/%s", issue.Resource.Kind, issue.Resource.Name)
		if issue.Resource.Namespace != "" {
			resource = fmt.Sprintf("%s/%s", issue.Resource.Namespace, resource)
		}

		level := "error"
		switch issue.Severity {
		case linter.SeverityFatal:
			level = "error"
		case linter.SeverityWarning:
			level = "warning"
		case linter.SeverityInfo:
			level = "notice"
		}

		title := fmt.Sprintf("[%s] %s", issue.Linter, resource)
		message := issue.Message
		if issue.Suggestion != "" {
			message = fmt.Sprintf("%s (Suggestion: %s)", message, issue.Suggestion)
		}

		fmt.Fprintf(w, "::%s title=%s::%s\n", level, title, message)
	}

	return nil
}
