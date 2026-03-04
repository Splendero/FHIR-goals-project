package observation

import (
	"context"
	"sync"
	"testing"

	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
	"github.com/spencerosborn/fhir-goals-engine/internal/goal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockGoalEvaluator struct {
	mu      sync.Mutex
	calls   []evaluateCall
	results []goal.Goal
	err     error
}

type evaluateCall struct {
	SubjectID string
	Code      string
	Value     float64
}

func (m *mockGoalEvaluator) EvaluateGoals(_ context.Context, subjectID, code string, value float64) ([]goal.Goal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, evaluateCall{SubjectID: subjectID, Code: code, Value: value})
	return m.results, m.err
}

type mockWSHub struct {
	mu     sync.Mutex
	events []broadcastCall
}

type broadcastCall struct {
	PatientID string
	Event     interface{}
}

func (m *mockWSHub) BroadcastToPatient(patientID string, event interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, broadcastCall{PatientID: patientID, Event: event})
}

// --- validateObservation tests ---

func TestValidateObservation(t *testing.T) {
	tests := []struct {
		name    string
		obs     Observation
		wantErr error
	}{
		{
			name: "valid observation",
			obs: Observation{
				Status: "final",
				Code:   fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "29463-7"}}},
			},
			wantErr: nil,
		},
		{
			name: "invalid status",
			obs: Observation{
				Status: "bogus",
				Code:   fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "29463-7"}}},
			},
			wantErr: ErrInvalidStatus,
		},
		{
			name: "empty status",
			obs: Observation{
				Status: "",
				Code:   fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "29463-7"}}},
			},
			wantErr: ErrInvalidStatus,
		},
		{
			name: "missing code - no coding slice",
			obs: Observation{
				Status: "final",
				Code:   fhir.CodeableConcept{Coding: nil},
			},
			wantErr: ErrCodeRequired,
		},
		{
			name: "missing code - empty coding slice",
			obs: Observation{
				Status: "final",
				Code:   fhir.CodeableConcept{Coding: []fhir.Coding{}},
			},
			wantErr: ErrCodeRequired,
		},
		{
			name: "missing code - coding with empty code string",
			obs: Observation{
				Status: "final",
				Code:   fhir.CodeableConcept{Coding: []fhir.Coding{{Code: ""}}},
			},
			wantErr: ErrCodeRequired,
		},
		{
			name: "all valid statuses accepted - registered",
			obs: Observation{
				Status: "registered",
				Code:   fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "X"}}},
			},
			wantErr: nil,
		},
		{
			name: "all valid statuses accepted - preliminary",
			obs: Observation{
				Status: "preliminary",
				Code:   fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "X"}}},
			},
			wantErr: nil,
		},
		{
			name: "all valid statuses accepted - amended",
			obs: Observation{
				Status: "amended",
				Code:   fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "X"}}},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateObservation(&tt.obs)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- evaluateAndBroadcast tests ---

func TestEvaluateAndBroadcast_WithValueQuantity(t *testing.T) {
	achieved := []goal.Goal{
		{
			ID:          "g-1",
			Description: fhir.CodeableConcept{Text: "Lose weight"},
		},
	}
	evaluator := &mockGoalEvaluator{results: achieved}
	hub := &mockWSHub{}

	svc := &Service{
		repo:          nil,
		goalEvaluator: evaluator,
		wsHub:         hub,
	}

	obs := &Observation{
		Subject: fhir.Reference{Reference: "Patient/p-42"},
		Code: fhir.CodeableConcept{
			Coding: []fhir.Coding{{Code: "29463-7"}},
		},
		ValueQuantity: &fhir.Quantity{Value: 79.0, Unit: "kg"},
	}

	svc.evaluateAndBroadcast(context.Background(), obs)

	require.Len(t, evaluator.calls, 1)
	assert.Equal(t, "p-42", evaluator.calls[0].SubjectID)
	assert.Equal(t, "29463-7", evaluator.calls[0].Code)
	assert.Equal(t, 79.0, evaluator.calls[0].Value)

	require.Len(t, hub.events, 1)
	assert.Equal(t, "p-42", hub.events[0].PatientID)
	evt, ok := hub.events[0].Event.(GoalAchievedEvent)
	require.True(t, ok)
	assert.Equal(t, "goal.achieved", evt.Type)
	assert.Equal(t, "g-1", evt.GoalID)
	assert.Equal(t, "Lose weight", evt.Description)
}

