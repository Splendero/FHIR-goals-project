package observation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/spencerosborn/fhir-goals-engine/internal/goal"
)

var (
	ErrInvalidStatus = errors.New("invalid observation status")
	ErrCodeRequired  = errors.New("observation code is required")
	ErrNotFound      = errors.New("observation not found")
)

type GoalEvaluator interface {
	EvaluateGoals(ctx context.Context, subjectID string, observationCode string, observationValue float64) ([]goal.Goal, error)
}

type WSHub interface {
	BroadcastToPatient(patientID string, event interface{})
}

type GoalAchievedEvent struct {
	Type        string `json:"type"`
	GoalID      string `json:"goalId"`
	Description string `json:"description"`
}

type Service struct {
	repo          *Repository
	goalEvaluator GoalEvaluator
	wsHub         WSHub
}

func NewService(repo *Repository, evaluator GoalEvaluator, hub WSHub) *Service {
	return &Service{
		repo:          repo,
		goalEvaluator: evaluator,
		wsHub:         hub,
	}
}

func (s *Service) Create(ctx context.Context, o *Observation) (*Observation, error) {
	if err := validateObservation(o); err != nil {
		return nil, err
	}

	o.ResourceType = "Observation"
	o.Meta.LastUpdated = time.Now().UTC()

	created, err := s.repo.Create(ctx, o)
	if err != nil {
		return nil, err
	}

	s.evaluateAndBroadcast(ctx, created)

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Observation, error) {
	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, ErrNotFound
	}
	return o, nil
}

func (s *Service) Search(ctx context.Context, params map[string]string) ([]Observation, error) {
	return s.repo.Search(ctx, params)
}

func (s *Service) Update(ctx context.Context, id string, o *Observation) (*Observation, error) {
	if err := validateObservation(o); err != nil {
		return nil, err
	}

	o.Meta.LastUpdated = time.Now().UTC()

	result, err := s.repo.Update(ctx, id, o)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrNotFound
	}
	return result, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) evaluateAndBroadcast(ctx context.Context, o *Observation) {
	if o.ValueQuantity == nil || s.goalEvaluator == nil {
		return
	}

	subjectID := strings.TrimPrefix(o.Subject.Reference, "Patient/")

	code := ""
	if len(o.Code.Coding) > 0 {
		code = strings.TrimSpace(o.Code.Coding[0].Code)
	}
	if code == "" {
		return
	}

	achievedGoals, err := s.goalEvaluator.EvaluateGoals(ctx, subjectID, code, o.ValueQuantity.Value)
	if err != nil || s.wsHub == nil {
		return
	}

	for _, g := range achievedGoals {
		description := g.Description.Text
		if description == "" && len(g.Description.Coding) > 0 {
			description = g.Description.Coding[0].Display
		}

		s.wsHub.BroadcastToPatient(subjectID, GoalAchievedEvent{
			Type:        "goal.achieved",
			GoalID:      g.ID,
			Description: description,
		})
	}
}

func validateObservation(o *Observation) error {
	if !ValidStatuses[o.Status] {
		return ErrInvalidStatus
	}
	if len(o.Code.Coding) == 0 || o.Code.Coding[0].Code == "" {
		return ErrCodeRequired
	}
	return nil
}
