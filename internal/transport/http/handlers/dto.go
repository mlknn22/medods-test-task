package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

// scheduleInputDTO is the JSON shape for a schedule sent by the client.
type scheduleInputDTO struct {
	Type       string   `json:"type"`
	Interval   *int     `json:"interval,omitempty"`
	DayOfMonth *int     `json:"day_of_month,omitempty"`
	Dates      []string `json:"dates,omitempty"`
	Parity     *string  `json:"parity,omitempty"`
}

// taskMutationDTO is used for both POST (create) and PUT (update) requests.
type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	Schedule    *scheduleInputDTO `json:"schedule"`
}

// scheduleDTO is the JSON shape for a schedule returned by the API.
type scheduleDTO struct {
	ID         int64    `json:"id"`
	Type       string   `json:"type"`
	Interval   *int     `json:"interval,omitempty"`
	DayOfMonth *int     `json:"day_of_month,omitempty"`
	Dates      []string `json:"dates,omitempty"`
	Parity     *string  `json:"parity,omitempty"`
}

// taskDTO is the JSON shape for a task returned by the API.
type taskDTO struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	Schedule    *scheduleDTO      `json:"schedule,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	dto := taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}

	if task.Schedule != nil {
		s := task.Schedule
		sd := &scheduleDTO{
			ID:         s.ID,
			Type:       string(s.Type),
			Interval:   s.Interval,
			DayOfMonth: s.DayOfMonth,
			Dates:      s.Dates,
		}
		if s.Parity != nil {
			p := string(*s.Parity)
			sd.Parity = &p
		}
		dto.Schedule = sd
	}

	return dto
}

// scheduleInputToUsecase converts the HTTP-layer DTO to the usecase-layer input.
func scheduleInputToUsecase(dto *scheduleInputDTO) *taskusecase.ScheduleInput {
	if dto == nil {
		return nil
	}

	input := &taskusecase.ScheduleInput{
		Type:       taskdomain.ScheduleType(dto.Type),
		Interval:   dto.Interval,
		DayOfMonth: dto.DayOfMonth,
		Dates:      dto.Dates,
	}

	if dto.Parity != nil {
		p := taskdomain.ParityType(*dto.Parity)
		input.Parity = &p
	}

	return input
}
