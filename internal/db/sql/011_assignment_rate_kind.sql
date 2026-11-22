ALTER TABLE assignment ADD COLUMN rate_kind TEXT NOT NULL DEFAULT 'hourly';
UPDATE assignment SET rate_kind = 'monthly' WHERE planned_cost_yen > 0;
