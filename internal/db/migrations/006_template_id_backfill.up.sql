-- Backfill: explizite Slot-Quelle pro Game.
-- Vor diesem Change löste der Auto-Regen bei games.template_id IS NULL auf das
-- Template mit der kleinsten ID des passenden template_type auf (ORDER BY id
-- LIMIT 1 in findTemplateForGameTx). Mit dem Change entfällt dieser Fallback;
-- NULL bedeutet "keine Auto-Dienste". Damit Bestands-Events nicht still ihre
-- Slots verlieren, wird der ehemalige Fallback-Wert hier einmalig in die Spalte
-- geschrieben. Generische Events bleiben NULL — sie nutzten den Fallback nie.

UPDATE games
SET template_id = (
    SELECT id FROM game_templates
    WHERE template_type = games.event_type
    ORDER BY id LIMIT 1
)
WHERE template_id IS NULL
  AND event_type IN ('heim','auswärts')
  AND EXISTS (
      SELECT 1 FROM game_templates
      WHERE template_type = games.event_type
  );
