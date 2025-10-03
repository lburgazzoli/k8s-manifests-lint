package sarif

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
)

type Formatter struct{}

func (f *Formatter) Format(w io.Writer, issues []linter.Issue) error {
	rules := make(map[string]rule)
	results := make([]result, 0, len(issues))

	for _, issue := range issues {
		ruleID := issue.Linter

		if _, exists := rules[ruleID]; !exists {
			rules[ruleID] = rule{
				ID:   ruleID,
				Name: ruleID,
				ShortDescription: message{
					Text: fmt.Sprintf("Linter: %s", ruleID),
				},
			}
		}

		level := "error"
		switch issue.Severity {
		case linter.SeverityFatal:
			level = "error"
		case linter.SeverityWarning:
			level = "warning"
		case linter.SeverityInfo:
			level = "note"
		}

		resource := fmt.Sprintf("%s/%s", issue.Resource.Kind, issue.Resource.Name)
		if issue.Resource.Namespace != "" {
			resource = fmt.Sprintf("%s/%s", issue.Resource.Namespace, resource)
		}

		messageText := issue.Message
		if issue.Suggestion != "" {
			messageText = fmt.Sprintf("%s\nSuggestion: %s", messageText, issue.Suggestion)
		}

		result := result{
			RuleID:  ruleID,
			Level:   level,
			Message: message{Text: messageText},
			Locations: []location{
				{
					PhysicalLocation: physicalLocation{
						ArtifactLocation: artifactLocation{
							URI: resource,
						},
						Region: region{
							StartLine: 1,
						},
					},
					LogicalLocations: []logicalLocation{
						{
							Name:               resource,
							FullyQualifiedName: fmt.Sprintf("%s.%s", issue.Resource.APIVersion, resource),
							Kind:               "resource",
						},
					},
				},
			},
		}

		if issue.Field != "" {
			result.Locations[0].LogicalLocations[0].FullyQualifiedName = fmt.Sprintf("%s.%s.%s",
				issue.Resource.APIVersion, resource, issue.Field)
		}

		results = append(results, result)
	}

	rulesList := make([]rule, 0, len(rules))
	for _, rule := range rules {
		rulesList = append(rulesList, rule)
	}

	sort.Slice(rulesList, func(i, j int) bool {
		return rulesList[i].ID < rulesList[j].ID
	})

	report := Report{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Runs: []run{
			{
				Tool: tool{
					Driver: driver{
						Name:           "k8s-manifests-lint",
						InformationURI: "https://github.com/lburgazzoli/k8s-manifests-lint",
						Version:        "0.1.0",
						Rules:          rulesList,
					},
				},
				Results: results,
			},
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

type Report struct {
	Version string `json:"version"`
	Schema  string `json:"$schema"`
	Runs    []run  `json:"runs"`
}

type run struct {
	Tool    tool     `json:"tool"`
	Results []result `json:"results"`
}

type tool struct {
	Driver driver `json:"driver"`
}

type driver struct {
	Name           string `json:"name"`
	InformationURI string `json:"informationUri"`
	Version        string `json:"version"`
	Rules          []rule `json:"rules"`
}

type rule struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	ShortDescription message `json:"shortDescription"`
}

type result struct {
	RuleID    string     `json:"ruleId"`
	Level     string     `json:"level"`
	Message   message    `json:"message"`
	Locations []location `json:"locations"`
}

type message struct {
	Text string `json:"text"`
}

type location struct {
	PhysicalLocation physicalLocation  `json:"physicalLocation"`
	LogicalLocations []logicalLocation `json:"logicalLocations,omitempty"`
}

type physicalLocation struct {
	ArtifactLocation artifactLocation `json:"artifactLocation"`
	Region           region           `json:"region,omitempty"`
}

type artifactLocation struct {
	URI string `json:"uri"`
}

type region struct {
	StartLine int `json:"startLine,omitempty"`
}

type logicalLocation struct {
	Name               string `json:"name"`
	FullyQualifiedName string `json:"fullyQualifiedName,omitempty"`
	Kind               string `json:"kind,omitempty"`
}
