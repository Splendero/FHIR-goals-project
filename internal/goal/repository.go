package goal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

type dbRow struct {
	ID                   string          `db:"id"`
	LifecycleStatus      string          `db:"lifecycle_status"`
	AchievementStatus    string          `db:"achievement_status"`
	CategoryCode         sql.NullString  `db:"category_code"`
	CategoryDisplay      sql.NullString  `db:"category_display"`
	Priority             sql.NullString  `db:"priority"`
	DescriptionText      string          `db:"description_text"`
	SubjectID            string          `db:"subject_id"`
	TargetMeasureCode    sql.NullString  `db:"target_measure_code"`
	TargetMeasureDisplay sql.NullString  `db:"target_measure_display"`
	TargetDetailValue    sql.NullFloat64 `db:"target_detail_value"`
	TargetDetailUnit     sql.NullString  `db:"target_detail_unit"`
	TargetDueDate        sql.NullTime    `db:"target_due_date"`
	StartDate            sql.NullTime    `db:"start_date"`
	StatusDate           sql.NullTime    `db:"status_date"`
	Note                 sql.NullString  `db:"note"`
	CreatedAt            time.Time       `db:"created_at"`
	UpdatedAt            time.Time       `db:"updated_at"`
}

func (r *Repository) Create(ctx context.Context, g *Goal) (*Goal, error) {
	row := toDBRow(g)
	row.ID = uuid.New().String()

	query := `INSERT INTO goals (
		id, lifecycle_status, achievement_status, category_code, category_display,
		priority, description_text, subject_id, target_measure_code, target_measure_display,
		target_detail_value, target_detail_unit, target_due_date, start_date, status_date, note
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
	) RETURNING *`

	var result dbRow
	err := r.db.QueryRowxContext(ctx, query,
		row.ID, row.LifecycleStatus, row.AchievementStatus,
		row.CategoryCode, row.CategoryDisplay, row.Priority,
		row.DescriptionText, row.SubjectID,
		row.TargetMeasureCode, row.TargetMeasureDisplay,
		row.TargetDetailValue, row.TargetDetailUnit,
		row.TargetDueDate, row.StartDate, row.StatusDate, row.Note,
	).StructScan(&result)
	if err != nil {
		return nil, fmt.Errorf("create goal: %w", err)
	}

	out := toModel(result)
	return &out, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Goal, error) {
	var row dbRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM goals WHERE id = $1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: goal %s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("get goal: %w", err)
	}
	out := toModel(row)
	return &out, nil
}

func (r *Repository) Search(ctx context.Context, params map[string]string) ([]Goal, error) {
	query := "SELECT * FROM goals WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if v, ok := params["patient"]; ok {
		query += fmt.Sprintf(" AND subject_id = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v, ok := params["status"]; ok {
		query += fmt.Sprintf(" AND lifecycle_status = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v, ok := params["category"]; ok {
		query += fmt.Sprintf(" AND category_code = $%d", idx)
		args = append(args, v)
		idx++
	}

	query += " ORDER BY updated_at DESC"

	var rows []dbRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("search goals: %w", err)
	}

	goals := make([]Goal, len(rows))
	for i, row := range rows {
		goals[i] = toModel(row)
	}
	return goals, nil
}

func (r *Repository) Update(ctx context.Context, id string, g *Goal) (*Goal, error) {
	row := toDBRow(g)

	query := `UPDATE goals SET
		lifecycle_status = $1, achievement_status = $2,
		category_code = $3, category_display = $4, priority = $5,
		description_text = $6, subject_id = $7,
		target_measure_code = $8, target_measure_display = $9,
		target_detail_value = $10, target_detail_unit = $11, target_due_date = $12,
		start_date = $13, status_date = $14, note = $15,
		updated_at = NOW()
	WHERE id = $16
	RETURNING *`

	var result dbRow
	err := r.db.QueryRowxContext(ctx, query,
		row.LifecycleStatus, row.AchievementStatus,
		row.CategoryCode, row.CategoryDisplay, row.Priority,
		row.DescriptionText, row.SubjectID,
		row.TargetMeasureCode, row.TargetMeasureDisplay,
		row.TargetDetailValue, row.TargetDetailUnit,
		row.TargetDueDate, row.StartDate, row.StatusDate, row.Note,
		id,
	).StructScan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: goal %s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("update goal: %w", err)
	}

	out := toModel(result)
	return &out, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM goals WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete goal: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete goal: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: goal %s", ErrNotFound, id)
	}
	return nil
}

func (r *Repository) GetBySubjectAndStatus(ctx context.Context, subjectID, status string) ([]Goal, error) {
	var rows []dbRow
	err := r.db.SelectContext(ctx, &rows,
		"SELECT * FROM goals WHERE subject_id = $1 AND lifecycle_status = $2 ORDER BY updated_at DESC",
		subjectID, status,
	)
	if err != nil {
		return nil, fmt.Errorf("get goals by subject and status: %w", err)
	}

	goals := make([]Goal, len(rows))
	for i, row := range rows {
		goals[i] = toModel(row)
	}
	return goals, nil
}

