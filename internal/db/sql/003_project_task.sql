CREATE TABLE project_task (
    id TEXT NOT NULL PRIMARY KEY,
    project_process_id TEXT NOT NULL,
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL,
    planned_minutes BIGINT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT project_task_minutes_check CHECK (planned_minutes >= 0),
    CONSTRAINT project_task_pp_fk FOREIGN KEY (project_process_id) REFERENCES project_process (id) ON DELETE CASCADE
);

CREATE TABLE task_allocation (
    task_id TEXT NOT NULL,
    year_month TEXT NOT NULL,
    decade INTEGER NOT NULL,
    planned_minutes BIGINT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT task_allocation_pk PRIMARY KEY (task_id, year_month, decade),
    CONSTRAINT task_allocation_decade_check CHECK (decade IN (1, 2, 3)),
    CONSTRAINT task_allocation_minutes_check CHECK (planned_minutes >= 0),
    CONSTRAINT task_allocation_task_fk FOREIGN KEY (task_id) REFERENCES project_task (id) ON DELETE CASCADE
);

CREATE INDEX idx_project_task_pp ON project_task (project_process_id, sort_order);
CREATE INDEX idx_task_allocation_ym ON task_allocation (year_month, decade);
