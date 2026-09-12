-- Betriebshärtung Welle 2 (design.md Entscheidung 4): Indizes für die
-- Abfragen, die am häufigsten über diese Spalten filtern (Dienst-Board je
-- Spiel, Regen-Konfliktprüfung je Datum+Saison, Familien-Auflösung,
-- Dienst-Bilanz je Nutzer). Rein additiv (CREATE INDEX IF NOT EXISTS) — kein
-- Datenverlust, falls der Lauf übersprungen oder wiederholt wird.
CREATE INDEX IF NOT EXISTS idx_duty_slots_game ON duty_slots(game_id);
CREATE INDEX IF NOT EXISTS idx_duty_slots_date_season ON duty_slots(event_date, season_id);
CREATE INDEX IF NOT EXISTS idx_family_links_member ON family_links(member_id);
CREATE INDEX IF NOT EXISTS idx_duty_assignments_user ON duty_assignments(user_id);
