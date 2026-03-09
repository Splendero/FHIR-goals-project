package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// Prefer DATABASE_URL (Railway, etc.) when set
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			env("DB_HOST", "localhost"),
			env("DB_PORT", "5432"),
			env("DB_USER", "fhir"),
			env("DB_PASSWORD", "fhir"),
			env("DB_NAME", "fhir_goals"),
		)
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	// Idempotent: skip if already seeded (safe for Railway redeploys)
	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM patients"); err == nil && count > 0 {
		fmt.Println("Already seeded, skipping")
		return
	}

	now := time.Now()
	ago := func(days int) time.Time { return now.AddDate(0, 0, -days) }

	// ── Patients ──

	type patient struct {
		id         uuid.UUID
		family     string
		given      string
		gender     string
		birthDate  string
		idSystem   string
		idValue    string
	}

	patients := []patient{
		{uuid.New(), "Garcia", "Maria", "female", "1985-03-15", "http://hospital.example/mrn", "MRN-10001"},
		{uuid.New(), "Wilson", "James", "male", "1972-08-22", "http://hospital.example/mrn", "MRN-10002"},
		{uuid.New(), "Chen", "Sarah", "female", "1990-11-03", "http://hospital.example/mrn", "MRN-10003"},
		{uuid.New(), "Johnson", "Robert", "male", "1968-05-10", "http://hospital.example/mrn", "MRN-10004"},
		{uuid.New(), "Thompson", "Emily", "female", "1995-07-28", "http://hospital.example/mrn", "MRN-10005"},
		{uuid.New(), "Brown", "Michael", "male", "1980-01-14", "http://hospital.example/mrn", "MRN-10006"},
		{uuid.New(), "Patel", "Lisa", "female", "1988-09-19", "http://hospital.example/mrn", "MRN-10007"},
		{uuid.New(), "Kim", "David", "male", "1975-12-05", "http://hospital.example/mrn", "MRN-10008"},
		{uuid.New(), "White", "Jennifer", "female", "1993-04-22", "http://hospital.example/mrn", "MRN-10009"},
		{uuid.New(), "Anderson", "Thomas", "male", "1965-06-30", "http://hospital.example/mrn", "MRN-10010"},
	}

	for _, p := range patients {
		_, err := db.Exec(`
			INSERT INTO patients (id, active, name_family, name_given, gender, birth_date, identifier_system, identifier_value, created_at, updated_at)
			VALUES ($1, true, $2, $3, $4, $5, $6, $7, $8, $8)`,
			p.id, p.family, pq.Array([]string{p.given}), p.gender, p.birthDate, p.idSystem, p.idValue, ago(90))
		if err != nil {
			log.Fatalf("insert patient %s: %v", p.family, err)
		}
	}

	// ── Goals ──

	type goal struct {
		id               uuid.UUID
		lifecycleStatus  string
		achievementStatus string
		categoryCode     string
		categoryDisplay  string
		priority         string
		description      string
		subjectID        uuid.UUID
		measureCode      string
		measureDisplay   string
		detailValue      float64
		detailUnit       string
		targetDue        *time.Time
		startDate        time.Time
		statusDate       *time.Time
	}

	future30 := now.AddDate(0, 0, 30)
	future60 := now.AddDate(0, 0, 60)
	future90 := now.AddDate(0, 0, 90)
	past7 := ago(7)

	goals := []goal{
		// Maria Garcia – weight loss + HbA1c
		{uuid.New(), "active", "in-progress", "dietary", "Dietary", "high-priority", "Reduce body weight to 75 kg", patients[0].id, "29463-7", "Body weight", 75, "kg", &future60, ago(80), nil},
		{uuid.New(), "active", "in-progress", "physiotherapy", "Physiotherapy", "high-priority", "Reduce HbA1c below 6.5%", patients[0].id, "4548-4", "HbA1c", 6.5, "%", &future90, ago(75), nil},
		{uuid.New(), "active", "in-progress", "behavioral", "Behavioral", "medium-priority", "Practice daily mindfulness for stress reduction", patients[0].id, "", "", 0, "", &future60, ago(45), nil},

		// James Wilson – blood pressure + exercise
		{uuid.New(), "active", "in-progress", "physiotherapy", "Physiotherapy", "high-priority", "Lower systolic blood pressure below 130 mmHg", patients[1].id, "8480-6", "Systolic blood pressure", 130, "mmHg", &future30, ago(85), nil},
		{uuid.New(), "active", "in-progress", "physical-activity", "Physical Activity", "medium-priority", "Achieve 10000 steps per day", patients[1].id, "41950-7", "Steps per day", 10000, "steps", &future60, ago(70), nil},
		{uuid.New(), "proposed", "in-progress", "dietary", "Dietary", "medium-priority", "Reduce daily sodium intake to under 2g", patients[1].id, "", "", 0, "", &future90, ago(10), nil},

		// Sarah Chen – exercise + mental health
		{uuid.New(), "active", "in-progress", "physical-activity", "Physical Activity", "high-priority", "Walk 8000 steps daily", patients[2].id, "41950-7", "Steps per day", 8000, "steps", &future30, ago(60), nil},
		{uuid.New(), "completed", "achieved", "behavioral", "Behavioral", "medium-priority", "Complete 8-week CBT program", patients[2].id, "", "", 0, "", nil, ago(90), &past7},
		{uuid.New(), "active", "in-progress", "behavioral", "Behavioral", "medium-priority", "Reduce anxiety score below moderate threshold", patients[2].id, "", "", 0, "", &future60, ago(50), nil},

		// Robert Johnson – weight + blood pressure + HbA1c
		{uuid.New(), "active", "in-progress", "dietary", "Dietary", "high-priority", "Reach target weight of 90 kg", patients[3].id, "29463-7", "Body weight", 90, "kg", &future60, ago(88), nil},
		{uuid.New(), "active", "in-progress", "physiotherapy", "Physiotherapy", "high-priority", "Maintain systolic BP under 135 mmHg", patients[3].id, "8480-6", "Systolic blood pressure", 135, "mmHg", &future60, ago(85), nil},
		{uuid.New(), "active", "in-progress", "physiotherapy", "Physiotherapy", "medium-priority", "Reduce HbA1c to 7.0%", patients[3].id, "4548-4", "HbA1c", 7.0, "%", &future90, ago(80), nil},

		// Emily Thompson – exercise + weight
		{uuid.New(), "active", "in-progress", "physical-activity", "Physical Activity", "high-priority", "Reach 12000 steps daily", patients[4].id, "41950-7", "Steps per day", 12000, "steps", &future60, ago(65), nil},
		{uuid.New(), "completed", "achieved", "dietary", "Dietary", "medium-priority", "Lose 5 kg over 2 months", patients[4].id, "29463-7", "Body weight", 60, "kg", nil, ago(90), &past7},

		// Michael Brown – blood pressure + exercise
		{uuid.New(), "active", "in-progress", "physiotherapy", "Physiotherapy", "high-priority", "Lower systolic BP below 125 mmHg", patients[5].id, "8480-6", "Systolic blood pressure", 125, "mmHg", &future60, ago(75), nil},
		{uuid.New(), "active", "in-progress", "physical-activity", "Physical Activity", "medium-priority", "Build up to 9000 steps daily", patients[5].id, "41950-7", "Steps per day", 9000, "steps", &future30, ago(60), nil},
		{uuid.New(), "proposed", "in-progress", "dietary", "Dietary", "low-priority", "Adopt Mediterranean diet", patients[5].id, "", "", 0, "", &future90, ago(5), nil},

		// Lisa Patel – HbA1c + weight
		{uuid.New(), "active", "in-progress", "physiotherapy", "Physiotherapy", "high-priority", "Bring HbA1c below 7.0%", patients[6].id, "4548-4", "HbA1c", 7.0, "%", &future60, ago(70), nil},
		{uuid.New(), "active", "in-progress", "dietary", "Dietary", "medium-priority", "Reduce weight to 68 kg", patients[6].id, "29463-7", "Body weight", 68, "kg", &future90, ago(65), nil},

		// David Kim – blood pressure + mental health
		{uuid.New(), "active", "in-progress", "physiotherapy", "Physiotherapy", "high-priority", "Reduce systolic BP below 128 mmHg", patients[7].id, "8480-6", "Systolic blood pressure", 128, "mmHg", &future60, ago(80), nil},
		{uuid.New(), "active", "in-progress", "behavioral", "Behavioral", "medium-priority", "Improve sleep quality with sleep hygiene program", patients[7].id, "", "", 0, "", &future30, ago(55), nil},
		{uuid.New(), "proposed", "in-progress", "physical-activity", "Physical Activity", "low-priority", "Start regular swimming routine", patients[7].id, "", "", 0, "", &future90, ago(8), nil},

		// Jennifer White – exercise + weight + mental health
		{uuid.New(), "active", "in-progress", "physical-activity", "Physical Activity", "high-priority", "Walk 10000 steps per day consistently", patients[8].id, "41950-7", "Steps per day", 10000, "steps", &future60, ago(72), nil},
		{uuid.New(), "active", "in-progress", "dietary", "Dietary", "medium-priority", "Reach target weight of 62 kg", patients[8].id, "29463-7", "Body weight", 62, "kg", &future90, ago(68), nil},
		{uuid.New(), "completed", "achieved", "behavioral", "Behavioral", "medium-priority", "Complete stress management workshop", patients[8].id, "", "", 0, "", nil, ago(85), &past7},

		// Thomas Anderson – blood pressure + HbA1c + exercise
		{uuid.New(), "active", "in-progress", "physiotherapy", "Physiotherapy", "high-priority", "Lower systolic BP to 130 mmHg", patients[9].id, "8480-6", "Systolic blood pressure", 130, "mmHg", &future60, ago(82), nil},
		{uuid.New(), "active", "in-progress", "physiotherapy", "Physiotherapy", "high-priority", "Reduce HbA1c below 7.5%", patients[9].id, "4548-4", "HbA1c", 7.5, "%", &future90, ago(78), nil},
		{uuid.New(), "active", "in-progress", "physical-activity", "Physical Activity", "medium-priority", "Increase daily steps to 7500", patients[9].id, "41950-7", "Steps per day", 7500, "steps", &future30, ago(60), nil},
		{uuid.New(), "proposed", "in-progress", "dietary", "Dietary", "low-priority", "Reduce processed food consumption", patients[9].id, "", "", 0, "", &future90, ago(3), nil},
	}

	for _, g := range goals {
		_, err := db.Exec(`
			INSERT INTO goals (id, lifecycle_status, achievement_status, category_code, category_display, priority, description_text,
				subject_id, target_measure_code, target_measure_display, target_detail_value, target_detail_unit,
				target_due_date, start_date, status_date, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$16)`,
			g.id, g.lifecycleStatus, g.achievementStatus, g.categoryCode, g.categoryDisplay, g.priority, g.description,
			g.subjectID, g.measureCode, g.measureDisplay, g.detailValue, g.detailUnit,
			g.targetDue, g.startDate, g.statusDate, g.startDate)
		if err != nil {
			log.Fatalf("insert goal %q: %v", g.description, err)
		}
	}

	// ── Observations ──

	type obs struct {
		subjectID uuid.UUID
		code      string
		display   string
		unit      string
		unitCode  string
		values    []float64
		startDay  int
		interval  int
	}

	observations := []obs{
		{patients[0].id, "29463-7", "Body weight", "kg", "kg", []float64{95, 92, 89, 87.5}, 84, 21},
		{patients[0].id, "4548-4", "HbA1c", "%", "%", []float64{8.2, 7.6, 7.0}, 80, 28},

		{patients[1].id, "8480-6", "Systolic blood pressure", "mmHg", "mm[Hg]", []float64{152, 145, 138, 134}, 84, 21},
		{patients[1].id, "41950-7", "Steps per day", "steps", "{steps}", []float64{4200, 6000, 8100, 9000}, 70, 18},

		{patients[2].id, "41950-7", "Steps per day", "steps", "{steps}", []float64{3500, 5500, 7200, 7800}, 60, 15},

		{patients[3].id, "29463-7", "Body weight", "kg", "kg", []float64{105, 101, 97, 95}, 84, 21},
		{patients[3].id, "8480-6", "Systolic blood pressure", "mmHg", "mm[Hg]", []float64{155, 147, 140, 137}, 84, 21},
		{patients[3].id, "4548-4", "HbA1c", "%", "%", []float64{8.5, 7.7, 7.3}, 80, 28},

		{patients[4].id, "41950-7", "Steps per day", "steps", "{steps}", []float64{6000, 9000, 11500, 12000}, 84, 21},
		{patients[4].id, "29463-7", "Body weight", "kg", "kg", []float64{65, 63, 61, 60}, 84, 21},

		{patients[5].id, "8480-6", "Systolic blood pressure", "mmHg", "mm[Hg]", []float64{145, 138, 131, 128}, 75, 18},
		{patients[5].id, "41950-7", "Steps per day", "steps", "{steps}", []float64{3800, 5800, 7700}, 60, 21},

		{patients[6].id, "4548-4", "HbA1c", "%", "%", []float64{8.0, 7.3, 7.1}, 70, 28},
		{patients[6].id, "29463-7", "Body weight", "kg", "kg", []float64{78, 75, 72, 70.5}, 65, 15},

		{patients[7].id, "8480-6", "Systolic blood pressure", "mmHg", "mm[Hg]", []float64{148, 140, 133, 130}, 80, 21},

		{patients[8].id, "41950-7", "Steps per day", "steps", "{steps}", []float64{4000, 6500, 8600}, 72, 21},
		{patients[8].id, "29463-7", "Body weight", "kg", "kg", []float64{70, 68, 66, 64.5}, 68, 18},

		{patients[9].id, "8480-6", "Systolic blood pressure", "mmHg", "mm[Hg]", []float64{158, 150, 142, 138}, 82, 21},
		{patients[9].id, "4548-4", "HbA1c", "%", "%", []float64{8.8, 7.9, 7.6}, 78, 28},
		{patients[9].id, "41950-7", "Steps per day", "steps", "{steps}", []float64{2500, 4500, 6300}, 60, 21},
	}

	catCode := "vital-signs"
	catDisplay := "Vital Signs"
	obsCount := 0

	for _, o := range observations {
		cat := catCode
		catDisp := catDisplay
		if o.code == "41950-7" {
			cat = "activity"
			catDisp = "Activity"
		}
		for i, v := range o.values {
			effectiveDate := ago(o.startDay - i*o.interval)
			_, err := db.Exec(`
				INSERT INTO observations (id, status, category_code, category_display, code_code, code_display,
					subject_id, effective_date, value_quantity_value, value_quantity_unit, value_quantity_code, created_at, updated_at)
				VALUES ($1,'final',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)`,
				uuid.New(), cat, catDisp, o.code, o.display,
				o.subjectID, effectiveDate, v, o.unit, o.unitCode, effectiveDate)
			if err != nil {
				log.Fatalf("insert observation %s for patient: %v", o.display, err)
			}
			obsCount++
		}
	}

	fmt.Printf("Seeded: %d patients, %d goals, %d observations\n",
		len(patients), len(goals), obsCount)
}
