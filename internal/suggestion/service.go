package suggestion

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
	"github.com/spencerosborn/fhir-goals-engine/internal/goal"
	"github.com/spencerosborn/fhir-goals-engine/internal/observation"
	"github.com/spencerosborn/fhir-goals-engine/internal/patient"
)

type ObservationRepo interface {
	Search(ctx context.Context, params map[string]string) ([]observation.Observation, error)
}

type PatientRepo interface {
	GetByID(ctx context.Context, id string) (*patient.Patient, error)
}

type Service struct {
	openaiKey       string
	observationRepo ObservationRepo
	patientRepo     PatientRepo
}

func NewService(openaiKey string, obsRepo ObservationRepo, patRepo PatientRepo) *Service {
	return &Service{
		openaiKey:       openaiKey,
		observationRepo: obsRepo,
		patientRepo:     patRepo,
	}
}

func (s *Service) SuggestGoals(ctx context.Context, patientID string) ([]goal.Goal, error) {
	pat, err := s.patientRepo.GetByID(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("fetch patient: %w", err)
	}

	obs, err := s.observationRepo.Search(ctx, map[string]string{
		"patient": patientID,
		"_sort":   "-date",
		"_count":  "20",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch observations: %w", err)
	}

	if s.openaiKey != "" {
		goals, err := s.suggestViaLLM(ctx, pat, obs)
		if err == nil {
			return goals, nil
		}
	}

	return GenerateFallbackSuggestions(obs), nil
}

func (s *Service) suggestViaLLM(ctx context.Context, pat *patient.Patient, obs []observation.Observation) ([]goal.Goal, error) {
	client := openai.NewClient(s.openaiKey)

	summary := buildPatientSummary(pat)
	userPrompt := BuildUserPrompt(summary, obs)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	return parseLLMResponse(resp.Choices[0].Message.Content, pat.ID)
}

type llmSuggestion struct {
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	TargetMeasure   string  `json:"targetMeasure"`
	TargetValue     float64 `json:"targetValue"`
	TargetUnit      string  `json:"targetUnit"`
	TargetDueInDays int     `json:"targetDueInDays"`
	Priority        string  `json:"priority"`
}

func parseLLMResponse(raw string, patientID string) ([]goal.Goal, error) {
	raw = strings.TrimSpace(raw)
	if start := strings.Index(raw, "["); start >= 0 {
		if end := strings.LastIndex(raw, "]"); end >= start {
			raw = raw[start : end+1]
		}
	}

	var suggestions []llmSuggestion
	if err := json.Unmarshal([]byte(raw), &suggestions); err != nil {
		return nil, fmt.Errorf("parse llm json: %w", err)
	}

	goals := make([]goal.Goal, 0, len(suggestions))
	for _, s := range suggestions {
		dueDate := time.Now().AddDate(0, 0, s.TargetDueInDays).Format("2006-01-02")

		g := goal.Goal{
			ResourceType:    "Goal",
			LifecycleStatus: goal.LifecycleProposed,
			Description: fhir.CodeableConcept{
				Text: s.Description,
			},
			Subject: fhir.Reference{
				Reference: "Patient/" + patientID,
			},
			Category: []fhir.CodeableConcept{
				{
					Coding: []fhir.Coding{
						{
							System:  "http://terminology.hl7.org/CodeSystem/goal-category",
							Code:    s.Category,
							Display: s.Category,
						},
					},
				},
			},
			Priority: &fhir.CodeableConcept{
				Coding: []fhir.Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/goal-priority",
						Code:    s.Priority,
						Display: s.Priority,
					},
				},
			},
			Target: []goal.GoalTarget{
				{
					Measure: &fhir.CodeableConcept{
						Text: s.TargetMeasure,
					},
					DetailQuantity: &fhir.Quantity{
						Value:  s.TargetValue,
						Unit:   s.TargetUnit,
						System: "http://unitsofmeasure.org",
					},
					DueDate: dueDate,
				},
			},
		}
		goals = append(goals, g)
	}

	return goals, nil
}

func buildPatientSummary(pat *patient.Patient) string {
	var b strings.Builder
	if len(pat.Name) > 0 {
		name := pat.Name[0]
		if len(name.Given) > 0 {
			b.WriteString(strings.Join(name.Given, " "))
			b.WriteString(" ")
		}
		b.WriteString(name.Family)
	}
	if pat.Gender != "" {
		b.WriteString(", Gender: ")
		b.WriteString(pat.Gender)
	}
	if pat.BirthDate != "" {
		b.WriteString(", DOB: ")
		b.WriteString(pat.BirthDate)
	}
	return b.String()
}
