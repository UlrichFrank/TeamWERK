-- SQLite hat keine ALTER TABLE DROP COLUMN (vor 3.35), daher müssen wir die Tabelle rebuilden
-- Temporäre Tabelle erstellen ohne die neuen Spalten
CREATE TABLE duty_types_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  hours_value REAL NOT NULL,
  cash_substitute REAL,
  default_anchor TEXT NOT NULL,
  default_offset_minutes INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Daten von der alten in die neue Tabelle kopieren
INSERT INTO duty_types_new (id, name, hours_value, cash_substitute, default_anchor, default_offset_minutes, created_at)
SELECT id, name, hours_value, cash_substitute, default_anchor, default_offset_minutes, created_at
FROM duty_types;

-- Alte Tabelle löschen
DROP TABLE duty_types;

-- Neue Tabelle umbenennen
ALTER TABLE duty_types_new RENAME TO duty_types;
