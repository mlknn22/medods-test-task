package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	if err := validateScheduleInput(input.Schedule); err != nil {
		return nil, err
	}

	now := s.now()
	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		Schedule:    scheduleInputToModel(input.Schedule),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, model)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	if err := validateScheduleInput(input.Schedule); err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		Schedule:    scheduleInputToModel(input.Schedule),
		UpdatedAt:   s.now(),
	}

	return s.repo.Update(ctx, model)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}

func validateScheduleInput(s *ScheduleInput) error {
	if s == nil {
		return nil
	}

	if !s.Type.Valid() {
		return fmt.Errorf("%w: invalid schedule type %q", ErrInvalidInput, s.Type)
	}

	switch s.Type {
	case taskdomain.ScheduleTypeDaily:
		if s.Interval == nil || *s.Interval < 1 {
			return fmt.Errorf("%w: daily schedule requires interval >= 1", ErrInvalidInput)
		}

	case taskdomain.ScheduleTypeMonthly:
		if s.DayOfMonth == nil || *s.DayOfMonth < 1 || *s.DayOfMonth > 30 {
			return fmt.Errorf("%w: monthly schedule requires day_of_month between 1 and 30", ErrInvalidInput)
		}

	case taskdomain.ScheduleTypeSpecific:
		if len(s.Dates) == 0 {
			return fmt.Errorf("%w: specific schedule requires at least one date", ErrInvalidInput)
		}
		for _, d := range s.Dates {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return fmt.Errorf("%w: invalid date %q, expected YYYY-MM-DD format", ErrInvalidInput, d)
			}
		}

	case taskdomain.ScheduleTypeParity:
		if s.Parity == nil || !s.Parity.Valid() {
			return fmt.Errorf("%w: parity schedule requires parity to be %q or %q", ErrInvalidInput, taskdomain.ParityEven, taskdomain.ParityOdd)
		}
	}

	return nil
}

func scheduleInputToModel(input *ScheduleInput) *taskdomain.Schedule {
	if input == nil {
		return nil
	}

	return &taskdomain.Schedule{
		Type:       input.Type,
		Interval:   input.Interval,
		DayOfMonth: input.DayOfMonth,
		Dates:      input.Dates,
		Parity:     input.Parity,
	}
}
