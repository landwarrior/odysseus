-- 実績をタスク単位でも入れられるようにする（task_id が空なら従来どおり工程＝アサイン単位）
CREATE TABLE time_entry_new (
    id TEXT NOT NULL PRIMARY KEY,
    assignment_id TEXT NOT NULL,
    task_id TEXT,
    work_date TEXT NOT NULL,
    minutes BIGINT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT time_entry_minutes_check CHECK (minutes > 0),
    CONSTRAINT time_entry_assignment_fk FOREIGN KEY (assignment_id) REFERENCES assignment (id) ON DELETE CASCADE,
    CONSTRAINT time_entry_task_fk FOREIGN KEY (task_id) REFERENCES project_task (id) ON DELETE CASCADE
);

INSERT INTO time_entry_new (id, assignment_id, task_id, work_date, minutes, created_at, updated_at)
SELECT id, assignment_id, NULL, work_date, minutes, created_at, updated_at FROM time_entry;

DROP TABLE time_entry;
ALTER TABLE time_entry_new RENAME TO time_entry;

CREATE UNIQUE INDEX time_entry_unique_with_task
    ON time_entry (assignment_id, work_date, task_id) WHERE task_id IS NOT NULL;
CREATE UNIQUE INDEX time_entry_unique_no_task
    ON time_entry (assignment_id, work_date) WHERE task_id IS NULL;
CREATE INDEX idx_time_entry_assignment ON time_entry (assignment_id, work_date);
CREATE INDEX idx_time_entry_task ON time_entry (task_id);
