package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const taskQuery = `
		INSERT INTO tasks (title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, status, created_at, updated_at
	`

	row := tx.QueryRow(ctx, taskQuery,
		task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt,
	)
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	if task.Schedule != nil {
		sched, err := insertSchedule(ctx, tx, created.ID, task.Schedule)
		if err != nil {
			return nil, err
		}
		created.Schedule = sched
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT
			t.id, t.title, t.description, t.status, t.created_at, t.updated_at,
			s.id, s.type, s.interval, s.day_of_month, s.dates, s.parity
		FROM tasks t
		LEFT JOIN schedules s ON s.task_id = t.id
		WHERE t.id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTaskWithSchedule(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const taskQuery = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			updated_at = $4
		WHERE id = $5
		RETURNING id, title, description, status, created_at, updated_at
	`

	row := tx.QueryRow(ctx, taskQuery,
		task.Title, task.Description, task.Status, task.UpdatedAt, task.ID,
	)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}

	if task.Schedule != nil {
		sched, err := upsertSchedule(ctx, tx, updated.ID, task.Schedule)
		if err != nil {
			return nil, err
		}
		updated.Schedule = sched
	} else {
		if err := deleteSchedule(ctx, tx, updated.ID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT
			t.id, t.title, t.description, t.status, t.created_at, t.updated_at,
			s.id, s.type, s.interval, s.day_of_month, s.dates, s.parity
		FROM tasks t
		LEFT JOIN schedules s ON s.task_id = t.id
		ORDER BY t.id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTaskWithSchedule(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func insertSchedule(ctx context.Context, tx pgx.Tx, taskID int64, s *taskdomain.Schedule) (*taskdomain.Schedule, error) {
	const query = `
		INSERT INTO schedules (task_id, type, interval, day_of_month, dates, parity)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, task_id, type, interval, day_of_month, dates, parity
	`
	row := tx.QueryRow(ctx, query,
		taskID, string(s.Type), s.Interval, s.DayOfMonth, s.Dates, parityToPtr(s.Parity),
	)
	return scanSchedule(row)
}

func upsertSchedule(ctx context.Context, tx pgx.Tx, taskID int64, s *taskdomain.Schedule) (*taskdomain.Schedule, error) {
	const query = `
		INSERT INTO schedules (task_id, type, interval, day_of_month, dates, parity)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (task_id) DO UPDATE SET
			type         = EXCLUDED.type,
			interval     = EXCLUDED.interval,
			day_of_month = EXCLUDED.day_of_month,
			dates        = EXCLUDED.dates,
			parity       = EXCLUDED.parity
		RETURNING id, task_id, type, interval, day_of_month, dates, parity
	`
	row := tx.QueryRow(ctx, query,
		taskID, string(s.Type), s.Interval, s.DayOfMonth, s.Dates, parityToPtr(s.Parity),
	)
	return scanSchedule(row)
}

func deleteSchedule(ctx context.Context, tx pgx.Tx, taskID int64) error {
	_, err := tx.Exec(ctx, `DELETE FROM schedules WHERE task_id = $1`, taskID)
	return err
}

func parityToPtr(p *taskdomain.ParityType) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task   taskdomain.Task
		status string
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	return &task, nil
}

func scanTaskWithSchedule(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task       taskdomain.Task
		status     string
		schedID    *int64
		schedType  *string
		interval   *int
		dayOfMonth *int
		dates      []string
		parity     *string
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
		&schedID,
		&schedType,
		&interval,
		&dayOfMonth,
		&dates,
		&parity,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	if schedID != nil {
		sched := &taskdomain.Schedule{
			ID:         *schedID,
			TaskID:     task.ID,
			Type:       taskdomain.ScheduleType(*schedType),
			Interval:   interval,
			DayOfMonth: dayOfMonth,
			Dates:      dates,
		}
		if parity != nil {
			p := taskdomain.ParityType(*parity)
			sched.Parity = &p
		}
		task.Schedule = sched
	}

	return &task, nil
}

func scanSchedule(scanner taskScanner) (*taskdomain.Schedule, error) {
	var (
		sched     taskdomain.Schedule
		schedType string
		parity    *string
	)

	if err := scanner.Scan(
		&sched.ID,
		&sched.TaskID,
		&schedType,
		&sched.Interval,
		&sched.DayOfMonth,
		&sched.Dates,
		&parity,
	); err != nil {
		return nil, err
	}

	sched.Type = taskdomain.ScheduleType(schedType)
	if parity != nil {
		p := taskdomain.ParityType(*parity)
		sched.Parity = &p
	}

	return &sched, nil
}
