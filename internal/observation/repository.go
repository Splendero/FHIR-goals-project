package observation

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
	ID                 string          `db:"id"`
	Status             string          `db:"status"`
	CategoryCode       sql.NullString  `db:"category_code"`
	CategoryDisplay    sql.NullString  `db:"category_display"`
	CodeCode           string          `db:"code_code"`
	CodeDisplay        sql.NullString  `db:"code_display"`
	SubjectID          string          `db:"subject_id"`
	EffectiveDate      time.Time       `db:"effective_date"`
	ValueQuantityValue sql.NullFloat64 `db:"value_quantity_value"`
	ValueQuantityUnit  sql.NullString  `db:"value_quantity_unit"`
	ValueQuantityCode  sql.NullString  `db:"value_quantity_code"`
	Note               sql.NullString  `db:"note"`
	CreatedAt          time.Time       `db:"created_at"`
	UpdatedAt          time.Time       `db:"updated_at"`
}

func (r *Repository) Create(ctx context.Context, o *Observation) (*Observation, error) {
	params := toDBParams(*o)
	row := r.db.QueryRowxContext(ctx,
		`INSERT INTO observations (status, category_code, category_display, code_code, code_display,
		 subject_id, effective_date, value_quantity_value, value_quantity_unit, value_quantity_code, note)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, status, category_code, category_display, code_code, code_display,
		 subject_id, effective_date, value_quantity_value, value_quantity_unit, value_quantity_code,
		 note, created_at, updated_at`,
		params.Status, params.CategoryCode, params.CategoryDisplay, params.CodeCode, params.CodeDisplay,
		params.SubjectID, params.EffectiveDate, params.ValueQuantityValue, params.ValueQuantityUnit,
		params.ValueQuantityCode, params.Note,
	)

	var created dbRow
	if err := row.StructScan(&created); err != nil {
		return nil, fmt.Errorf("inserting observation: %w", err)
	}

	result := toModel(created)
	return &result, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Observation, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid observation id: %w", err)
	}

	var row dbRow
	if err := r.db.GetContext(ctx, &row, `SELECT * FROM observations WHERE id = $1`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying observation: %w", err)
	}

	result := toModel(row)
	return &result, nil
}

func (r *Repository) Search(ctx context.Context, params map[string]string) ([]Observation, error) {
	query := `SELECT * FROM observations WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if patient, ok := params["patient"]; ok {
		patient = strings.TrimPrefix(patient, "Patient/")
		query += fmt.Sprintf(` AND subject_id = $%d`, idx)
		args = append(args, patient)
		idx++
	}

	if code, ok := params["code"]; ok {
		query += fmt.Sprintf(` AND code_code = $%d`, idx)
		args = append(args, code)
		idx++
	}

	if date, ok := params["date"]; ok {
		query += fmt.Sprintf(` AND effective_date::date = $%d::date`, idx)
		args = append(args, date)
		idx++
	}

	query += ` ORDER BY effective_date DESC`

	var rows []dbRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("searching observations: %w", err)
	}

	observations := make([]Observation, len(rows))
	for i, row := range rows {
		observations[i] = toModel(row)
	}
	return observations, nil
}

func (r *Repository) Update(ctx context.Context, id string, o *Observation) (*Observation, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid observation id: %w", err)
	}

	params := toDBParams(*o)
	row := r.db.QueryRowxContext(ctx,
		`UPDATE observations
		 SET status = $1, category_code = $2, category_display = $3, code_code = $4, code_display = $5,
		     subject_id = $6, effective_date = $7, value_quantity_value = $8, value_quantity_unit = $9,
		     value_quantity_code = $10, note = $11, updated_at = NOW()
		 WHERE id = $12
		 RETURNING id, status, category_code, category_display, code_code, code_display,
		 subject_id, effective_date, value_quantity_value, value_quantity_unit, value_quantity_code,
		 note, created_at, updated_at`,
		params.Status, params.CategoryCode, params.CategoryDisplay, params.CodeCode, params.CodeDisplay,
		params.SubjectID, params.EffectiveDate, params.ValueQuantityValue, params.ValueQuantityUnit,
		params.ValueQuantityCode, params.Note, id,
	)

	var updated dbRow
	if err := row.StructScan(&updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("updating observation: %w", err)
	}

	result := toModel(updated)
	return &result, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid observation id: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM observations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting observation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking delete result: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) GetBySubjectAndCode(ctx context.Context, subjectID, code string) ([]Observation, error) {
	var rows []dbRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM observations WHERE subject_id = $1 AND code_code = $2 ORDER BY effective_date DESC`,
		subjectID, code,
	); err != nil {
		return nil, fmt.Errorf("querying observations by subject and code: %w", err)
	}

	observations := make([]Observation, len(rows))
	for i, row := range rows {
		observations[i] = toModel(row)
	}
	return observations, nil
}

