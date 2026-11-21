-- 工程の背景色に桃色・黄色を足す（SQLite は CHECK を直せないので列を付け替える）
ALTER TABLE process ADD COLUMN color_new TEXT NOT NULL DEFAULT 'cyan' CHECK (color_new IN ('cyan', 'orange', 'green', 'lavender', 'rose', 'yellow'));
UPDATE process SET color_new = color;
ALTER TABLE process DROP COLUMN color;
ALTER TABLE process RENAME COLUMN color_new TO color;
