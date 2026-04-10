package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	Schedule    *Schedule `json:"schedule,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

type ScheduleType string

const (
	ScheduleTypeDaily    ScheduleType = "daily"
	ScheduleTypeMonthly  ScheduleType = "monthly"
	ScheduleTypeSpecific ScheduleType = "specific"
	ScheduleTypeParity   ScheduleType = "parity"
)

func (t ScheduleType) Valid() bool {
	switch t {
	case ScheduleTypeDaily, ScheduleTypeMonthly, ScheduleTypeSpecific, ScheduleTypeParity:
		return true
	default:
		return false
	}
}

type ParityType string

const (
	ParityEven ParityType = "even"
	ParityOdd  ParityType = "odd"
)

func (p ParityType) Valid() bool {
	return p == ParityEven || p == ParityOdd
}


type Schedule struct {
	ID         int64
	TaskID     int64
	Type       ScheduleType
	Interval   *int      
	DayOfMonth *int      
	Dates      []string   
	Parity     *ParityType 
}
