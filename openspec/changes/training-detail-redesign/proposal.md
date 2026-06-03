## Why

Die TrainingsDetailPage zeigt Rückmeldungen und Anwesenheit in zwei getrennten Karten. Trainer müssen scrollen und haben zwei verschiedene Workflows. Anwesenheit erfordert einen expliziten „Speichern"-Klick. Die RSVP-Statistik ist dezent im Kartenheader versteckt. Das alles ist mehr Reibung als nötig — die Daten gehören zusammen.

## What Changes

- **Session-Header** erhält Stat-Badges: ✓/✗/? mit Zahlen, farbig, prominent; Trainer sehen zusätzlich die Anzahl ohne Rückmeldung
- **Zwei Karten → eine Karte** „Teilnahme" mit vereinter Tabelle: Spalten Name / RSVP / Anwesend
- **RSVP-Kommentare** erscheinen als Tooltip (Desktop: Hover, Mobile: Tap auf Kommentar-Icon); ein `MessageCircle`-Icon signalisiert, dass ein Kommentar existiert
- **Anwesenheits-Checkboxen** speichern sofort beim Toggle (kein Speichern-Button); bei Fehler wird die Checkbox zurückgesetzt und ein Fehler-Banner erscheint
- **Datenquelle vereinfacht**: Trainer nutzen ausschließlich `GET /attendances` (enthält bereits `rsvp_status`), nicht-Trainer `session.responses`

## Capabilities

### Modified Capabilities

- `training-rsvp`: Anzeige der RSVP-Übersicht; Stats jetzt als Badges im Session-Header; Kommentare als Tooltip statt Inline-Text
- `training-attendance`: Anwesenheitserfassung ohne Speichern-Button; immediate save mit Fehler-Feedback

## Impact

- **Nur Frontend-Änderungen:** `web/src/pages/TrainingsDetailPage.tsx`
- **Keine neuen Backend-Routen**, keine DB-Änderungen
- **Keine neuen Dependencies**
- **Rollen:** Trainer/Admin sehen vereinte Tabelle mit Anwesend-Spalte und No-RSVP-Badge; Spieler/Elternteil sehen nur RSVP-Spalte ohne Vollständige Mitgliederliste (Privacy unverändert)
