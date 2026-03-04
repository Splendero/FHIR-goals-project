package suggestion

import (
	"fmt"
	"strings"

	"github.com/spencerosborn/fhir-goals-engine/internal/observation"
)

const systemPrompt = `You are a clinical healthcare goal advisor. Given a patient summary and their recent observations, suggest personalized FHIR-compliant health goals.

Return ONLY a valid JSON array. Each element must have these fields:
- "description" (string): a clear, patient-friendly goal description
- "category" (string): one of "dietary", "physiotherapy", "physical-activity", "behavioral", "safety"
- "targetMeasure" (string): the LOINC display name for the target measure
- "targetValue" (float): the numeric target value
- "targetUnit" (string): the unit of measure (e.g., "kg", "mmHg", "%", "steps/day")
- "targetDueInDays" (int): suggested number of days to achieve the goal
- "priority" (string): one of "high-priority", "medium-priority", "low-priority"

Base suggestions on clinical guidelines. Prioritize goals that address the most critical observations first. Do not include explanations outside the JSON array.`

func BuildUserPrompt(patientSummary string, observations []observation.Observation) string {
	var b strings.Builder
	b.WriteString("Patient Summary:\n")
	b.WriteString(patientSummary)
	b.WriteString("\n\nRecent Observations:\n")

	for _, obs := range observations {
		display := obs.Code.Text
		if display == "" && len(obs.Code.Coding) > 0 {
			display = obs.Code.Coding[0].Display
		}
		if obs.ValueQuantity != nil {
			b.WriteString(fmt.Sprintf("- %s: %.2f %s (recorded %s)\n",
				display,
				obs.ValueQuantity.Value,
				obs.ValueQuantity.Unit,
				obs.EffectiveDateTime,
			))
		} else {
			b.WriteString(fmt.Sprintf("- %s (recorded %s)\n", display, obs.EffectiveDateTime))
		}
	}

	b.WriteString("\nBased on these observations, suggest appropriate health goals as a JSON array.")
	return b.String()
}
