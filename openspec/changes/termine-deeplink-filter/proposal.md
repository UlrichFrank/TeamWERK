## Why

Wenn ein Spieler oder Elternteil eine Push-Notification oder E-Mail zu einem neuen oder geänderten Termin erhält, landet er aktuell auf einer übersichtsartigen Seite (`/kalender`, `/training`, `/dienste`) und muss den fraglichen Termin selbst suchen, bevor er zu- oder absagen kann. Das verzögert die RSVP-Reaktion und führt nachweislich zu mehr „nicht beantworteten" Spielen. Zugleich speichert die `/termine`-Seite ihre Filter (Team, Termin-Typ, Vergangene) nur im React-State — Deep-Links und Browser-Back-Verhalten sind damit nicht möglich.

## What Changes

- Push-Notifications und E-Mail-Benachrichtigungen für **Spiele** und **Trainings** verlinken nicht mehr auf eine Übersichtsseite (`/kalender`, `/training`), sondern auf den konkreten Termin in `/termine` via Deep-Link mit Query-Parametern. Dienst-Notifications (`/dienste`) bleiben unverändert — Dienste sind keine „Termine" im Sinne der `/termine`-Seite.
- Die Termine-Seite (`/termine`) liest beim Mount Query-Parameter aus der URL und schreibt Filteränderungen zurück. Unterstützte Parameter:
  - `team` (Team-ID, einzelne Zahl)
  - `types` (kommaseparierte Liste: `training,heim,auswaerts`)
  - `past` (`1`/`0`, default `0`)
  - `focus` (Form `training-<id>` oder `game-<id>`) — scrollt die Karte in den Viewport und hebt sie kurz visuell hervor; aktiviert bei Bedarf automatisch „Vergangene anzeigen", falls der Termin in der Vergangenheit liegt
- Das `EventInfoModal` auf `/kalender` erhält einen zusätzlichen Button **„In Terminen öffnen"**, der zum entsprechenden Eintrag in `/termine?focus=<type>-<id>` springt.
- Spec-Anpassungen in den Push-Specs, sodass `url`-Parameter in `notify.Send` / `push.SendToUsers` auf `/termine?focus=…` zeigt statt auf die alten generischen Routen.

## Capabilities

### New Capabilities
_keine_

### Modified Capabilities
- `termine-unified-view`: Filterzustand wird über URL-Query-Parameter abgebildet und ist deep-linkbar; neuer `focus`-Parameter scrollt und hebt einen Termin hervor.
- `event-info-modal`: Neuer Button „In Terminen öffnen" navigiert zu `/termine?focus=<type>-<id>`.
- `push-games`: Notification-`url` zeigt auf `/termine?focus=game-<id>` statt `/kalender`.
- `push-trainings`: Notification-`url` zeigt auf `/termine?focus=training-<id>` statt `/training`.

## Impact

- **Backend:** `internal/games/handler.go`, `internal/trainings/handler.go`, `internal/duties/handler.go` — Anpassung der `notify.Send(...)`-URL-Parameter. Falls E-Mail-Body Hardcoded-Links enthält (Reminder-Mails in `internal/scheduler`, `duty-reminder-emails`), werden diese ebenfalls auf das neue Schema umgestellt.
- **Frontend:**
  - `web/src/pages/TerminePage.tsx` — `useSearchParams` von React Router v6 für Lesen/Schreiben des Filterzustands; Auto-Scroll + Highlight-Logik via `ref` und `useEffect`.
  - `web/src/components/EventInfoModal.tsx` (oder Pendant) — neuer Button.
- **Keine** neuen Migrationen, keine neuen API-Routen, kein neuer SQL-Spaltenzugriff.
- **Kein** Einfluss auf das Rollen-Modell; alle Rollen, die heute `/termine` sehen, verhalten sich identisch.
- **RAM-Footprint:** unverändert (nur Frontend-Logik + String-Änderungen im Backend).
