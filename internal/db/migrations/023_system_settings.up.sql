-- system_settings: generisches Key-Value für zur Laufzeit toggelbare System-Zustände.
-- Erster Nutzer: maintenance_mode ('on'|'off'). Weitere Keys kommen ohne Schema-
-- Änderung dazu. Bewusst nicht getypt (Werte sind Strings); typspezifisches Parsen
-- macht der jeweilige Konsument.

CREATE TABLE IF NOT EXISTS system_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL
);

INSERT OR IGNORE INTO system_settings (key, value) VALUES ('maintenance_mode', 'off');
