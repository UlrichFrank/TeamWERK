## 1. Datenbank

- [x] 1.1 Migration `015_rsvp_event_config.up.sql` anlegen: `ALTER TABLE training_series ADD COLUMN rsvp_opt_out INTEGER NOT NULL DEFAULT 0 CHECK(rsvp_opt_out IN (0,1))` + `rsvp_require_reason INTEGER NOT NULL DEFAULT 1 CHECK(rsvp_require_reason IN (0,1))`
- [x] 1.2 Migration `015_rsvp_event_config.up.sql` erweitern: gleiche zwei Spalten auf `training_sessions` (DEFAULT 0 / DEFAULT 1)
- [x] 1.3 Migration `015_rsvp_event_config.up.sql` erweitern: gleiche zwei Spalten auf `games` (DEFAULT 0 / DEFAULT 1)
- [x] 1.4 Migration `015_rsvp_event_config.down.sql` anlegen: Tabellen neu erstellen ohne die zwei Spalten (SQLite hat kein DROP COLUMN vor 3.35)
- [x] 1.5 Migration lokal ausführen und Spalten verifizieren (`make migrate-up`)

## 2. Backend — training_series und training_sessions

- [x] 2.1 Go-Struct für `trainingSeriesRow` um `RsvpOptOut` und `RsvpRequireReason` (int) ergänzen; `POST /api/training-series`-Handler nimmt beide Felder aus Request-Body entgegen und persistiert sie
- [x] 2.2 `PUT /api/training-series/{id}`-Handler nimmt `rsvp_opt_out` und `rsvp_require_reason` entgegen und aktualisiert sie (für zukünftige Sessions)
- [x] 2.3 `POST /api/training-sessions`-Handler: beim Anlegen Flags von der zugehörigen Serie lesen und auf Session kopieren; wenn kein `series_id` → Default 0/1 verwenden
- [x] 2.4 `PUT /api/training-sessions/{id}`-Handler: `rsvp_opt_out` und `rsvp_require_reason` explizit aus UPDATE-Statement ausschließen
- [x] 2.5 `sessionListItem`-Struct um `RsvpOptOut` und `RsvpRequireReason` ergänzen; beide Felder in `ListSessions`-Query selektieren
- [x] 2.6 `ListSessions`-Query: `confirmed_count`-Berechnung auf Opt-Out-Logik umstellen (CASE WHEN rsvp_opt_out=1 → Subquery für Mitglieder ohne Eintrag + explizit confirmed, sonst bisherige Logik)
- [x] 2.7 `ListSessions`: nach Query-Scan prüfen ob `rsvp_opt_out=1` und `my_rsvp=nil` → `my_rsvp = &"confirmed"` setzen

## 3. Backend — games

- [x] 3.1 `POST /api/admin/games`-Handler: `rsvp_opt_out` und `rsvp_require_reason` aus Request-Body entgegen nehmen; bei `event_type = 'generisch'` Default für `rsvp_require_reason = 0` setzen falls nicht angegeben
- [x] 3.2 `PUT /api/admin/games/{id}`-Handler: `rsvp_opt_out` und `rsvp_require_reason` nicht im UPDATE enthalten (eingefroren nach Anlegen)
- [x] 3.3 `gameListItem`-Struct um `RsvpOptOut` und `RsvpRequireReason` ergänzen; beide Felder in `ListMyGames`-Query selektieren
- [x] 3.4 `ListMyGames`-Query: `confirmed_count`-Berechnung analog zu 2.6 für games (Subquery via `team_memberships` + aktiver `season_id` des Spiels)
- [x] 3.5 `ListMyGames`: nach Query-Scan prüfen ob `rsvp_opt_out=1` und `my_rsvp=nil` → `my_rsvp = &"confirmed"` setzen

## 4. Frontend — TerminePage (Modal-Logik und Opt-Out-UI)

- [x] 4.1 `rsvp_opt_out` und `rsvp_require_reason` aus API-Response in Termin-Typ-Definitionen ergänzen
- [x] 4.2 Bestehende kaputte `reasons`/`setReasons`-State-Referenzen entfernen; `pendingRSVP` und `modalReason` States sicherstellen
- [x] 4.3 Absagen/Vielleicht-Handler für Training (eigene RSVP): wenn `rsvp_require_reason=1` → `pendingRSVP` setzen (Modal öffnen); wenn `rsvp_require_reason=0` → direkt `respondTraining` aufrufen
- [x] 4.4 Absagen/Vielleicht-Handler für Training (Kind-RSVP): analog zu 4.3
- [x] 4.5 Absagen/Vielleicht-Handler für Spiel (eigene RSVP + Kind-RSVP): analog zu 4.3/4.4
- [x] 4.6 Zusagen-Button: aktiv/highlighted darstellen wenn `my_rsvp === 'confirmed'`
- [x] 4.7 Zusagen-Button-Handler: klick sendet expliziten confirmed-Record auch wenn bereits implizit confirmed (idempotent)
- [x] 4.8 Inline-`<input>`-Felder für Begründung unter den RSVP-Buttons entfernen
- [x] 4.9 Modal-JSX implementieren: Overlay, Titel mit Aktion und ggf. Kindname, `<textarea>` gebunden an `modalReason`, OK-Button (disabled wenn leer), Abbrechen-Button — Styling nach CLAUDE.md-Konventionen
- [x] 4.10 Modal-OK-Handler: ruft `respondTraining`/`respondGame` mit `modalReason` auf, setzt `pendingRSVP = null` und `modalReason = ''`

## 5. Frontend — Anlegen-Formulare (Serie und Spiel)

- [x] 5.1 `AdminTrainingsPage.tsx` Serie-Anlegen-Formular: zwei Checkboxen hinzufügen — „Alle Spieler standardmäßig zugesagt" und „Begründung bei Absage erforderlich"; beide an Request-Body binden
- [x] 5.2 `AdminTrainingsPage.tsx` Serie-Bearbeiten-Formular: Checkboxen nur lesbar anzeigen (disabled) mit Hinweis „Gilt nur für neue Termine"
- [x] 5.3 Kalender-Wizard in `KalenderPage.tsx`: Spiel-Anlegen-Schritt um zwei Checkboxen ergänzen; bei `event_type = 'generisch'` ist `rsvp_require_reason` mit false vorbelegt
- [x] 5.4 Kalender-Wizard: Spiel-Bearbeiten schließt RSVP-Flags aus (Felder nicht anzeigen)

## 6. Cleanup und Verifikation

- [x] 6.1 `rsvp-reason-modal`-Change als ersetzt/obsolet markieren (`.openspec.yaml` Status auf `archived` setzen)
- [ ] 6.2 Manueller End-to-End-Test: Training-Serie mit opt-out anlegen → Session im Kalender prüfen (Zusagen vorausgewählt) → Absagen ohne Modal wenn require_reason=0 → Absagen mit Modal wenn require_reason=1 ← MANUELL
- [ ] 6.3 Manueller Test: Spiel (generisch) anlegen → require_reason Default = 0 prüfen → RSVP direkt ohne Modal ← MANUELL
- [ ] 6.4 Trainer-Detailansicht: confirmed_count bei opt-out prüfen (Spieler ohne Eintrag zählen mit) ← MANUELL
