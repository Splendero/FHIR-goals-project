package goal

import (
	"context"
	"fmt"

	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
)

type Evaluator struct {
	repo *Repository
}

func NewEvaluator(repo *Repository) *Evaluator {
	return &Evaluator{repo: repo}
}

func (e *Evaluator) EvaluateGoals(ctx context.Context, subjectID, observationCode string, observationValue float64) ([]Goal, error) {
	goals, err := e.repo.GetBySubjectAndStatus(ctx, subjectID, LifecycleActive)
	if err != nil {
		return nil, fmt.Errorf("fetch active goals: %w", err)
	}

	var achieved []Goal
	for _, g := range goals {
		if !meetsTarget(g, observationCode, observationValue) {
			continue
		}

		g.LifecycleStatus = LifecycleCompleted
		g.AchievementStatus = &fhir.CodeableConcept{
			Coding: []fhir.Coding{{Code: AchievementAchieved}},
			Text:   "Achieved",
		}

		updated, err := e.repo.Update(ctx, g.ID, &g)
		if err != nil {
			return nil, fmt.Errorf("update achieved goal %s: %w", g.ID, err)
		}
		achieved = append(achieved, *updated)
	}

	return achieved, nil
}

var decreaseGoalCodes = map[string]bool{
	"29463-7": true, // body weight
	"8480-6":  true, // systolic BP
	"4548-4":  true, // HbA1c
}

func meetsTarget(g Goal, observationCode string, observationValue float64) bool {
	for _, t := range g.Target {
		if t.Measure == nil || t.DetailQuantity == nil {
			continue
		}
		for _, c := range t.Measure.Coding {
			if c.Code != observationCode {
				continue
			}
			target := t.DetailQuantity.Value
			if decreaseGoalCodes[observationCode] {
				return observationValue <= target
			}
			return observationValue >= target
		}
	}
	return false
}
