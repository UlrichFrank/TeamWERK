-- Kinderaccounts ohne E-Mail: zweiter Login-Schlüssel (Vorname.Nachname)
-- und Beitrittsantrag-Variante "Kinderaccount".

-- Login-Name als Alternative zur E-Mail. Vergleich case-insensitiv (LOWER),
-- eindeutig nur solange das Konto login-fähig ist (can_login=1) — analog zum
-- bestehenden users_email_login_unique.
ALTER TABLE users ADD COLUMN login_name TEXT;
CREATE UNIQUE INDEX users_login_name_unique ON users(LOWER(login_name))
    WHERE can_login = 1 AND login_name IS NOT NULL;

-- Beitrittsantrag: Kinderaccount-Flag + verwaltende Eltern-E-Mail (Korrespondenz).
ALTER TABLE membership_requests ADD COLUMN is_child INTEGER NOT NULL DEFAULT 0;
ALTER TABLE membership_requests ADD COLUMN parent_email TEXT;
