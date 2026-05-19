ALTER TABLE members ADD COLUMN gender TEXT NOT NULL DEFAULT 'u' CHECK (gender IN ('m', 'f', 'u'));