func toModel(row dbRow) Observation {
	o := Observation{
		ResourceType:      "Observation",
		ID:                row.ID,
		Meta:              fhir.Meta{LastUpdated: row.UpdatedAt},
		Status:            row.Status,
		Code:              fhir.CodeableConcept{Coding: []fhir.Coding{{Code: row.CodeCode}}},
		Subject:           fhir.Reference{Reference: "Patient/" + row.SubjectID},
		EffectiveDateTime: row.EffectiveDate.Format(time.RFC3339),
	}

	if row.CodeDisplay.Valid {
		o.Code.Coding[0].Display = row.CodeDisplay.String
	}

	if row.CategoryCode.Valid {
		cat := fhir.CodeableConcept{Coding: []fhir.Coding{{Code: row.CategoryCode.String}}}
		if row.CategoryDisplay.Valid {
			cat.Coding[0].Display = row.CategoryDisplay.String
		}
		o.Category = []fhir.CodeableConcept{cat}
	}

	if row.ValueQuantityValue.Valid {
		q := &fhir.Quantity{Value: row.ValueQuantityValue.Float64}
		if row.ValueQuantityUnit.Valid {
			q.Unit = row.ValueQuantityUnit.String
		}
		if row.ValueQuantityCode.Valid {
			q.Code = row.ValueQuantityCode.String
		}
		o.ValueQuantity = q
	}

	if row.Note.Valid {
		o.Note = []fhir.Annotation{{Text: row.Note.String}}
	}

	return o
}

func toDBParams(o Observation) dbRow {
	row := dbRow{
		Status:   o.Status,
		CodeCode: extractCode(o.Code),
	}

	if len(o.Code.Coding) > 0 && o.Code.Coding[0].Display != "" {
		row.CodeDisplay = sql.NullString{String: o.Code.Coding[0].Display, Valid: true}
	}

	row.SubjectID = strings.TrimPrefix(o.Subject.Reference, "Patient/")

	if len(o.Category) > 0 && len(o.Category[0].Coding) > 0 {
		cat := o.Category[0].Coding[0]
		if cat.Code != "" {
			row.CategoryCode = sql.NullString{String: cat.Code, Valid: true}
		}
		if cat.Display != "" {
			row.CategoryDisplay = sql.NullString{String: cat.Display, Valid: true}
		}
	}

	if o.EffectiveDateTime != "" {
		if t, err := time.Parse(time.RFC3339, o.EffectiveDateTime); err == nil {
			row.EffectiveDate = t
		} else {
			row.EffectiveDate = time.Now().UTC()
		}
	} else {
		row.EffectiveDate = time.Now().UTC()
	}

	if o.ValueQuantity != nil {
		row.ValueQuantityValue = sql.NullFloat64{Float64: o.ValueQuantity.Value, Valid: true}
		if o.ValueQuantity.Unit != "" {
			row.ValueQuantityUnit = sql.NullString{String: o.ValueQuantity.Unit, Valid: true}
		}
		if o.ValueQuantity.Code != "" {
			row.ValueQuantityCode = sql.NullString{String: o.ValueQuantity.Code, Valid: true}
		}
	}

	if len(o.Note) > 0 && o.Note[0].Text != "" {
		row.Note = sql.NullString{String: o.Note[0].Text, Valid: true}
	}

	return row
}

func extractCode(cc fhir.CodeableConcept) string {
	if len(cc.Coding) > 0 {
		return cc.Coding[0].Code
	}
	return ""
}
