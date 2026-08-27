-- Bei spielgebundenen Dienst-Slots war team_id nur eine Kopie von game_teams:
-- Kaderwechsel folgten in game_teams sofort, die kopierte team_id fror den
-- Anlage-Stand ein. Die Sichtbarkeit löst jetzt ausschließlich über
-- game_id -> game_teams auf, deshalb räumt diese Migration den Bestand auf.
-- team_id bleibt als Spalte für game-lose Slots (z.B. Vereinsfest) erhalten.
UPDATE duty_slots SET team_id = NULL WHERE game_id IS NOT NULL;
