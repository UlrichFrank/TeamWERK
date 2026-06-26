-- Account-Lockout: aufeinanderfolgende Login-Fehlversuche und optionale Sperre.
-- failed_login_count wird bei jedem Fehlversuch erhöht und bei Erfolg auf 0 gesetzt.
-- locked_until (ISO-Timestamp) sperrt den Login bis zu diesem Zeitpunkt.
ALTER TABLE users ADD COLUMN failed_login_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN locked_until TEXT;
