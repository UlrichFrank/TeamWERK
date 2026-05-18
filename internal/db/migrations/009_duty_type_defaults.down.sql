PRAGMA foreign_keys=OFF;
CREATE TABLE duty_types_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  hours_value REAL NOT NULL DEFAULT 1.0,
  cash_substitute REAL
);
INSERT INTO duty_types_new SELECT id, name, hours_value, cash_substitute FROM duty_types;
DROP TABLE duty_types;
ALTER TABLE duty_types_new RENAME TO duty_types;
PRAGMA foreign_keys=ON;
