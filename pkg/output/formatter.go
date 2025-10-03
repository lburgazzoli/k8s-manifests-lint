package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

type Formatter interface {
	Format(w io.Writer, issues []linter.Issue) error
}

func NewFormatter(format string, useColor bool) (Formatter, error) {
	switch format {
	case "text":
		return &TextFormatter{UseColor: useColor}, nil
	case "json":
		return &JSONFormatter{}, nil
	case "yaml":
		return &YAMLFormatter{}, nil
	case "github-actions":
		return &GitHubActionsFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

type TextFormatter struct {
	UseColor bool
}

func (f *TextFormatter) Format(w io.Writer, issues []linter.Issue) error {
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

type JSONFormatter struct{}

func (f *JSONFormatter) Format(w io.Writer, issues []linter.Issue) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

type YAMLFormatter struct{}

func (f *YAMLFormatter) Format(w io.Writer, issues []linter.Issue) error {
	encoder := yaml.NewEncoder(w)
	defer encoder.Close()
	return encoder.Encode(map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

type GitHubActionsFormatter struct{}

func (f *GitHubActionsFormatter) Format(w io.Writer, issues []linter.Issue) error {
	for _, issue := range issues {
		resource := fmt.Sprintf("%s/%s", issue.Resource.Kind, issue.Resource.Name)
		if issue.Resource.Namespace != "" {
			resource = fmt.Sprintf("%s/%s", issue.Resource.Namespace, resource)
		}

		level := "error"
		switch issue.Severity {
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
