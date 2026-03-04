package goal

import (
	"testing"

	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
	"github.com/stretchr/testify/assert"
)

func makeGoalWithTarget(measureCode string, targetValue float64, unit string) Goal {
	return Goal{
		ResourceType:    "Goal",
		ID:              "goal-1",
		LifecycleStatus: LifecycleActive,
		Description:     fhir.CodeableConcept{Text: "Test goal"},
		Subject:         fhir.Reference{Reference: "Patient/p-1"},
		Target: []GoalTarget{
			{
				Measure: &fhir.CodeableConcept{
					Coding: []fhir.Coding{{Code: measureCode}},
				},
				DetailQuantity: &fhir.Quantity{
					Value: targetValue,
					Unit:  unit,
				},
			},
		},
	}
}

func TestMeetsTarget(t *testing.T) {
	tests := []struct {
		name             string
		goal             Goal
		observationCode  string
		observationValue float64
		want             bool
	}{
		{
			name:             "weight decrease - observation meets target",
			goal:             makeGoalWithTarget("29463-7", 80, "kg"),
			observationCode:  "29463-7",
			observationValue: 79,
			want:             true,
		},
		{
			name:             "weight decrease - observation exactly at target",
			goal:             makeGoalWithTarget("29463-7", 80, "kg"),
			observationCode:  "29463-7",
			observationValue: 80,
			want:             true,
		},
		{
			name:             "weight decrease - observation above target",
			goal:             makeGoalWithTarget("29463-7", 80, "kg"),
			observationCode:  "29463-7",
			observationValue: 85,
			want:             false,
		},
		{
			name:             "steps increase - observation meets target",
			goal:             makeGoalWithTarget("41950-7", 10000, "steps/day"),
			observationCode:  "41950-7",
			observationValue: 12000,
			want:             true,
		},
		{
			name:             "steps increase - observation below target",
			goal:             makeGoalWithTarget("41950-7", 10000, "steps/day"),
			observationCode:  "41950-7",
			observationValue: 4000,
			want:             false,
		},
		{
			name:             "BP decrease - observation meets target",
			goal:             makeGoalWithTarget("8480-6", 120, "mmHg"),
			observationCode:  "8480-6",
			observationValue: 118,
			want:             true,
		},
		{
			name:             "BP decrease - observation above target",
			goal:             makeGoalWithTarget("8480-6", 120, "mmHg"),
			observationCode:  "8480-6",
			observationValue: 145,
			want:             false,
		},
		{
			name:             "HbA1c decrease - observation meets target",
			goal:             makeGoalWithTarget("4548-4", 6.0, "%"),
			observationCode:  "4548-4",
			observationValue: 5.8,
			want:             true,
		},
		{
			name: "no matching target code",
			goal: makeGoalWithTarget("29463-7", 80, "kg"),
			observationCode:  "41950-7",
			observationValue: 12000,
			want:             false,
		},
		{
			name: "target with nil measure - skip",
			goal: Goal{
				ResourceType:    "Goal",
				ID:              "goal-nil",
				LifecycleStatus: LifecycleActive,
				Target: []GoalTarget{
					{
						Measure:        nil,
						DetailQuantity: &fhir.Quantity{Value: 80, Unit: "kg"},
					},
				},
			},
			observationCode:  "29463-7",
			observationValue: 79,
			want:             false,
		},
		{
			name: "target with nil detail quantity - skip",
			goal: Goal{
				ResourceType:    "Goal",
				ID:              "goal-nil-qty",
				LifecycleStatus: LifecycleActive,
				Target: []GoalTarget{
					{
						Measure: &fhir.CodeableConcept{
							Coding: []fhir.Coding{{Code: "29463-7"}},
						},
						DetailQuantity: nil,
					},
				},
			},
			observationCode:  "29463-7",
			observationValue: 79,
			want:             false,
		},
		{
			name: "empty targets",
			goal: Goal{
				ResourceType:    "Goal",
				ID:              "goal-empty",
				LifecycleStatus: LifecycleActive,
				Target:          nil,
			},
			observationCode:  "29463-7",
			observationValue: 79,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := meetsTarget(tt.goal, tt.observationCode, tt.observationValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecreaseGoalCodes(t *testing.T) {
	assert.True(t, decreaseGoalCodes["29463-7"], "body weight should be a decrease goal")
	assert.True(t, decreaseGoalCodes["8480-6"], "systolic BP should be a decrease goal")
	assert.True(t, decreaseGoalCodes["4548-4"], "HbA1c should be a decrease goal")
	assert.False(t, decreaseGoalCodes["41950-7"], "step count should NOT be a decrease goal")
}
