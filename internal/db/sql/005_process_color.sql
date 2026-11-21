ALTER TABLE process ADD COLUMN color TEXT NOT NULL DEFAULT 'cyan' CHECK (color IN ('cyan', 'orange', 'green', 'lavender'));
