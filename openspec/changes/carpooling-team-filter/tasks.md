## 1. Backend: Team-Filter in carpooling/handler.go

- [ ] 1.1 Private Hilfsfunktion `teamQueryForUser(role string) string` in `internal/carpooling/handler.go` ergänzen — analog zur bestehenden Implementierung in `internal/dashboard/handler.go` (Subqueries für elternteil, spieler, trainer; leer für admin/vorstand)
- [ ] 1.2 In `List()`: aktive Season-ID aus DB lesen (`SELECT id FROM seasons WHERE is_active = 1 LIMIT 1`)
- [ ] 1.3 In `List()`: wenn `teamQuery != ""` (Rollen elternteil/spieler/trainer), Haupt-Query um `JOIN`-Bedingung auf `game_teams` und `WHERE team_id IN (teamQuery)` sowie `AND g.season_id = ?` erweitern; für admin/vorstand bleibt die Query unverändert
- [ ] 1.4 Sicherstellen, dass bei leerem `teamQueryForUser`-Ergebnis (z.B. keine family_links) eine leere Spielliste ohne Fehler zurückgegeben wird

## 2. Frontend: Toggle "Alle" → "Team"

- [ ] 2.1 In `web/src/pages/MitfahrgelegenheitenPage.tsx` den Button-Label `"Alle"` durch `"Team"` ersetzen (State `viewMine=false`)
- [ ] 2.2 Leerstate-Text für "Team"-Ansicht prüfen: falls keine Spiele vorhanden, passende Meldung anzeigen (z.B. „Keine Spiele deines Teams geplant.")

## 3. Verifikation

- [ ] 3.1 Als `elternteil` (Ulrich@diefranks.eu) einloggen: Carpooling-Liste zeigt nur Spiele von Jakobs Team — nach Anlage von family_link und kader_member-Eintrag
- [ ] 3.2 Als `admin` einloggen: alle Spiele weiterhin sichtbar
- [ ] 3.3 Toggle "Meine" zeigt nach wie vor nur Spiele mit eigenem Eintrag; Tab-Counts stimmen
- [ ] 3.4 Dashboard-Carpooling-Hint zeigt "JANO vs Team Stuttgart" korrekt für Ulrich (setzt korrekte Testdaten voraus)
