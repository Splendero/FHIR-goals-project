package careplan

import "github.com/spencerosborn/fhir-goals-engine/internal/fhir"

var (
	ValidStatuses = map[string]bool{
		"draft":            true,
		"active":           true,
		"on-hold":          true,
		"revoked":          true,
		"completed":        true,
		"entered-in-error": true,
		"unknown":          true,
	}

	ValidIntents = map[string]bool{
		"proposal": true,
		"plan":     true,
		"order":    true,
		"option":   true,
	}
)

type CarePlan struct {
	ResourceType string                 `json:"resourceType"`
	ID           string                 `json:"id"`
	Meta         fhir.Meta              `json:"meta"`
	Status       string                 `json:"status"`
	Intent       string                 `json:"intent"`
	Title        string                 `json:"title,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Subject      fhir.Reference         `json:"subject"`
	Period       *fhir.Period           `json:"period,omitempty"`
	Goal         []fhir.Reference       `json:"goal,omitempty"`
	Category     []fhir.CodeableConcept `json:"category,omitempty"`
}
