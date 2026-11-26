-- 人材の責任者権限と、案件ごとの責任者紐づけ。

ALTER TABLE person ADD COLUMN is_manager INTEGER NOT NULL DEFAULT 0;

CREATE TABLE project_manager (
    project_id TEXT NOT NULL,
    person_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    CONSTRAINT project_manager_pk PRIMARY KEY (project_id, person_id),
    CONSTRAINT project_manager_project_fk FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE,
    CONSTRAINT project_manager_person_fk FOREIGN KEY (person_id) REFERENCES person (id) ON DELETE CASCADE
);

CREATE INDEX project_manager_person_idx ON project_manager (person_id);
