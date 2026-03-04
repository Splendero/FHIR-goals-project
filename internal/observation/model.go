package observation

import "github.com/spencerosborn/fhir-goals-engine/internal/fhir"

var ValidStatuses = map[string]bool{
	"registered":       true,
	"preliminary":      true,
	"final":            true,
	"amended":          true,
	"corrected":        true,
	"cancelled":        true,
	"entered-in-error": true,
	"unknown":          true,
}

type Observation struct {
	ResourceType      string                 `json:"resourceType"`
	ID                string                 `json:"id"`
	Meta              fhir.Meta              `json:"meta"`
	Status            string                 `json:"status"`
	Category          []fhir.CodeableConcept `json:"category,omitempty"`
	Code              fhir.CodeableConcept   `json:"code"`
	Subject           fhir.Reference         `json:"subject"`
	EffectiveDateTime string                 `json:"effectiveDateTime,omitempty"`
	ValueQuantity     *fhir.Quantity         `json:"valueQuantity,omitempty"`
	Note              []fhir.Annotation      `json:"note,omitempty"`
}
