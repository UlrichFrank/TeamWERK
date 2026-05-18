-- Mitgliedsnummer hinzufügen (nullable, unique nur für nicht-NULL Werte)
ALTER TABLE members ADD COLUMN member_number TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_members_member_number
    ON members(member_number) WHERE member_number IS NOT NULL;
