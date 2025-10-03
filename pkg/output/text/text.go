package text

import (
	"fmt"
	"io"
	"sort"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

type Formatter struct {
	UseColor bool
}

func (f *Formatter) Format(w io.Writer, issues []linter.Issue) error {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Resource.Kind != issues[j].Resource.Kind {
			return issues[i].Resource.Kind < issues[j].Resource.Kind
		}
		if issues[i].Resource.Name != issues[j].Resource.Name {
			return issues[i].Resource.Name < issues[j].Resource.Name
		}
		return issues[i].Linter < issues[j].Linter
	})

	for _, issue := range issues {
		severity := issue.Severity
		if f.UseColor {
			switch issue.Severity {
			case linter.SeverityFatal:
				severity = "\033[31;1mfatal\033[0m"
			case linter.SeverityError:
				severity = "\033[31merror\033[0m"
			case linter.SeverityWarning:
				severity = "\033[33mwarning\033[0m"
			case linter.SeverityInfo:
				severity = "\033[36minfo\033[0m"
			}
		}

		resource := fmt.Sprintf("%s/%s", issue.Resource.Kind, issue.Resource.Name)
		if issue.Resource.Namespace != "" {
			resource = fmt.Sprintf("%s/%s", issue.Resource.Namespace, resource)
		}

		fmt.Fprintf(w, "[%s] %s: %s (%s)\n", severity, resource, issue.Message, issue.Linter)

		if issue.Field != "" {
			fmt.Fprintf(w, "  Field: %s\n", issue.Field)
		}
		if issue.Suggestion != "" {
			fmt.Fprintf(w, "  Suggestion: %s\n", issue.Suggestion)
		}
	}

	if len(issues) > 0 {
		fmt.Fprintf(w, "\nFound %d issue(s)\n", len(issues))
	}

	return nil
}
