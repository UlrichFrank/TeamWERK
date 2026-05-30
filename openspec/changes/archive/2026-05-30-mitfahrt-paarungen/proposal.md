## Why

Das bestehende Mitfahrgelegenheiten-Board ist ein schwarzes Brett: Nutzer sehen wer fährt und wer mitfahren möchte, aber es gibt keine verbindliche Zusage. Paarungen entstehen außerhalb der App (Telefon, WhatsApp) und sind nicht nachvollziehbar. Mit verbindlichen Paarungen wird die gesamte Koordination in TeamWERK abgebildet.

## What Changes

- **Neue Tabelle `mitfahrt_paarungen`**: Speichert Paarungen zwischen einem Angebot (biete) und einem Gesuch (suche) mit Status `pending` / `confirmed` / `rejected`.
- **UNIQUE-Constraint gelockert**: `UNIQUE(game_id, user_id)` gilt künftig nur noch für `biete`-Einträge. Sucher können mehrere Gesuche pro Spiel anlegen (für manuelles Aufteilen einer Gruppe über mehrere Fahrer).
- **`plaetze` auch für Suche**: Das bestehende Feld wird nun auch für Gesuche genutzt — wie viele Plätze der Sucher benötigt.
- **Beidseitige Initiierung**: Sowohl Bieter als auch Sucher können eine Paarungsanfrage stellen. Die Gegenseite muss bestätigen.
- **Kapazitätsschutz**: Beim Anfragen wird geprüft, ob der Bieter noch genug freie Plätze hat. Ist die Kapazität erschöpft, wird die Anfrage sofort abgewiesen — weitere Anfragen an einen vollen Bieter sind nicht möglich.
- **Push-Benachrichtigungen bei Stornierung**: Löscht ein Bieter seinen Eintrag oder eine bestätigte Paarung, werden betroffene Sucher per Push informiert — und umgekehrt.
- **Paarungen öffentlich sichtbar**: Bestätigte Paarungen erscheinen im Board für alle Nutzer sichtbar.

## Capabilities

### New Capabilities

- `mitfahrt-paarungen`: Paarungssystem zwischen Mitfahrangeboten und Mitfahrgesuchen — Anfrage, Bestätigung, Ablehnung, Kapazitätsprüfung, Stornierungsbenachrichtigungen und öffentliche Anzeige.

### Modified Capabilities

- `mitfahrgelegenheiten-board`: Sucher können mehrere Einträge pro Spiel anlegen (UNIQUE nur noch für biete). `plaetze` wird auch für Gesuche genutzt. Bestätigte Paarungen werden im Board angezeigt.

## Impact

- **DB**: Migration 013 — neue Tabelle `mitfahrt_paarungen`, Anpassung des UNIQUE-Constraints auf `mitfahrgelegenheiten`
- **Backend**: Neue Endpunkte in `internal/carpooling/` für Paarungs-CRUD und Bestätigung; bestehende `Upsert`- und `Delete`-Handler anpassen
- **Frontend**: `MitfahrgelegenheitenPage.tsx` — Paarungs-UI in den GameCards, neue Anfrage/Bestätigungs-Flows
- **Notifications**: `internal/notifications/` — Push bei Bestätigung, Ablehnung, Stornierung
