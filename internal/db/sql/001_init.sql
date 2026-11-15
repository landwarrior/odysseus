CREATE TABLE schema_migration (
    version TEXT NOT NULL PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE person (
    id TEXT NOT NULL PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    name_kana TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    is_admin INTEGER NOT NULL,
    active INTEGER NOT NULL,
    remarks TEXT NOT NULL DEFAULT '',
    must_change_password INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT person_code_unique UNIQUE (code),
    CONSTRAINT person_flag_check CHECK (is_admin IN (0, 1) AND active IN (0, 1) AND must_change_password IN (0, 1))
);

CREATE TABLE role (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL,
    active INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT role_name_unique UNIQUE (name),
    CONSTRAINT role_active_check CHECK (active IN (0, 1))
);

CREATE TABLE process (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL,
    active INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT process_name_unique UNIQUE (name),
    CONSTRAINT process_active_check CHECK (active IN (0, 1))
);

CREATE TABLE person_rate (
    id TEXT NOT NULL PRIMARY KEY,
    person_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    from_date TEXT NOT NULL,
    hourly_yen BIGINT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT person_rate_unique UNIQUE (person_id, role_id, from_date),
    CONSTRAINT person_rate_yen_check CHECK (hourly_yen >= 0),
    CONSTRAINT person_rate_person_fk FOREIGN KEY (person_id) REFERENCES person (id) ON DELETE CASCADE,
    CONSTRAINT person_rate_role_fk FOREIGN KEY (role_id) REFERENCES role (id)
);

CREATE TABLE project (
    id TEXT NOT NULL PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    order_amount BIGINT,
    start_date TEXT,
    end_date TEXT,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT project_code_unique UNIQUE (code),
    CONSTRAINT project_status_check CHECK (status IN ('planning', 'active', 'closed')),
    CONSTRAINT project_amount_check CHECK (order_amount IS NULL OR order_amount >= 0)
);

CREATE TABLE project_process (
    id TEXT NOT NULL PRIMARY KEY,
    project_id TEXT NOT NULL,
    process_id TEXT NOT NULL,
    start_date TEXT,
    end_date TEXT,
    planned_minutes BIGINT NOT NULL,
    budget_yen BIGINT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT project_process_unique UNIQUE (project_id, process_id),
    CONSTRAINT project_process_minutes_check CHECK (planned_minutes >= 0),
    CONSTRAINT project_process_budget_check CHECK (budget_yen IS NULL OR budget_yen >= 0),
    CONSTRAINT project_process_project_fk FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE,
    CONSTRAINT project_process_process_fk FOREIGN KEY (process_id) REFERENCES process (id)
);

CREATE TABLE assignment (
    id TEXT NOT NULL PRIMARY KEY,
    project_process_id TEXT NOT NULL,
    person_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    planned_minutes BIGINT NOT NULL,
    planned_cost_yen BIGINT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT assignment_unique UNIQUE (project_process_id, person_id),
    CONSTRAINT assignment_minutes_check CHECK (planned_minutes >= 0),
    CONSTRAINT assignment_pp_fk FOREIGN KEY (project_process_id) REFERENCES project_process (id) ON DELETE CASCADE,
    CONSTRAINT assignment_person_fk FOREIGN KEY (person_id) REFERENCES person (id),
    CONSTRAINT assignment_role_fk FOREIGN KEY (role_id) REFERENCES role (id)
);

CREATE TABLE time_entry (
    id TEXT NOT NULL PRIMARY KEY,
    assignment_id TEXT NOT NULL,
    work_date TEXT NOT NULL,
    minutes BIGINT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT time_entry_unique UNIQUE (assignment_id, work_date),
    CONSTRAINT time_entry_minutes_check CHECK (minutes > 0),
    CONSTRAINT time_entry_assignment_fk FOREIGN KEY (assignment_id) REFERENCES assignment (id) ON DELETE CASCADE
);

CREATE INDEX idx_person_rate_lookup ON person_rate (person_id, role_id, from_date);
CREATE INDEX idx_project_status ON project (status);
CREATE INDEX idx_project_process_project ON project_process (project_id);
CREATE INDEX idx_assignment_pp ON assignment (project_process_id);
CREATE INDEX idx_assignment_person ON assignment (person_id);
CREATE INDEX idx_time_entry_assignment ON time_entry (assignment_id, work_date);