func toModel(row dbRow) Goal {
	g := Goal{
		ResourceType:    "Goal",
		ID:              row.ID,
		Meta:            fhir.Meta{LastUpdated: row.UpdatedAt},
		LifecycleStatus: row.LifecycleStatus,
		Description:     fhir.CodeableConcept{Text: row.DescriptionText},
		Subject:         fhir.Reference{Reference: "Patient/" + row.SubjectID},
	}

	if row.AchievementStatus != "" {
		g.AchievementStatus = &fhir.CodeableConcept{
			Coding: []fhir.Coding{{Code: row.AchievementStatus}},
			Text:   row.AchievementStatus,
		}
	}

	if row.CategoryCode.Valid {
		g.Category = []fhir.CodeableConcept{{
			Coding: []fhir.Coding{{
				Code:    row.CategoryCode.String,
				Display: row.CategoryDisplay.String,
			}},
		}}
	}

	if row.Priority.Valid {
		g.Priority = &fhir.CodeableConcept{
			Coding: []fhir.Coding{{Code: row.Priority.String}},
		}
	}

	if row.TargetMeasureCode.Valid || row.TargetDetailValue.Valid || row.TargetDueDate.Valid {
		t := GoalTarget{}
		if row.TargetMeasureCode.Valid {
			t.Measure = &fhir.CodeableConcept{
				Coding: []fhir.Coding{{
					Code:    row.TargetMeasureCode.String,
					Display: row.TargetMeasureDisplay.String,
				}},
			}
		}
		if row.TargetDetailValue.Valid {
			t.DetailQuantity = &fhir.Quantity{
				Value: row.TargetDetailValue.Float64,
				Unit:  row.TargetDetailUnit.String,
			}
		}
		if row.TargetDueDate.Valid {
			t.DueDate = row.TargetDueDate.Time.Format("2006-01-02")
		}
		g.Target = []GoalTarget{t}
	}

	if row.StartDate.Valid {
		g.StartDate = row.StartDate.Time.Format("2006-01-02")
	}
	if row.StatusDate.Valid {
		g.StatusDate = row.StatusDate.Time.Format("2006-01-02")
	}

	if row.Note.Valid {
		g.Note = []fhir.Annotation{{Text: row.Note.String}}
	}

	return g
}

func toDBRow(g *Goal) dbRow {
	row := dbRow{
		LifecycleStatus:   g.LifecycleStatus,
		AchievementStatus: AchievementInProgress,
		DescriptionText:   g.Description.Text,
		SubjectID:         strings.TrimPrefix(g.Subject.Reference, "Patient/"),
	}

	if g.Description.Text == "" && len(g.Description.Coding) > 0 {
		row.DescriptionText = g.Description.Coding[0].Display
	}

	if g.AchievementStatus != nil {
		if len(g.AchievementStatus.Coding) > 0 {
			row.AchievementStatus = g.AchievementStatus.Coding[0].Code
		} else if g.AchievementStatus.Text != "" {
			row.AchievementStatus = g.AchievementStatus.Text
		}
	}

	if len(g.Category) > 0 && len(g.Category[0].Coding) > 0 {
		row.CategoryCode = sql.NullString{String: g.Category[0].Coding[0].Code, Valid: true}
		row.CategoryDisplay = sql.NullString{String: g.Category[0].Coding[0].Display, Valid: true}
	}

	if g.Priority != nil && len(g.Priority.Coding) > 0 {
		row.Priority = sql.NullString{String: g.Priority.Coding[0].Code, Valid: true}
	}

	if len(g.Target) > 0 {
		t := g.Target[0]
		if t.Measure != nil && len(t.Measure.Coding) > 0 {
			row.TargetMeasureCode = sql.NullString{String: t.Measure.Coding[0].Code, Valid: true}
			row.TargetMeasureDisplay = sql.NullString{String: t.Measure.Coding[0].Display, Valid: true}
		}
		if t.DetailQuantity != nil {
			row.TargetDetailValue = sql.NullFloat64{Float64: t.DetailQuantity.Value, Valid: true}
			row.TargetDetailUnit = sql.NullString{String: t.DetailQuantity.Unit, Valid: true}
		}
		if t.DueDate != "" {
			if parsed, err := time.Parse("2006-01-02", t.DueDate); err == nil {
				row.TargetDueDate = sql.NullTime{Time: parsed, Valid: true}
			}
		}
	}

	if g.StartDate != "" {
		if parsed, err := time.Parse("2006-01-02", g.StartDate); err == nil {
			row.StartDate = sql.NullTime{Time: parsed, Valid: true}
		}
	}
	if g.StatusDate != "" {
		if parsed, err := time.Parse("2006-01-02", g.StatusDate); err == nil {
			row.StatusDate = sql.NullTime{Time: parsed, Valid: true}
		}
	}

	if len(g.Note) > 0 {
		row.Note = sql.NullString{String: g.Note[0].Text, Valid: true}
	}

	return row
}
