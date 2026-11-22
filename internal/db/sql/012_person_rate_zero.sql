CREATE TABLE person_rate_new (
    id TEXT NOT NULL PRIMARY KEY,
    person_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    from_date TEXT NOT NULL,
    hourly_yen BIGINT,
    monthly_yen BIGINT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT person_rate_unique UNIQUE (person_id, role_id, from_date),
    CONSTRAINT person_rate_yen_check CHECK (
        (hourly_yen IS NOT NULL OR monthly_yen IS NOT NULL)
        AND (hourly_yen IS NULL OR hourly_yen >= 0)
        AND (monthly_yen IS NULL OR monthly_yen >= 0)
    ),
    CONSTRAINT person_rate_person_fk FOREIGN KEY (person_id) REFERENCES person (id) ON DELETE CASCADE,
    CONSTRAINT person_rate_role_fk FOREIGN KEY (role_id) REFERENCES role (id)
);
INSERT INTO person_rate_new (
    id, person_id, role_id, from_date, hourly_yen, monthly_yen, created_at, updated_at
)
SELECT
    id, person_id, role_id, from_date,
    NULLIF(hourly_yen, 0), NULLIF(monthly_yen, 0),
    created_at, updated_at
FROM person_rate;
DROP TABLE person_rate;
ALTER TABLE person_rate_new RENAME TO person_rate;
CREATE INDEX idx_person_rate_lookup ON person_rate (person_id, role_id, from_date);
