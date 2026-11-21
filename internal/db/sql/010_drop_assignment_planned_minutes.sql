CREATE TABLE assignment_new (
    id TEXT NOT NULL PRIMARY KEY,
    project_process_id TEXT NOT NULL,
    person_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    planned_cost_yen BIGINT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT assignment_unique UNIQUE (project_process_id, person_id),
    CONSTRAINT assignment_pp_fk FOREIGN KEY (project_process_id) REFERENCES project_process (id) ON DELETE CASCADE,
    CONSTRAINT assignment_person_fk FOREIGN KEY (person_id) REFERENCES person (id),
    CONSTRAINT assignment_role_fk FOREIGN KEY (role_id) REFERENCES role (id)
);
INSERT INTO assignment_new (
    id, project_process_id, person_id, role_id, start_date, end_date, planned_cost_yen, created_at, updated_at
)
SELECT
    id, project_process_id, person_id, role_id, start_date, end_date, planned_cost_yen, created_at, updated_at
FROM assignment;
DROP TABLE assignment;
ALTER TABLE assignment_new RENAME TO assignment;
CREATE INDEX idx_assignment_pp ON assignment (project_process_id);
CREATE INDEX idx_assignment_person ON assignment (person_id);
