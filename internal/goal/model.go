package goal

import (
	"errors"

	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
)

type Goal struct {
	ResourceType      string                 `json:"resourceType"`
	ID                string                 `json:"id"`
	Meta              fhir.Meta              `json:"meta"`
	LifecycleStatus   string                 `json:"lifecycleStatus"`
	AchievementStatus *fhir.CodeableConcept  `json:"achievementStatus,omitempty"`
	Category          []fhir.CodeableConcept `json:"category,omitempty"`
	Priority          *fhir.CodeableConcept  `json:"priority,omitempty"`
	Description       fhir.CodeableConcept   `json:"description"`
	Subject           fhir.Reference         `json:"subject"`
	Target            []GoalTarget           `json:"target,omitempty"`
	StartDate         string                 `json:"startDate,omitempty"`
	StatusDate        string                 `json:"statusDate,omitempty"`
	Note              []fhir.Annotation      `json:"note,omitempty"`
}

type GoalTarget struct {
	Measure        *fhir.CodeableConcept `json:"measure,omitempty"`
	DetailQuantity *fhir.Quantity        `json:"detailQuantity,omitempty"`
	DueDate        string                `json:"dueDate,omitempty"`
}

const (
	LifecycleProposed       = "proposed"
	LifecyclePlanned        = "planned"
	LifecycleAccepted       = "accepted"
	LifecycleActive         = "active"
	LifecycleOnHold         = "on-hold"
	LifecycleCompleted      = "completed"
	LifecycleCancelled      = "cancelled"
	LifecycleEnteredInError = "entered-in-error"
	LifecycleRejected       = "rejected"
)

const (
	AchievementInProgress    = "in-progress"
	AchievementImproving     = "improving"
	AchievementWorsening     = "worsening"
	AchievementNoChange      = "no-change"
	AchievementAchieved      = "achieved"
	AchievementSustaining    = "sustaining"
	AchievementNotAchieved   = "not-achieved"
	AchievementNoProgress    = "no-progress"
	AchievementNotAttainable = "not-attainable"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

func isValidLifecycleStatus(s string) bool {
	switch s {
	case LifecycleProposed, LifecyclePlanned, LifecycleAccepted, LifecycleActive,
		LifecycleOnHold, LifecycleCompleted, LifecycleCancelled, LifecycleEnteredInError,
		LifecycleRejected:
		return true
	}
	return false
}
