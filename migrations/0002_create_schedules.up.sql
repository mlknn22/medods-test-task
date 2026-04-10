CREATE TABLE IF NOT EXISTS schedules (
    id           BIGSERIAL PRIMARY KEY,
    task_id      BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    interval     INT,
    day_of_month INT,
    dates        TEXT[],
    parity       TEXT,
    UNIQUE(task_id)
);
