# FHIR Goals Engine

A FHIR R4-compliant health goals platform with AI-powered suggestions, real-time notifications, and automatic goal achievement evaluation.

![Go 1.22](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)
![FHIR R4](https://img.shields.io/badge/FHIR-R4%20(4.0.1)-E44D26)
![PostgreSQL 16](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![WebSocket](https://img.shields.io/badge/WebSocket-Real--time-010101)
![AI Powered](https://img.shields.io/badge/AI-OpenAI%20GPT--4o-412991?logo=openai&logoColor=white)

## Quick Start

```bash
git clone https://github.com/spencerosborn/fhir-goals-engine.git
cd fhir-goals-engine
docker compose up --build
```

The server starts at **http://localhost:8080** with an embedded dashboard at the root path.

Seed sample data (10 patients, 29 goals, 10 care plans, 70+ observations):

```bash
make seed
```

### Running without Docker

Requires Go 1.22+ and PostgreSQL 16+.

```bash
createdb fhir_goals
psql "postgres://fhir:fhir@localhost:5432/fhir_goals?sslmode=disable" -f migrations/001_create_tables.up.sql
make run
```

### Enabling AI Suggestions

Set `OPENAI_API_KEY` before starting. Without it, the suggestion endpoint uses a rule-based fallback.

```bash
OPENAI_API_KEY=sk-... docker compose up --build
```

## API Endpoints

Base URL: `http://localhost:8080` | Content-Type: `application/fhir+json`

| Resource | Endpoints | Search Params |
|---|---|---|
| **Patient** | `GET/POST /Patient`, `GET/PUT/DELETE /Patient/{id}` | `name`, `gender` |
| **Goal** | `GET/POST /Goal`, `GET/PUT/DELETE /Goal/{id}` | `patient`, `status`, `category` |
| **CarePlan** | `GET/POST /CarePlan`, `GET/PUT/DELETE /CarePlan/{id}` | `patient`, `status` |
| **Observation** | `GET/POST /Observation`, `GET/PUT/DELETE /Observation/{id}` | `patient`, `code`, `date` |
| **AI Suggest** | `POST /Goal/$suggest` | body: `{"patientId": "..."}` |
| **WebSocket** | `ws://localhost:8080/ws?patient={id}` | -- |
| **Metadata** | `GET /metadata` | -- |

Search endpoints return FHIR `Bundle` resources. Errors return `OperationOutcome` resources.

> See the [Postman collection](postman/) for ready-to-use requests with test scripts and auto-populated variables.

## How It Works

### Goal Achievement Engine

When an Observation is created, the engine automatically evaluates all active goals for that patient:

1. Matches the observation's LOINC code against goal targets
2. Compares the value directionally (weight/BP/HbA1c: achieved when value <= target; steps: achieved when value >= target)
3. Updates matched goals to `completed` / `achieved`
4. Broadcasts `goal.achieved` events via WebSocket to connected clients

| LOINC Code | Measure | Direction | Example |
|---|---|---|---|
| `29463-7` | Body weight | <= target | <= 75 kg |
| `8480-6` | Systolic BP | <= target | <= 120 mmHg |
| `4548-4` | HbA1c | <= target | <= 6.5% |
| `41950-7` | Steps/day | >= target | >= 10,000 |

### AI Suggestion Engine

- **With `OPENAI_API_KEY`**: GPT-4o generates personalized FHIR Goal resources from patient data and recent observations.
- **Without it**: A deterministic rule-based engine produces clinically reasonable suggestions based on established thresholds (elevated weight, hypertension, poor glycemic control, low activity).

All suggested goals are returned as a FHIR `Bundle` with `lifecycleStatus: proposed`.

## Project Structure

```
cmd/server/main.go          Entry point, router, middleware
internal/
  config/                    Environment-based configuration
  fhir/                      Shared FHIR R4 types (Bundle, OperationOutcome)
  patient/                   Patient handler, service, repository
  goal/                      Goal handler, service, repository, evaluator
  careplan/                  CarePlan handler, service, repository
  observation/               Observation handler, service, repository
  suggestion/                AI service, prompt builder, rule-based fallback
  websocket/                 Hub, client, event types
migrations/                  PostgreSQL schema (up/down)
static/index.html            Embedded dashboard
postman/                     Postman collection + docs
```

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22 |
| Router | chi v5 (logger, recoverer, CORS) |
| Database | PostgreSQL 16 |
| SQL | lib/pq + sqlx |
| WebSocket | gorilla/websocket |
| AI | go-openai (GPT-4o) |
| Container | Docker multi-stage, Docker Compose |
| Standard | HL7 FHIR R4, LOINC, UCUM |

## Testing

```bash
make test       # Run all tests
make lint       # Run golangci-lint
make build      # Compile to bin/fhir-goals-engine
```

## License

MIT
