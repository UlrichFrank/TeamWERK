## Why

Mitglieder von Team Stuttgart wollen ihre Spieltermine und Dienste in ihrem persönlichen Kalender (Google Calendar, Apple Calendar, Outlook) sehen — ohne manuellen Export. SpielerPlus bietet genau das per iCal-Feed; TeamWERK hat bislang keine Kalender-Export-Funktion.

## What Changes

- **Neues Package `internal/calendar`** mit iCal-Generierung (reines Go, kein externes Package) und Token-Management
- **Neue DB-Tabelle `calendar_tokens`** speichert pro User ein UUID-Token plus 5 Toggles (Heim, Auswärts, Training, Sonstiges, Dienste)
- **Migration 045**: nur `calendar_tokens` anlegen — die ursprünglich geplante Erweiterung von `games.event_type` entfällt, da Trainings bereits in der dedizierten Tabelle `training_sessions` leben
- **4 neue API-Routen**: öffentlicher Feed-Endpunkt (`/api/calendar/feed/{token}.ics`) + 3 Auth-Routen für Token-Verwaltung
- **Toggle „Training"**: Feed liest aus `training_sessions` (existierende Tabelle) — Heim/Auswärts/Sonstiges aus `games`, Dienste aus `duty_assignments`
- **Frontend**: neuer Bereich „Kalender-Abo" in den Nutzer-Einstellungen mit 5 Checkboxen, kopierbarem Link und „Link löschen"-Button

**Nicht im Scope:** Fahrgemeinschaften, Token-Regenerierung, Eltern sehen Kinder-Termine im eigenen Feed.

## Capabilities

### New Capabilities

- `ical-feed`: Persönlicher iCal-Feed pro Nutzer — Token-Verwaltung, Feed-Generierung, konfigurierbare Inhalte (5 Toggles: Heim-Spiele, Auswärts-Spiele, Training, Sonstige Events, Dienste)

## Impact

- **Backend**: neues Package `internal/calendar`; Migration 045 (nur `calendar_tokens`); Router-Erweiterung (1 Public-Route + 3 Auth-Routen)
- **Frontend**: neuer Tab/Bereich in `ProfilePage.tsx` für das Kalender-Abo
- **DB**: nur neue Tabelle `calendar_tokens`; `games`-Schema bleibt unverändert
- **Keine neuen Abhängigkeiten** — iCal ist Plain Text, UUID via `crypto/rand`
