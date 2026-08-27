-- Dienst-Dauer: duty_types.hours_value ist ab hier die DAUER eines Dienstes,
-- nicht nur seine Gutschrift — eine Zahl, aus der sowohl die angezeigte
-- Zeitspanne (8:00–9:00) als auch die Anrechnung auf duty_accounts.ist folgt.
-- Damit die Dauer pro Vorlage und pro Slot abweichen kann, bekommt die Größe
-- zwei weitere Ebenen. Einheit bleibt REAL in Stunden wie am Typ, damit die
-- bestehende SUM()-Aggregation ohne Umrechnung darauf zeigen kann.
ALTER TABLE game_template_items ADD COLUMN hours_value REAL NOT NULL DEFAULT 1.0;
ALTER TABLE duty_slots          ADD COLUMN hours_value REAL NOT NULL DEFAULT 1.0;

-- Backfill wendet Copy-on-pick rückwirkend an: jede Bestandszeile bekommt den
-- Wert des Diensttyps, den sie heute effektiv hätte. Keine Zeile bleibt auf dem
-- 1.0-Default stehen — duty_type_id ist in beiden Tabellen NOT NULL REFERENCES.
UPDATE game_template_items
   SET hours_value = (SELECT hours_value FROM duty_types WHERE id = duty_type_id);
UPDATE duty_slots
   SET hours_value = (SELECT hours_value FROM duty_types WHERE id = duty_type_id);
