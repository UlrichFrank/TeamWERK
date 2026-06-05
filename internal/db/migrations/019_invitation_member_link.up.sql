ALTER TABLE invitation_tokens ADD COLUMN member_id INTEGER REFERENCES members(id) ON DELETE SET NULL;
