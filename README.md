# FHIR Goals Engine

**A FHIR R4-compliant health goals tracking platform with AI-powered suggestions, real-time notifications, and automatic goal achievement evaluation.**

![Go 1.22](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)
![FHIR R4](https://img.shields.io/badge/FHIR-R4%20(4.0.1)-E44D26)
![PostgreSQL 16](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![WebSocket](https://img.shields.io/badge/WebSocket-Real--time-010101)
![AI Powered](https://img.shields.io/badge/AI-OpenAI%20GPT--4o-412991?logo=openai&logoColor=white)

---

## Overview

FHIR Goals Engine is a healthcare interoperability platform that implements the HL7 FHIR R4 specification for managing patient health goals, care plans, and clinical observations. It features an event-driven goal achievement engine that automatically evaluates incoming observations against active goal targets, an AI-powered suggestion service that generates personalized FHIR-compliant goals from patient data, and a real-time WebSocket layer that pushes achievement notifications to connected clients.

---

## Architecture

```mermaid
graph TB
    subgraph Client Layer
        UI[Embedded Dashboard]
        WS_Client[WebSocket Client]
        API_Client[REST Client]
    end

    subgraph HTTP Layer
        Router[Chi Router<br/>middleware: logger, recoverer, CORS]
    end

    subgraph Handlers
        PH[Patient Handler]
        GH[Goal Handler]
        CPH[CarePlan Handler]
        OH[Observation Handler]
        SH[Suggest Handler]
        CAP[/metadata<br/>CapabilityStatement/]
    end

    subgraph Services
        PS[Patient Service]
        GS[Goal Service]
        CPS[CarePlan Service]
        OS[Observation Service]
        SS[Suggestion Service]
    end

    subgraph Core Engine
        GE[Goal Evaluator]
        WSHub[WebSocket Hub]
    end

    subgraph Repositories
        PR[Patient Repo]
        GR[Goal Repo]
        CPR[CarePlan Repo]
        OR[Observation Repo]
    end

    DB[(PostgreSQL 16)]
    OpenAI[OpenAI API<br/>GPT-4o]
    Fallback[Rule-Based<br/>Fallback Engine]

    UI --> Router
    WS_Client --> WSHub
    API_Client --> Router

    Router --> PH & GH & CPH & OH & SH & CAP

    PH --> PS
    GH --> GS
    CPH --> CPS
    OH --> OS
    SH --> SS

    PS --> PR
    GS --> GR
    CPS --> CPR
    OS --> OR

    PR & GR & CPR & OR --> DB

    OS -- "on create" --> GE
    GE --> GR
    GE -- "achieved goals" --> WSHub
    WSHub -- "push events" --> WS_Client

    SS --> OpenAI
    SS -. "fallback" .-> Fallback
    SS --> OR
    SS --> PR
```

---

## FHIR Resources

This engine implements full CRUD and search operations for the following FHIR R4 resources:

| Resource | Description |
|---|---|
| **Patient** | Demographics, identifiers, and administrative information. Serves as the subject reference for all clinical resources. |
| **Goal** | Health objectives with measurable targets, lifecycle tracking (`proposed` through `completed`), and achievement status. Supports LOINC-coded target measures. |
| **CarePlan** | Coordinated care plans linking patients to their goals with status, intent, time periods, and SNOMED-coded categories. |
| **Observation** | Clinical measurements (vitals, lab results, activity data) that drive the goal evaluation engine. LOINC-coded with quantity values. |

All responses use the `application/fhir+json` content type. Search endpoints return FHIR `Bundle` resources of type `searchset`. Errors are returned as `OperationOutcome` resources.

---

## Key Features

- **FHIR R4-compliant REST API** -- Full CRUD on Patient, Goal, CarePlan, and Observation with proper Bundle responses, OperationOutcome errors, and a CapabilityStatement at `/metadata`.
- **AI-powered goal suggestions** -- OpenAI GPT-4o integration generates personalized, clinically-informed FHIR Goal resources from patient observations, with a deterministic rule-based fallback when no API key is configured.
- **Automatic goal achievement evaluation** -- Every new Observation triggers evaluation of the patient's active goals. When a measurement meets or exceeds a target, the goal is automatically transitioned to `completed` / `achieved`.
- **Real-time WebSocket notifications** -- Connected clients receive `goal.achieved` events instantly when a patient's goal is met, enabling reactive UIs and downstream integrations.
- **Embedded dashboard** -- A built-in web interface served at the root path for visualizing patients, goals, and observations without external tooling.
- **Docker Compose one-command setup** -- A single `docker compose up --build` brings up PostgreSQL 16, runs migrations, and starts the server.

---

## Quick Start

### Using Docker (recommended)

```bash
git clone https://github.com/spencerosborn/fhir-goals-engine.git
cd fhir-goals-engine
docker compose up --build
```

The server starts at [http://localhost:8080](http://localhost:8080). The dashboard is served at the root path.

### Seeding sample data

With the database running, populate it with 10 patients, 29 goals, 10 care plans, and 70+ observations:

```bash
make seed
```

### Running locally (without Docker)

Prerequisites: Go 1.22+, PostgreSQL 16+.

```bash
# Start PostgreSQL and create the database
createdb fhir_goals

# Run migrations
psql "postgres://fhir:fhir@localhost:5432/fhir_goals?sslmode=disable" \
  -f migrations/001_create_tables.up.sql

# Start the server
make run
```

### Enabling AI suggestions

Set the `OPENAI_API_KEY` environment variable before starting the server. Without it, the suggestion endpoint falls back to the rule-based engine.

```bash
export OPENAI_API_KEY=sk-...
make run
```

Or via Docker Compose:

```bash
OPENAI_API_KEY=sk-... docker compose up --build
```

---

## API Reference

Base URL: `http://localhost:8080`

All resource endpoints follow the FHIR RESTful API pattern. Responses use `Content-Type: application/fhir+json`.

### Patient

```bash
# Create a patient
curl -X POST http://localhost:8080/Patient \
  -H "Content-Type: application/json" \
  -d '{
    "resourceType": "Patient",
    "active": true,
    "name": [{"use": "official", "family": "Garcia", "given": ["Maria"]}],
    "gender": "female",
    "birthDate": "1985-03-15"
  }'

# Search patients
curl http://localhost:8080/Patient

# Get a patient by ID
curl http://localhost:8080/Patient/{id}

# Update a patient
curl -X PUT http://localhost:8080/Patient/{id} \
  -H "Content-Type: application/json" \
  -d '{"resourceType": "Patient", "active": true, "name": [{"family": "Garcia", "given": ["Maria", "Elena"]}]}'

# Delete a patient
curl -X DELETE http://localhost:8080/Patient/{id}
```

### Goal

```bash
# Create a goal with a measurable target
curl -X POST http://localhost:8080/Goal \
  -H "Content-Type: application/json" \
  -d '{
    "resourceType": "Goal",
    "lifecycleStatus": "active",
    "description": {"text": "Reduce body weight to 75 kg"},
    "subject": {"reference": "Patient/{patient-id}"},
    "target": [{
      "measure": {"coding": [{"system": "http://loinc.org", "code": "29463-7", "display": "Body weight"}]},
      "detailQuantity": {"value": 75, "unit": "kg", "system": "http://unitsofmeasure.org"},
      "dueDate": "2026-06-01"
    }]
  }'

# Search goals (supports ?patient=, ?status=, ?category= parameters)
curl "http://localhost:8080/Goal?patient={patient-id}&status=active"

# Get a goal by ID
curl http://localhost:8080/Goal/{id}

# Update a goal
curl -X PUT http://localhost:8080/Goal/{id} \
  -H "Content-Type: application/json" \
  -d '{"resourceType": "Goal", "lifecycleStatus": "completed", "description": {"text": "Reduce body weight to 75 kg"}, "subject": {"reference": "Patient/{patient-id}"}}'

# Delete a goal
curl -X DELETE http://localhost:8080/Goal/{id}
```

### CarePlan

```bash
# Create a care plan linking goals
curl -X POST http://localhost:8080/CarePlan \
  -H "Content-Type: application/json" \
  -d '{
    "resourceType": "CarePlan",
    "status": "active",
    "intent": "plan",
    "title": "Diabetes Management Plan",
    "subject": {"reference": "Patient/{patient-id}"},
    "goal": [{"reference": "Goal/{goal-id}"}],
    "period": {"start": "2026-01-01", "end": "2026-06-30"}
  }'

# Search care plans
curl http://localhost:8080/CarePlan

# Get, update, delete
curl http://localhost:8080/CarePlan/{id}
curl -X PUT http://localhost:8080/CarePlan/{id} -H "Content-Type: application/json" -d '{...}'
curl -X DELETE http://localhost:8080/CarePlan/{id}
```

### Observation

```bash
# Create an observation (triggers goal evaluation)
curl -X POST http://localhost:8080/Observation \
  -H "Content-Type: application/json" \
  -d '{
    "resourceType": "Observation",
    "status": "final",
    "code": {
      "coding": [{"system": "http://loinc.org", "code": "29463-7", "display": "Body weight"}]
    },
    "subject": {"reference": "Patient/{patient-id}"},
    "effectiveDateTime": "2026-03-04T10:00:00Z",
    "valueQuantity": {"value": 74.5, "unit": "kg", "system": "http://unitsofmeasure.org", "code": "kg"}
  }'

# Search observations (supports ?patient=, ?code=, ?_sort=, ?_count= parameters)
curl "http://localhost:8080/Observation?patient={patient-id}&code=29463-7"

# Get, update, delete
curl http://localhost:8080/Observation/{id}
curl -X PUT http://localhost:8080/Observation/{id} -H "Content-Type: application/json" -d '{...}'
curl -X DELETE http://localhost:8080/Observation/{id}
```

### AI Goal Suggestions

```bash
# Generate AI-powered goal suggestions for a patient
curl -X POST http://localhost:8080/Goal/\$suggest \
  -H "Content-Type: application/json" \
  -d '{"patientId": "{patient-id}"}'
```

Returns a FHIR `Bundle` of type `collection` containing proposed `Goal` resources.

### WebSocket (Real-time Notifications)

```bash
# Connect to the WebSocket for a specific patient
wscat -c "ws://localhost:8080/ws?patient={patient-id}"
```

Events are pushed as JSON with the following structure:

```json
{
  "type": "goal.achieved",
  "patientId": "abc-123",
  "data": {
    "type": "goal.achieved",
    "goalId": "goal-456",
    "description": "Reduce body weight to 75 kg"
  }
}
```

### CapabilityStatement

```bash
# Retrieve the server's FHIR CapabilityStatement
curl http://localhost:8080/metadata
```

---

## Goal Achievement Engine

The achievement engine is the core of the system. It implements an event-driven evaluation loop triggered by every new Observation:

```
POST /Observation
    |
    v
Observation Service: persist to database
    |
    v
Goal Evaluator: fetch all active goals for the patient
    |
    v
For each active goal with a matching target measure:
    |
    +-- Compare observation value against target
    |   - Decrease goals (weight, BP, HbA1c): achieved if value <= target
    |   - Increase goals (steps):             achieved if value >= target
    |
    +-- If met: update lifecycle to "completed", achievement to "achieved"
    |
    v
WebSocket Hub: broadcast "goal.achieved" to connected clients for the patient
```

### Supported LOINC Codes

| LOINC Code | Measure | Direction | Example Target |
|---|---|---|---|
| `29463-7` | Body weight | Decrease | <= 75 kg |
| `8480-6` | Systolic blood pressure | Decrease | <= 120 mmHg |
| `4548-4` | Hemoglobin A1c (HbA1c) | Decrease | <= 6.5% |
| `41950-7` | Number of steps per day | Increase | >= 10,000 steps |

The evaluator uses directional logic: goals targeting weight, blood pressure, and HbA1c are met when the observed value drops to or below the target. Step count goals are met when the observed value reaches or exceeds the target.

---

## AI Suggestion Engine

The suggestion service implements a dual-strategy approach for generating personalized health goals.

### Primary: OpenAI GPT-4o

When `OPENAI_API_KEY` is configured, the service constructs a structured prompt containing the patient summary (name, gender, date of birth) and their most recent observations (up to 20, sorted by date descending). GPT-4o returns a JSON array of goal suggestions with descriptions, categories, measurable targets, units, due dates, and priority levels. Each suggestion is mapped to a fully FHIR-compliant `Goal` resource with `lifecycleStatus: proposed`.

### Fallback: Rule-Based Engine

When the API key is absent or the OpenAI call fails, the engine falls back to deterministic clinical rules:

| Condition | Trigger | Suggested Goal |
|---|---|---|
| Elevated body weight | > 90 kg | Reduce weight by 10%, due in 180 days |
| Hypertension | Systolic BP > 140 mmHg | Lower to 120 mmHg, due in 90 days (high priority) |
| Poor glycemic control | HbA1c > 6.5% | Reduce to 6.0%, due in 120 days (high priority) |
| Low physical activity | < 5,000 steps/day | Increase to 10,000 steps/day, due in 60 days |
| Baseline | Always | Monthly mental health and wellness check-in |

All generated goals use proper FHIR value sets for category (`http://terminology.hl7.org/CodeSystem/goal-category`) and priority (`http://terminology.hl7.org/CodeSystem/goal-priority`).

---

## Project Structure

```
fhir-goals-engine/
|-- cmd/
|   +-- server/
|       +-- main.go                 # Application entry point, router setup, middleware
|-- internal/
|   |-- config/
|   |   +-- config.go               # Environment-based configuration
|   |-- fhir/
|   |   |-- types.go                # Shared FHIR R4 data types (Coding, Reference, Quantity, etc.)
|   |   +-- bundle.go               # Bundle, BundleEntry, OperationOutcome
|   |-- patient/
|   |   |-- model.go                # Patient resource model
|   |   |-- handler.go              # HTTP handlers (CRUD + search)
|   |   |-- service.go              # Business logic
|   |   +-- repository.go           # PostgreSQL persistence
|   |-- goal/
|   |   |-- model.go                # Goal resource model with lifecycle/achievement constants
|   |   |-- handler.go              # HTTP handlers
|   |   |-- service.go              # Business logic
|   |   |-- repository.go           # PostgreSQL persistence
|   |   +-- evaluator.go            # Goal achievement evaluation engine
|   |-- careplan/
|   |   |-- model.go                # CarePlan resource model
|   |   |-- handler.go              # HTTP handlers
|   |   |-- service.go              # Business logic
|   |   +-- repository.go           # PostgreSQL persistence
|   |-- observation/
|   |   |-- model.go                # Observation resource model
|   |   |-- handler.go              # HTTP handlers
|   |   |-- service.go              # Business logic with post-create evaluation hook
|   |   +-- repository.go           # PostgreSQL persistence
|   |-- suggestion/
|   |   |-- service.go              # AI suggestion orchestrator (OpenAI + fallback)
|   |   |-- prompt.go               # LLM system/user prompt construction
|   |   +-- fallback.go             # Rule-based suggestion engine
|   +-- websocket/
|       |-- hub.go                  # Connection hub, patient-scoped broadcasting
|       |-- client.go               # WebSocket client read/write pumps
|       +-- events.go               # Event type definitions
|-- migrations/
|   |-- 001_create_tables.up.sql    # Schema: patients, goals, care_plans, observations
|   +-- 001_create_tables.down.sql  # Rollback migration
|-- static/
|   +-- index.html                  # Embedded dashboard
|-- seed.go                         # Sample data: 10 patients, 29 goals, 10 care plans, 70+ observations
|-- Dockerfile                      # Multi-stage build (golang:1.22-alpine -> alpine:3.19)
|-- docker-compose.yml              # PostgreSQL 16 + app with health checks
|-- Makefile                        # Build, run, test, seed, docker, lint targets
|-- go.mod
+-- go.sum
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22 |
| HTTP Router | [chi v5](https://github.com/go-chi/chi) with middleware (logger, recoverer, request ID, CORS) |
| Database | PostgreSQL 16 (Alpine) |
| SQL Driver | [lib/pq](https://github.com/lib/pq) + [sqlx](https://github.com/jmoiron/sqlx) |
| WebSocket | [gorilla/websocket](https://github.com/gorilla/websocket) |
| AI | [go-openai](https://github.com/sashabaranov/go-openai) (GPT-4o) |
| UUIDs | [google/uuid](https://github.com/google/uuid) |
| Container | Docker multi-stage build, Docker Compose |
| Standard | HL7 FHIR R4 (4.0.1), LOINC, UCUM |

---

## Testing

```bash
make test
```

Runs all tests with verbose output and no caching:

```bash
go test ./... -v -count=1
```

Additional targets:

```bash
make lint      # Run golangci-lint
make build     # Compile to bin/fhir-goals-engine
```

---

## Design Decisions

**Chi over Gin.** Chi is lightweight, fully compatible with the `net/http` standard library, and composes middleware naturally. Handlers are plain `http.HandlerFunc` signatures, which avoids framework lock-in and makes the codebase accessible to any Go developer.

**sqlx over GORM.** This project uses explicit SQL queries via sqlx rather than an ORM. In a healthcare context, SQL transparency matters: every query is auditable, there are no hidden N+1 problems, and the mapping between FHIR resource fields and relational columns is intentional and visible.

**Structured columns over JSONB.** Rather than storing entire FHIR resources as JSONB blobs, each resource is decomposed into typed, indexed columns (e.g., `lifecycle_status`, `target_measure_code`, `subject_id`). This enables efficient queries (`WHERE lifecycle_status = 'active' AND subject_id = $1`), referential integrity via foreign keys, and proper indexing -- while the handler layer reconstructs FHIR-compliant JSON for API responses.

**Clean architecture with handler/service/repository layers.** Each FHIR resource follows the same layered pattern. Handlers own HTTP concerns (parsing, status codes, content types). Services contain business logic and validation. Repositories encapsulate SQL. Dependencies flow inward and are injected at startup in `main.go`, making each layer independently testable.

**Event-driven goal evaluation.** Rather than polling or batch-processing, goal achievement is evaluated synchronously within the Observation create path. This guarantees that every clinically relevant observation is immediately assessed against active goals, and connected clients are notified in real time via the WebSocket hub.

**AI with deterministic fallback.** The suggestion service gracefully degrades. When OpenAI is available, it provides nuanced, patient-specific recommendations. When it is not, the rule-based engine ensures the endpoint always returns clinically reasonable suggestions based on established thresholds, with no external dependency required.

---

## License

MIT
