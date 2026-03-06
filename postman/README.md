# Postman Collection — FHIR Goals Engine

This folder contains a ready-to-use Postman collection for testing every endpoint in the FHIR Goals Engine API.

## Importing the Collection

1. Open Postman
2. Click **Import** (top left)
3. Drag in `FHIR_Goals_Engine.postman_collection.json` or click **Upload Files** and select it
4. The collection **FHIR Goals Engine** will appear in your sidebar

## Collection Variables

The collection uses variables so you don't have to copy-paste IDs between requests. They are auto-populated by test scripts when you create resources.

| Variable | Description | Auto-set by |
|---|---|---|
| `baseUrl` | Server URL (default: `http://localhost:8080`) | Pre-configured |

To change the base URL, click the collection name in the sidebar, go to the **Variables** tab, and edit `baseUrl`.

## Request Folders

| # | Folder | What's Inside |
|---|---|---|
| 1 | **Patients** | Search, Create, Get, Update, Delete |
| 2 | **Goals** | Search by patient, Create with measurable target, Get, Update, Delete |
| 3 | **Observations** | Search, Create (triggers goal evaluation), Create (goal achievement trigger), Get, Delete |
| 4 | **AI Suggestions** | Generate AI-powered or rule-based goal suggestions |
| 5 | **Server Info** | FHIR CapabilityStatement (`/metadata`) |

## Recommended Workflow

Run these requests in order to see the full goal achievement flow:

1. **Create Patient** — creates a patient and saves their ID
2. **Create Goal** — creates a weight loss goal (target: 80 kg) for that patient
3. **Create Observation** — records a weight of 85 kg (above target, goal stays active)
4. **Create Observation (Goal Achievement Trigger)** — records 79 kg (below target)
5. **Get Goal by ID** — verify the goal's `lifecycleStatus` is now `completed` and `achievementStatus` is `achieved`
6. **Suggest Goals** — generate new AI/rule-based goal suggestions for the patient

If you have a WebSocket client connected (`ws://localhost:8080/ws?patient={patientId}`), you'll see a `goal.achieved` event pushed in real time at step 5.

## Test Scripts

Several requests include built-in test scripts that:

- Verify correct HTTP status codes (201, 200)
- Confirm the response contains the expected `resourceType`
- Auto-save returned IDs to collection variables for use in subsequent requests

These run automatically when you send a request. You can view results in the **Tests** tab of the response panel.

## Prerequisites

Make sure the server is running before sending requests:

```bash
docker compose up --build
make seed  # optional: populate sample data
```
