package goal

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, g *Goal) (*Goal, error) {
	if err := validate(g); err != nil {
		return nil, err
	}
	if g.LifecycleStatus == "" {
		g.LifecycleStatus = LifecycleProposed
	}
	g.ResourceType = "Goal"
	g.Meta.LastUpdated = time.Now().UTC()
	return s.repo.Create(ctx, g)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Goal, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Search(ctx context.Context, params map[string]string) ([]Goal, error) {
	return s.repo.Search(ctx, params)
}

func (s *Service) Update(ctx context.Context, id string, g *Goal) (*Goal, error) {
	if err := validate(g); err != nil {
		return nil, err
	}
	g.ResourceType = "Goal"
	g.Meta.LastUpdated = time.Now().UTC()
	return s.repo.Update(ctx, id, g)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) GetActiveBySubject(ctx context.Context, subjectID string) ([]Goal, error) {
	subjectID = strings.TrimPrefix(subjectID, "Patient/")
	return s.repo.GetBySubjectAndStatus(ctx, subjectID, LifecycleActive)
}

func validate(g *Goal) error {
	if g.Description.Text == "" && len(g.Description.Coding) == 0 {
		return fmt.Errorf("%w: description is required", ErrValidation)
	}
	if g.Subject.Reference == "" {
		return fmt.Errorf("%w: subject is required", ErrValidation)
	}
	if g.LifecycleStatus != "" && !isValidLifecycleStatus(g.LifecycleStatus) {
		return fmt.Errorf("%w: invalid lifecycle status: %s", ErrValidation, g.LifecycleStatus)
	}
	return nil
}
