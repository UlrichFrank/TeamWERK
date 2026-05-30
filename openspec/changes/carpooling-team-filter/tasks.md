## 1. DB-Migration: View user_accessible_teams

- [x] 1.1 Datei `internal/db/migrations/020_user_accessible_teams.up.sql` anlegen mit `CREATE VIEW user_accessible_teams AS ...` (UNION ALL der drei Rollen: spieler via kader_members, elternteil via family_links+kader_members, trainer via kader_trainers)
- [x] 1.2 Datei `internal/db/migrations/020_user_accessible_teams.down.sql` anlegen mit `DROP VIEW IF EXISTS user_accessible_teams`
- [x] 1.3 Migration lokal ausführen (`make migrate-up`) und View mit SQLite-Query prüfen

## 2. Backend: List()-Handler anpassen

- [x] 2.1 In `internal/carpooling/handler.go`, `List()`: aktive Season-ID aus DB lesen (`SELECT id FROM seasons WHERE is_active = 1 LIMIT 1`)
- [x] 2.2 Rollenprüfung einbauen: für `admin` und `vorstand` bleibt die Query ungefiltert (kein JOIN auf user_accessible_teams); für alle anderen Rollen `AND gt.team_id IN (SELECT team_id FROM user_accessible_teams WHERE user_id = ? AND season_id = ?)` ergänzen
- [x] 2.3 Optionalen `?team_id=X` Query-Parameter auslesen; wenn gesetzt und Nutzer kein admin/vorstand: zusätzlich `AND gt.team_id = ?` ergänzen (sicher: team_id muss bereits im user_accessible_teams-Filter liegen)
- [x] 2.4 Sicherstellen dass bei leerem Ergebnis eine leere `games`-Liste (nicht null) zurückgegeben wird

## 3. Frontend: Toggle „Alle" → „Team"

- [x] 3.1 In `web/src/pages/MitfahrgelegenheitenPage.tsx`: Button-Label `"Alle"` durch `"Team"` ersetzen (State `viewMine=false`)
- [x] 3.2 Leerstate-Text für „Team"-Ansicht prüfen: sinnvolle Meldung wenn keine Spiele vorhanden (z.B. „Keine Spiele deines Teams geplant.")

## 4. Verifikation

- [x] 4.1 Als `elternteil` ohne family_links einloggen → leere Liste, kein Fehler
- [x] 4.2 family_link und kader_member-Eintrag für Testnutzer anlegen (Ulrich → Jakob, Jakob in B-Jugend männlich Kader) → „JANO vs Team Stuttgart" erscheint in der Liste
- [x] 4.3 Als `admin` einloggen → alle Spiele weiterhin sichtbar
- [x] 4.4 `?team_id=15` Parameter testen: nur B-Jugend-Spiele; `?team_id=99` (nicht zugänglich) → leere Liste
- [x] 4.5 Dashboard-Carpooling-Hint zeigt „JANO vs Team Stuttgart" korrekt für Ulrich
- [x] 4.6 Toggle „Meine" zeigt weiterhin nur Spiele mit eigenem Eintrag; Tab-Counts korrekt
