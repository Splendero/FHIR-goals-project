package careplan

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

type dbRow struct {
	ID              string         `db:"id"`
	Status          string         `db:"status"`
	Intent          string         `db:"intent"`
	Title           string         `db:"title"`
	Description     sql.NullString `db:"description"`
	SubjectID       string         `db:"subject_id"`
	PeriodStart     sql.NullTime   `db:"period_start"`
	PeriodEnd       sql.NullTime   `db:"period_end"`
	GoalIDs         pq.StringArray `db:"goal_ids"`
	CategoryCode    sql.NullString `db:"category_code"`
	CategoryDisplay sql.NullString `db:"category_display"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

func (r *Repository) Create(ctx context.Context, cp *CarePlan) (*CarePlan, error) {
	row := toDBRow(cp)
	row.ID = uuid.New().String()

	query := `
		INSERT INTO care_plans (id, status, intent, title, description, subject_id, period_start, period_end, goal_ids, category_code, category_display)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		row.ID, row.Status, row.Intent, row.Title, row.Description,
		row.SubjectID, row.PeriodStart, row.PeriodEnd, row.GoalIDs,
		row.CategoryCode, row.CategoryDisplay,
	).Scan(&row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert care_plan: %w", err)
	}

	result := toModel(row)
	return &result, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*CarePlan, error) {
	var row dbRow
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, status, intent, title, description, subject_id, period_start, period_end, goal_ids, category_code, category_display, created_at, updated_at
		 FROM care_plans WHERE id = $1`, id,
	).StructScan(&row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get care_plan %s: %w", id, err)
	}

	result := toModel(row)
	return &result, nil
}

func (r *Repository) Search(ctx context.Context, params map[string]string) ([]CarePlan, error) {
	query := `SELECT id, status, intent, title, description, subject_id, period_start, period_end, goal_ids, category_code, category_display, created_at, updated_at FROM care_plans`
	var conditions []string
	var args []interface{}
	argIdx := 1

	if patient, ok := params["patient"]; ok {
		conditions = append(conditions, fmt.Sprintf("subject_id = $%d", argIdx))
		args = append(args, patient)
		argIdx++
	}
	if status, ok := params["status"]; ok {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY updated_at DESC"

	var rows []dbRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("search care_plans: %w", err)
	}

	results := make([]CarePlan, len(rows))
	for i, row := range rows {
		results[i] = toModel(row)
	}
	return results, nil
}

func (r *Repository) Update(ctx context.Context, id string, cp *CarePlan) (*CarePlan, error) {
	row := toDBRow(cp)

	query := `
		UPDATE care_plans
		SET status = $1, intent = $2, title = $3, description = $4, subject_id = $5,
		    period_start = $6, period_end = $7, goal_ids = $8, category_code = $9,
		    category_display = $10, updated_at = NOW()
		WHERE id = $11
		RETURNING created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		row.Status, row.Intent, row.Title, row.Description, row.SubjectID,
		row.PeriodStart, row.PeriodEnd, row.GoalIDs, row.CategoryCode,
		row.CategoryDisplay, id,
	).Scan(&row.CreatedAt, &row.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update care_plan %s: %w", id, err)
	}

	row.ID = id
	result := toModel(row)
	return &result, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM care_plans WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete care_plan %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("care_plan %s not found", id)
	}
	return nil
}

func toDBRow(cp *CarePlan) dbRow {
	row := dbRow{
		Status:    cp.Status,
		Intent:    cp.Intent,
		Title:     cp.Title,
		SubjectID: extractID(cp.Subject.Reference),
	}

	if cp.Description != "" {
		row.Description = sql.NullString{String: cp.Description, Valid: true}
	}

	if cp.Period != nil {
		if cp.Period.Start != nil {
			row.PeriodStart = sql.NullTime{Time: *cp.Period.Start, Valid: true}
		}
		if cp.Period.End != nil {
			row.PeriodEnd = sql.NullTime{Time: *cp.Period.End, Valid: true}
		}
	}

	goalIDs := make([]string, len(cp.Goal))
	for i, g := range cp.Goal {
		goalIDs[i] = extractID(g.Reference)
	}
	row.GoalIDs = goalIDs

	if len(cp.Category) > 0 && len(cp.Category[0].Coding) > 0 {
		row.CategoryCode = sql.NullString{String: cp.Category[0].Coding[0].Code, Valid: true}
		row.CategoryDisplay = sql.NullString{String: cp.Category[0].Coding[0].Display, Valid: true}
	}

	return row
}

func toModel(row dbRow) CarePlan {
	cp := CarePlan{
		ResourceType: "CarePlan",
		ID:           row.ID,
		Meta: fhir.Meta{
			VersionID:   "1",
			LastUpdated: row.UpdatedAt,
		},
		Status: row.Status,
		Intent: row.Intent,
		Title:  row.Title,
		Subject: fhir.Reference{
			Reference: fmt.Sprintf("Patient/%s", row.SubjectID),
		},
	}

	if row.Description.Valid {
		cp.Description = row.Description.String
	}

	if row.PeriodStart.Valid || row.PeriodEnd.Valid {
		cp.Period = &fhir.Period{}
		if row.PeriodStart.Valid {
			t := row.PeriodStart.Time
			cp.Period.Start = &t
		}
		if row.PeriodEnd.Valid {
			t := row.PeriodEnd.Time
			cp.Period.End = &t
		}
	}

	if len(row.GoalIDs) > 0 {
		cp.Goal = make([]fhir.Reference, len(row.GoalIDs))
		for i, gid := range row.GoalIDs {
			cp.Goal[i] = fhir.Reference{Reference: fmt.Sprintf("Goal/%s", gid)}
		}
	}

	if row.CategoryCode.Valid {
		cp.Category = []fhir.CodeableConcept{{
			Coding: []fhir.Coding{{
				Code:    row.CategoryCode.String,
				Display: row.CategoryDisplay.String,
			}},
		}}
	}

	return cp
}

func extractID(ref string) string {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ref
}
