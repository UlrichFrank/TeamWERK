-- 062_duty_assignment_comments: Freitext-Kommentar je Dienst-Zuteilung
-- (dienst-kommentare) — z.B. "welche Art Kuchen bringe ich mit".
--
-- Hängt bewusst an duty_assignments.id statt an (duty_slot_id, user_id):
-- UNIQUE(assignment_id) erzwingt "genau ein Kommentar pro Zuteilung" als
-- Nebenprodukt der Spalte, und ON DELETE CASCADE räumt den Kommentar bei
-- JEDEM Pfad auf, der eine Zuteilung löscht (Austragen, Massen-Regen,
-- Spiel-Löschung, Slot-Löschung übers Kalender-Modal) — ohne dass jeder
-- dieser Pfade den Kommentar einzeln mitlöschen müsste (design.md).
CREATE TABLE duty_assignment_comments (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    assignment_id INTEGER  NOT NULL UNIQUE REFERENCES duty_assignments(id) ON DELETE CASCADE,
    body          TEXT     NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME
);
