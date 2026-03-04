package patient

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNameRequired = errors.New("patient name must include at least a family or given name")
	ErrNotFound     = errors.New("patient not found")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, p *Patient) (*Patient, error) {
	if err := validatePatient(p); err != nil {
		return nil, err
	}

	p.ResourceType = "Patient"
	p.Meta.LastUpdated = time.Now().UTC()

	return s.repo.Create(ctx, p)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Patient, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Service) Search(ctx context.Context, params map[string]string) ([]Patient, error) {
	return s.repo.Search(ctx, params)
}

func (s *Service) Update(ctx context.Context, id string, p *Patient) (*Patient, error) {
	if err := validatePatient(p); err != nil {
		return nil, err
	}

	p.Meta.LastUpdated = time.Now().UTC()

	result, err := s.repo.Update(ctx, id, p)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrNotFound
	}
	return result, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func validatePatient(p *Patient) error {
	if len(p.Name) == 0 {
		return ErrNameRequired
	}
	name := p.Name[0]
	if name.Family == "" && len(name.Given) == 0 {
		return ErrNameRequired
	}
	return nil
}
