package patient

import "github.com/spencerosborn/fhir-goals-engine/internal/fhir"

type Patient struct {
	ResourceType string            `json:"resourceType"`
	ID           string            `json:"id"`
	Meta         fhir.Meta         `json:"meta"`
	Active       bool              `json:"active"`
	Name         []fhir.HumanName  `json:"name"`
	Gender       string            `json:"gender,omitempty"`
	BirthDate    string            `json:"birthDate,omitempty"`
	Identifier   []fhir.Identifier `json:"identifier,omitempty"`
}
