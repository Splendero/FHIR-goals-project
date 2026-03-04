package websocket

const (
	EventGoalAchieved   = "goal.achieved"
	EventGoalUpdated    = "goal.updated"
	EventGoalSuggested  = "goal.suggested"
	EventCarePlanUpdated = "careplan.updated"
)

type Event struct {
	Type      string      `json:"type"`
	PatientID string      `json:"patientId"`
	Data      interface{} `json:"data"`
}
