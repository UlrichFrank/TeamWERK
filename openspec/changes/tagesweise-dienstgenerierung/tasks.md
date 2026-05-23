## 1. Backend — Neuer Endpoint

- [x] 1.1 Handler `RegenerateDaySlots` in `internal/games/handler.go` anlegen: Query-Parameter `date` und `season_id` einlesen, alle Spiele des Tages laden
- [x] 1.2 Pro Spiel Template auflösen: gespeichertes `template_id` oder `findTemplateForGame(isHome)` als Fallback; Spiele ohne Template überspringen
- [x] 1.3 Alle Spiele des Tages in einer DB-Transaktion verarbeiten: leere Slots löschen (`slots_filled=0`), neue Slots mit `applyBehavior`/`classifySlotPosition` erzeugen
- [x] 1.4 Response-Struktur: JSON mit `games`-Array (je `game_id`, `slots_created`, `kept_slots`, `skipped` wenn kein Template)
- [x] 1.5 Route `POST /api/admin/games/regenerate-day` in `cmd/teamwerk/main.go` unter `RequireRole("admin","trainer")` registrieren

## 2. Backend — Konflikt-Erkennung

- [x] 2.1 Nach dem Generieren aller Slots des Tages: Duplikate erkennen (gleicher `duty_type_id` + `event_time` für verschiedene `game_id`), in Response als `conflicts`-Array zurückgeben

## 3. Frontend — Dialog im Spielplan-Kalender

- [x] 3.1 In `SpielplanPage.tsx`: Tages-Klick-Handler erweitern — wenn Spiele an dem Tag vorhanden sind, neuen State `showDayRegen` + `dayRegenDate` setzen
- [x] 3.2 Dialog-Komponente im JSX: Listet alle Spiele des Tages mit Uhrzeit, Gegner und zugewiesenem Template auf
- [x] 3.3 „Generieren"-Button im Dialog ruft `POST /api/admin/games/regenerate-day` auf, zeigt Ladezustand
- [x] 3.4 Nach Erfolg: Spielplan neu laden (`loadGames()`), Erfolgsmeldung mit Anzahl erzeugter Slots einblenden
- [x] 3.5 Konflikte aus Response anzeigen: rote Warnmeldung mit Hinweis auf Optimierungsregeln

## 4. Qualitätssicherung

- [ ] 4.1 Manueller Test: Zwei Heimspiele am gleichen Tag — Abbau nach Spiel 1 und Aufbau vor Spiel 2 werden übersprungen (`same_day_behavior=skip`)
- [ ] 4.2 Manueller Test: Belegte Slots bleiben erhalten, `kept_slots` wird korrekt gezählt
- [ ] 4.3 Manueller Test: Tag ohne Spiele gibt leere Liste ohne Fehler zurück
- [ ] 4.4 Deploy auf VPS, Smoke-Test auf https://internal.team-stuttgart.org
