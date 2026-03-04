package careplan

import (
	"context"
	"errors"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, cp *CarePlan) (*CarePlan, error) {
	if cp.Status == "" {
		cp.Status = "draft"
	}
	if cp.Intent == "" {
		cp.Intent = "plan"
	}
	if err := validate(cp); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, cp)
}

func (s *Service) GetByID(ctx context.Context, id string) (*CarePlan, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Search(ctx context.Context, params map[string]string) ([]CarePlan, error) {
	return s.repo.Search(ctx, params)
}

func (s *Service) Update(ctx context.Context, id string, cp *CarePlan) (*CarePlan, error) {
	if err := validate(cp); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, cp)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func validate(cp *CarePlan) error {
	if cp.Subject.Reference == "" {
		return errors.New("subject is required")
	}
	if cp.Status == "" {
		return errors.New("status is required")
	}
	if !ValidStatuses[cp.Status] {
		return errors.New("invalid status: " + cp.Status)
	}
	if cp.Intent == "" {
		return errors.New("intent is required")
	}
	if !ValidIntents[cp.Intent] {
		return errors.New("invalid intent: " + cp.Intent)
	}
	return nil
}
