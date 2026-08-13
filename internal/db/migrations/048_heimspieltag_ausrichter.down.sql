-- Rückweg additiv angelegt: kein Bestandsspiel referenziert diese Tabellen
-- oder Spalte über ihren Lebenszyklus hinaus zwingend (design.md, Migration
-- Plan "Rollback"). Verlorene Ausrichter-Zuordnungen sind Konfiguration,
-- keine Nutzerdaten.
--
-- Reihenfolge zwingend: erst die Spalte, die auf `ausrichter` zeigt, dann die
-- Tabelle, die ebenfalls auf `ausrichter` zeigt, zuletzt `ausrichter` selbst
-- (inkl. Index) — jede andere Reihenfolge ließe eine baumelnde FK-Referenz auf
-- eine bereits verschwundene Tabelle stehen.
ALTER TABLE game_template_items DROP COLUMN ausrichter_id;

DROP TABLE spieltag_ausrichter;

DROP INDEX idx_ausrichter_default;
DROP TABLE ausrichter;
