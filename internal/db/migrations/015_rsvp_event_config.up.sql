-- 015: RSVP event configuration flags
ALTER TABLE training_series ADD COLUMN rsvp_opt_out INTEGER NOT NULL DEFAULT 0;
ALTER TABLE training_series ADD COLUMN rsvp_require_reason INTEGER NOT NULL DEFAULT 1;
ALTER TABLE training_sessions ADD COLUMN rsvp_opt_out INTEGER NOT NULL DEFAULT 0;
ALTER TABLE training_sessions ADD COLUMN rsvp_require_reason INTEGER NOT NULL DEFAULT 1;
ALTER TABLE games ADD COLUMN rsvp_opt_out INTEGER NOT NULL DEFAULT 0;
ALTER TABLE games ADD COLUMN rsvp_require_reason INTEGER NOT NULL DEFAULT 1;
