package suggestion

import (
	"fmt"
	"math"
	"time"

	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
	"github.com/spencerosborn/fhir-goals-engine/internal/goal"
	"github.com/spencerosborn/fhir-goals-engine/internal/observation"
)

const (
	loincBodyWeight = "29463-7"
	loincBPSystolic = "8480-6"
	loincHbA1c      = "4548-4"
	loincStepCount  = "41950-7"
)

func GenerateFallbackSuggestions(observations []observation.Observation) []goal.Goal {
	var goals []goal.Goal

	for _, obs := range observations {
		code := observationLOINC(obs)
		if obs.ValueQuantity == nil {
			continue
		}
		val := obs.ValueQuantity.Value

		switch code {
		case loincBodyWeight:
			if val > 90 {
				target := math.Round(val*0.9*10) / 10
				goals = append(goals, buildGoal(
					fmt.Sprintf("Reduce body weight from %.1f kg to %.1f kg", val, target),
					"dietary",
					"Body weight",
					loincBodyWeight,
					target,
					"kg",
					180,
					"medium-priority",
				))
			}
		case loincBPSystolic:
			if val > 140 {
				goals = append(goals, buildGoal(
					fmt.Sprintf("Lower systolic blood pressure from %.0f mmHg to 120 mmHg", val),
					"physiotherapy",
					"Systolic blood pressure",
					loincBPSystolic,
					120,
					"mmHg",
					90,
					"high-priority",
				))
			}
		case loincHbA1c:
			if val > 6.5 {
				goals = append(goals, buildGoal(
					fmt.Sprintf("Reduce HbA1c from %.1f%% to 6.0%%", val),
					"physiotherapy",
					"Hemoglobin A1c",
					loincHbA1c,
					6.0,
					"%",
					120,
					"high-priority",
				))
			}
		case loincStepCount:
			if val < 5000 {
				goals = append(goals, buildGoal(
					fmt.Sprintf("Increase daily steps from %.0f to 10000", val),
					"physical-activity",
					"Steps per day",
					loincStepCount,
					10000,
					"steps/day",
					60,
					"medium-priority",
				))
			}
		}
	}

	goals = append(goals, buildGoal(
		"Complete monthly mental health and wellness check-in",
		"behavioral",
		"Mental health wellness check",
		"",
		1,
		"{check-in}",
		30,
		"low-priority",
	))

	return goals
}

func observationLOINC(obs observation.Observation) string {
	for _, c := range obs.Code.Coding {
		if c.System == "http://loinc.org" {
			return c.Code
		}
	}
	if len(obs.Code.Coding) > 0 {
		return obs.Code.Coding[0].Code
	}
	return ""
}

func buildGoal(
	description, category, measureDisplay, measureCode string,
	targetValue float64, targetUnit string,
	dueDays int,
	priority string,
) goal.Goal {
	dueDate := time.Now().AddDate(0, 0, dueDays).Format("2006-01-02")

	g := goal.Goal{
		ResourceType:    "Goal",
		LifecycleStatus: goal.LifecycleProposed,
		Description: fhir.CodeableConcept{
			Text: description,
		},
		Category: []fhir.CodeableConcept{
			{
				Coding: []fhir.Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/goal-category",
						Code:    category,
						Display: category,
					},
				},
			},
		},
		Priority: &fhir.CodeableConcept{
			Coding: []fhir.Coding{
				{
					System:  "http://terminology.hl7.org/CodeSystem/goal-priority",
					Code:    priority,
					Display: priority,
				},
			},
		},
		Target: []goal.GoalTarget{
			{
				DueDate: dueDate,
				DetailQuantity: &fhir.Quantity{
					Value:  targetValue,
					Unit:   targetUnit,
					System: "http://unitsofmeasure.org",
				},
			},
		},
	}

	if measureCode != "" {
		g.Target[0].Measure = &fhir.CodeableConcept{
			Coding: []fhir.Coding{
				{
					System:  "http://loinc.org",
					Code:    measureCode,
					Display: measureDisplay,
				},
			},
		}
	}

	return g
}