func TestEvaluateAndBroadcast_NilValueQuantity(t *testing.T) {
	evaluator := &mockGoalEvaluator{}
	hub := &mockWSHub{}

	svc := &Service{
		repo:          nil,
		goalEvaluator: evaluator,
		wsHub:         hub,
	}

	obs := &Observation{
		Subject:       fhir.Reference{Reference: "Patient/p-1"},
		Code:          fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "29463-7"}}},
		ValueQuantity: nil,
	}

	svc.evaluateAndBroadcast(context.Background(), obs)

	assert.Empty(t, evaluator.calls, "should not call evaluator when ValueQuantity is nil")
	assert.Empty(t, hub.events, "should not broadcast when ValueQuantity is nil")
}

func TestEvaluateAndBroadcast_NilEvaluator(t *testing.T) {
	hub := &mockWSHub{}

	svc := &Service{
		repo:          nil,
		goalEvaluator: nil,
		wsHub:         hub,
	}

	obs := &Observation{
		Subject:       fhir.Reference{Reference: "Patient/p-1"},
		Code:          fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "29463-7"}}},
		ValueQuantity: &fhir.Quantity{Value: 79.0},
	}

	svc.evaluateAndBroadcast(context.Background(), obs)

	assert.Empty(t, hub.events, "should not broadcast when evaluator is nil")
}

func TestEvaluateAndBroadcast_NilWSHub(t *testing.T) {
	achieved := []goal.Goal{
		{ID: "g-1", Description: fhir.CodeableConcept{Text: "Test"}},
	}
	evaluator := &mockGoalEvaluator{results: achieved}

	svc := &Service{
		repo:          nil,
		goalEvaluator: evaluator,
		wsHub:         nil,
	}

	obs := &Observation{
		Subject:       fhir.Reference{Reference: "Patient/p-1"},
		Code:          fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "29463-7"}}},
		ValueQuantity: &fhir.Quantity{Value: 79.0},
	}

	// Should not panic even with nil hub
	svc.evaluateAndBroadcast(context.Background(), obs)
	require.Len(t, evaluator.calls, 1)
}

func TestEvaluateAndBroadcast_NoAchievedGoals(t *testing.T) {
	evaluator := &mockGoalEvaluator{results: nil}
	hub := &mockWSHub{}

	svc := &Service{
		repo:          nil,
		goalEvaluator: evaluator,
		wsHub:         hub,
	}

	obs := &Observation{
		Subject:       fhir.Reference{Reference: "Patient/p-1"},
		Code:          fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "29463-7"}}},
		ValueQuantity: &fhir.Quantity{Value: 100.0},
	}

	svc.evaluateAndBroadcast(context.Background(), obs)

	require.Len(t, evaluator.calls, 1)
	assert.Empty(t, hub.events, "no broadcast when no goals achieved")
}

func TestEvaluateAndBroadcast_DescriptionFallbackToCoding(t *testing.T) {
	achieved := []goal.Goal{
		{
			ID: "g-2",
			Description: fhir.CodeableConcept{
				Text:   "",
				Coding: []fhir.Coding{{Display: "Coding display text"}},
			},
		},
	}
	evaluator := &mockGoalEvaluator{results: achieved}
	hub := &mockWSHub{}

	svc := &Service{
		repo:          nil,
		goalEvaluator: evaluator,
		wsHub:         hub,
	}

	obs := &Observation{
		Subject:       fhir.Reference{Reference: "Patient/p-1"},
		Code:          fhir.CodeableConcept{Coding: []fhir.Coding{{Code: "29463-7"}}},
		ValueQuantity: &fhir.Quantity{Value: 79.0},
	}

	svc.evaluateAndBroadcast(context.Background(), obs)

	require.Len(t, hub.events, 1)
	evt := hub.events[0].Event.(GoalAchievedEvent)
	assert.Equal(t, "Coding display text", evt.Description)
}
