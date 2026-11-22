CREATE TABLE process_allocation (
    project_process_id TEXT NOT NULL,
    year_month TEXT NOT NULL,
    decade INTEGER NOT NULL,
    planned_minutes BIGINT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT process_allocation_pk PRIMARY KEY (project_process_id, year_month, decade),
    CONSTRAINT process_allocation_decade_check CHECK (decade IN (1, 2, 3)),
    CONSTRAINT process_allocation_minutes_check CHECK (planned_minutes >= 0),
    CONSTRAINT process_allocation_pp_fk FOREIGN KEY (project_process_id) REFERENCES project_process (id) ON DELETE CASCADE
);
CREATE INDEX idx_process_allocation_ym ON process_allocation (year_month, decade);

INSERT INTO process_allocation (project_process_id, year_month, decade, planned_minutes, created_at, updated_at)
SELECT
    pt.project_process_id,
    ta.year_month,
    ta.decade,
    SUM(ta.planned_minutes),
    MIN(ta.created_at),
    MAX(ta.updated_at)
FROM task_allocation ta
JOIN project_task pt ON pt.id = ta.task_id
GROUP BY pt.project_process_id, ta.year_month, ta.decade
HAVING SUM(ta.planned_minutes) > 0;
