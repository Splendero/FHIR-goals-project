package patient

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
	ID               string         `db:"id"`
	Active           bool           `db:"active"`
	NameFamily       sql.NullString `db:"name_family"`
	NameGiven        pq.StringArray `db:"name_given"`
	Gender           sql.NullString `db:"gender"`
	BirthDate        sql.NullString `db:"birth_date"`
	IdentifierSystem sql.NullString `db:"identifier_system"`
	IdentifierValue  sql.NullString `db:"identifier_value"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
}

func (r *Repository) Create(ctx context.Context, p *Patient) (*Patient, error) {
	params := toDBParams(*p)
	row := r.db.QueryRowxContext(ctx,
		`INSERT INTO patients (active, name_family, name_given, gender, birth_date, identifier_system, identifier_value)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, active, name_family, name_given, gender, birth_date, identifier_system, identifier_value, created_at, updated_at`,
		params.Active, params.NameFamily, params.NameGiven, params.Gender, params.BirthDate,
		params.IdentifierSystem, params.IdentifierValue,
	)

	var created dbRow
	if err := row.StructScan(&created); err != nil {
		return nil, fmt.Errorf("inserting patient: %w", err)
	}

	result := toModel(created)
	return &result, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Patient, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid patient id: %w", err)
	}

	var row dbRow
	if err := r.db.GetContext(ctx, &row, `SELECT * FROM patients WHERE id = $1`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("querying patient: %w", err)
	}

	result := toModel(row)
	return &result, nil
}

func (r *Repository) Search(ctx context.Context, params map[string]string) ([]Patient, error) {
	query := `SELECT * FROM patients WHERE 1=1`
	args := []interface{}{}
	idx := 1

	if name, ok := params["name"]; ok {
		query += fmt.Sprintf(` AND (LOWER(name_family) LIKE LOWER($%d) OR EXISTS (SELECT 1 FROM unnest(name_given) g WHERE LOWER(g) LIKE LOWER($%d)))`, idx, idx)
		args = append(args, "%"+name+"%")
		idx++
	}

	if gender, ok := params["gender"]; ok {
		query += fmt.Sprintf(` AND gender = $%d`, idx)
		args = append(args, strings.ToLower(gender))
		idx++
	}

	query += ` ORDER BY created_at DESC`

	var rows []dbRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("searching patients: %w", err)
	}

	patients := make([]Patient, len(rows))
	for i, row := range rows {
		patients[i] = toModel(row)
	}
	return patients, nil
}

func (r *Repository) Update(ctx context.Context, id string, p *Patient) (*Patient, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid patient id: %w", err)
	}

	params := toDBParams(*p)
	row := r.db.QueryRowxContext(ctx,
		`UPDATE patients
		 SET active = $1, name_family = $2, name_given = $3, gender = $4, birth_date = $5,
		     identifier_system = $6, identifier_value = $7, updated_at = NOW()
		 WHERE id = $8
		 RETURNING id, active, name_family, name_given, gender, birth_date, identifier_system, identifier_value, created_at, updated_at`,
		params.Active, params.NameFamily, params.NameGiven, params.Gender, params.BirthDate,
		params.IdentifierSystem, params.IdentifierValue, id,
	)

	var updated dbRow
	if err := row.StructScan(&updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("updating patient: %w", err)
	}

	result := toModel(updated)
	return &result, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid patient id: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM patients WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting patient: %w", err)
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

func toModel(row dbRow) Patient {
	p := Patient{
		ResourceType: "Patient",
		ID:           row.ID,
		Meta: fhir.Meta{
			LastUpdated: row.UpdatedAt,
		},
		Active: row.Active,
	}

	hn := fhir.HumanName{}
	if row.NameFamily.Valid {
		hn.Family = row.NameFamily.String
	}
	if len(row.NameGiven) > 0 {
		hn.Given = []string(row.NameGiven)
	}
	if hn.Family != "" || len(hn.Given) > 0 {
		p.Name = []fhir.HumanName{hn}
	}

	if row.Gender.Valid {
		p.Gender = row.Gender.String
	}
	if row.BirthDate.Valid {
		p.BirthDate = row.BirthDate.String
	}

	if row.IdentifierSystem.Valid || row.IdentifierValue.Valid {
		id := fhir.Identifier{}
		if row.IdentifierSystem.Valid {
			id.System = row.IdentifierSystem.String
		}
		if row.IdentifierValue.Valid {
			id.Value = row.IdentifierValue.String
		}
		p.Identifier = []fhir.Identifier{id}
	}

	return p
}

func toDBParams(p Patient) dbRow {
	row := dbRow{
		Active: p.Active,
	}

	if len(p.Name) > 0 {
		name := p.Name[0]
		if name.Family != "" {
			row.NameFamily = sql.NullString{String: name.Family, Valid: true}
		}
		if len(name.Given) > 0 {
			row.NameGiven = pq.StringArray(name.Given)
		}
	}

	if p.Gender != "" {
		row.Gender = sql.NullString{String: p.Gender, Valid: true}
	}
	if p.BirthDate != "" {
		row.BirthDate = sql.NullString{String: p.BirthDate, Valid: true}
	}

	if len(p.Identifier) > 0 {
		ident := p.Identifier[0]
		if ident.System != "" {
			row.IdentifierSystem = sql.NullString{String: ident.System, Valid: true}
		}
		if ident.Value != "" {
			row.IdentifierValue = sql.NullString{String: ident.Value, Valid: true}
		}
	}

	return row
}
