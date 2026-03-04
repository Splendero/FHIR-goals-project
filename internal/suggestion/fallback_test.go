package suggestion

import (
	"testing"

	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
	"github.com/spencerosborn/fhir-goals-engine/internal/goal"
	"github.com/spencerosborn/fhir-goals-engine/internal/observation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeObs(code string, value float64, unit string) observation.Observation {
	return observation.Observation{
		ResourceType: "Observation",
		Status:       "final",
		Code: fhir.CodeableConcept{
			Coding: []fhir.Coding{
				{System: "http://loinc.org", Code: code},
			},
		},
		Subject:       fhir.Reference{Reference: "Patient/p-1"},
		ValueQuantity: &fhir.Quantity{Value: value, Unit: unit},
	}
}

func assertValidGoalStructure(t *testing.T, g goal.Goal) {
	t.Helper()
	assert.Equal(t, "Goal", g.ResourceType)
	assert.Equal(t, goal.LifecycleProposed, g.LifecycleStatus)
	assert.NotEmpty(t, g.Description.Text)
	require.NotEmpty(t, g.Target)
	assert.NotNil(t, g.Target[0].DetailQuantity)
	assert.NotEmpty(t, g.Target[0].DueDate)
	require.NotEmpty(t, g.Category)
	assert.NotEmpty(t, g.Category[0].Coding)
	require.NotNil(t, g.Priority)
	assert.NotEmpty(t, g.Priority.Coding)
}

func TestGenerateFallbackSuggestions_HighWeight(t *testing.T) {
	obs := []observation.Observation{makeObs("29463-7", 100, "kg")}
	goals := GenerateFallbackSuggestions(obs)

	// Should have weight goal + always-present mental health goal
	require.Len(t, goals, 2)

	weightGoal := goals[0]
	assertValidGoalStructure(t, weightGoal)
	assert.Contains(t, weightGoal.Description.Text, "Reduce body weight")
	assert.Contains(t, weightGoal.Description.Text, "100.0 kg")
	assert.Contains(t, weightGoal.Description.Text, "90.0 kg") // 90% of 100
	assert.Equal(t, "29463-7", weightGoal.Target[0].Measure.Coding[0].Code)
	assert.Equal(t, 90.0, weightGoal.Target[0].DetailQuantity.Value)
	assert.Equal(t, "kg", weightGoal.Target[0].DetailQuantity.Unit)
}

func TestGenerateFallbackSuggestions_NormalWeight(t *testing.T) {
	obs := []observation.Observation{makeObs("29463-7", 75, "kg")}
	goals := GenerateFallbackSuggestions(obs)

	// Only the mental health goal (75 kg is not > 90)
	require.Len(t, goals, 1)
	assert.Contains(t, goals[0].Description.Text, "mental health")
}

func TestGenerateFallbackSuggestions_HighBP(t *testing.T) {
	obs := []observation.Observation{makeObs("8480-6", 155, "mmHg")}
	goals := GenerateFallbackSuggestions(obs)

	require.Len(t, goals, 2)

	bpGoal := goals[0]
	assertValidGoalStructure(t, bpGoal)
	assert.Contains(t, bpGoal.Description.Text, "Lower systolic blood pressure")
	assert.Contains(t, bpGoal.Description.Text, "155 mmHg")
	assert.Equal(t, "8480-6", bpGoal.Target[0].Measure.Coding[0].Code)
	assert.Equal(t, 120.0, bpGoal.Target[0].DetailQuantity.Value)
	assert.Equal(t, "mmHg", bpGoal.Target[0].DetailQuantity.Unit)
}

func TestGenerateFallbackSuggestions_LowStepCount(t *testing.T) {
	obs := []observation.Observation{makeObs("41950-7", 3000, "steps/day")}
	goals := GenerateFallbackSuggestions(obs)

	require.Len(t, goals, 2)

	stepGoal := goals[0]
	assertValidGoalStructure(t, stepGoal)
	assert.Contains(t, stepGoal.Description.Text, "Increase daily steps")
	assert.Contains(t, stepGoal.Description.Text, "3000")
	assert.Equal(t, "41950-7", stepGoal.Target[0].Measure.Coding[0].Code)
	assert.Equal(t, 10000.0, stepGoal.Target[0].DetailQuantity.Value)
}

func TestGenerateFallbackSuggestions_HighStepCount(t *testing.T) {
	obs := []observation.Observation{makeObs("41950-7", 8000, "steps/day")}
	goals := GenerateFallbackSuggestions(obs)

	// 8000 is not < 5000, so only mental health goal
	require.Len(t, goals, 1)
	assert.Contains(t, goals[0].Description.Text, "mental health")
}

func TestGenerateFallbackSuggestions_NoObservations(t *testing.T) {
	goals := GenerateFallbackSuggestions(nil)

	require.Len(t, goals, 1)
	g := goals[0]
	assertValidGoalStructure(t, g)
	assert.Contains(t, g.Description.Text, "mental health")
	assert.Equal(t, "behavioral", g.Category[0].Coding[0].Code)
}

func TestGenerateFallbackSuggestions_ObservationWithoutValue(t *testing.T) {
	obs := []observation.Observation{
		{
			ResourceType: "Observation",
			Status:       "final",
			Code: fhir.CodeableConcept{
				Coding: []fhir.Coding{{System: "http://loinc.org", Code: "29463-7"}},
			},
			ValueQuantity: nil, // no value
		},
	}
	goals := GenerateFallbackSuggestions(obs)

	// Should skip the observation and only return mental health goal
	require.Len(t, goals, 1)
	assert.Contains(t, goals[0].Description.Text, "mental health")
}

func TestGenerateFallbackSuggestions_MultipleObservations(t *testing.T) {
	obs := []observation.Observation{
		makeObs("29463-7", 95, "kg"),  // triggers weight goal
		makeObs("8480-6", 150, "mmHg"), // triggers BP goal
		makeObs("41950-7", 2000, "steps/day"), // triggers step goal
	}
	goals := GenerateFallbackSuggestions(obs)

	// 3 condition-based goals + 1 mental health
	require.Len(t, goals, 4)
	assert.Contains(t, goals[0].Description.Text, "Reduce body weight")
	assert.Contains(t, goals[1].Description.Text, "Lower systolic blood pressure")
	assert.Contains(t, goals[2].Description.Text, "Increase daily steps")
	assert.Contains(t, goals[3].Description.Text, "mental health")
}

func TestGenerateFallbackSuggestions_HighHbA1c(t *testing.T) {
	obs := []observation.Observation{makeObs("4548-4", 7.5, "%")}
	goals := GenerateFallbackSuggestions(obs)

	require.Len(t, goals, 2)
	hba1cGoal := goals[0]
	assertValidGoalStructure(t, hba1cGoal)
	assert.Contains(t, hba1cGoal.Description.Text, "Reduce HbA1c")
	assert.Equal(t, "4548-4", hba1cGoal.Target[0].Measure.Coding[0].Code)
	assert.Equal(t, 6.0, hba1cGoal.Target[0].DetailQuantity.Value)
}
