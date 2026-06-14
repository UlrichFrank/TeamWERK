ALTER TABLE users ADD COLUMN maps_provider TEXT NOT NULL DEFAULT 'auto' CHECK(maps_provider IN ('auto','google','apple'));
