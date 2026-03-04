package fhir

import "fmt"

type Bundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type"`
	Total        int           `json:"total"`
	Entry        []BundleEntry `json:"entry"`
}

type BundleEntry struct {
	FullURL  string      `json:"fullUrl,omitempty"`
	Resource interface{} `json:"resource"`
}

func NewSearchBundle(resources []interface{}, baseURL string) Bundle {
	entries := make([]BundleEntry, len(resources))
	for i, r := range resources {
		entries[i] = BundleEntry{
			FullURL:  fmt.Sprintf("%s/%d", baseURL, i),
			Resource: r,
		}
	}
	return Bundle{
		ResourceType: "Bundle",
		Type:         "searchset",
		Total:        len(resources),
		Entry:        entries,
	}
}

type OperationOutcome struct {
	ResourceType string                     `json:"resourceType"`
	Issue        []OperationOutcomeIssue    `json:"issue"`
}

type OperationOutcomeIssue struct {
	Severity    string          `json:"severity"`
	Code        string          `json:"code"`
	Diagnostics string          `json:"diagnostics,omitempty"`
	Details     *CodeableConcept `json:"details,omitempty"`
}

func NewErrorOutcome(severity, code, message string) OperationOutcome {
	return OperationOutcome{
		ResourceType: "OperationOutcome",
		Issue: []OperationOutcomeIssue{
			{
				Severity:    severity,
				Code:        code,
				Diagnostics: message,
			},
		},
	}
}
