-- Anleitung pro Dienst-Typ (Markdown, gerendert im Frontend).
-- Bilder werden über /dokumente/datei/{fileId} referenziert; kein separater
-- Bild-Endpoint und keine neue Berechtigungs-Achse (Konvention: Ordner
-- „Anleitungen" mit everyone/read).

ALTER TABLE duty_types ADD COLUMN instruction_md TEXT NOT NULL DEFAULT '';
ALTER TABLE duty_types ADD COLUMN instruction_updated_at TEXT;
ALTER TABLE duty_types ADD COLUMN instruction_updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL;
