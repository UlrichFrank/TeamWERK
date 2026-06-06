## Why

Trainer benötigen zwei eng zusammenhängende Werkzeuge: erstens eine Möglichkeit, Gelegenheitsspieler (z. B. jüngerer Jahrgang) dem Kader lose zuzuordnen, ohne sie als vollwertige Mitglieder zu führen; zweitens eine strukturierte Aufstellung pro Spiel, damit Spieler und Eltern sehen können, wer nominiert ist — bisher existiert dafür kein Feld.

## What Changes

- Neue Tabelle `kader_extended_members` — Gelegenheitsspieler werden je Kader gepflegt
- Neuer UI-Abschnitt „Erweiterter Kader" auf `/admin/kader` (gleiches Muster wie Trainer/Mitglieder: Suche + Liste + Entfernen)
- Neue Tabelle `game_lineup` — Trainer wählt pro Spiel, wer spielt
- Neuer Backend-Endpoint `GET /api/games/{id}/participants` — liefert alle `kader_members` + `kader_extended_members` des Teams mit RSVP- und Lineup-Status; löst die bisherige „nur Respondenten"-Ansicht auf der Spieldetail-Seite ab
- Neue Spalte „Aufstellung" in der Teilnahme-Tabelle auf `/termine/spiel/{id}`: Trainer editierbar, Spieler/Eltern read-only
- Erweiterte Kader-Mitglieder erhalten **kein RSVP** und erscheinen **nicht** in Training-Teilnahmelisten

## Capabilities

### New Capabilities

- `erweiterter-kader`: Verwaltung von Gelegenheitsspielern je Kader (DB + Admin-UI)
- `spiel-aufstellung`: Trainer-kuratierte Spielaufstellung je Spiel (DB + API + UI-Spalte)

### Modified Capabilities

- `spiel-teilnahme`: Spieldetail-Seite zeigt künftig alle Kader-Mitglieder (inkl. erweiterte), nicht nur Respondenten

## Impact

- **DB-Migrationen**: 2 neue Tabellen (`kader_extended_members`, `game_lineup`)
- **Backend**: `internal/kader/handler.go` (extended members CRUD), `internal/games/handler.go` (neuer `/participants`-Endpoint, Lineup-Endpoints)
- **Frontend**: `AdminKaderPage.tsx` (neuer Abschnitt), `TermineDetailPage.tsx` (neue Spalte + Datenquelle)
- **Kein Breaking Change** an bestehenden Endpoints; `/games/{id}/responses` bleibt erhalten
