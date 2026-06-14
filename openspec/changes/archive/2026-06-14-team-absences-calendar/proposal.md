## Why

Trainer sehen heute keine Abwesenheiten ihrer Spieler im Kalender — obwohl Spieler die Einstellung „Abwesenheiten für Trainer sichtbar" aktivieren können. Die Einstellung ist außerdem defekt (Lesen kaputt, Schreiben funktioniert). Trainer können so nicht planen, wer an einem Training oder Spiel wahrscheinlich fehlt.

## What Changes

- **Bugfix:** `Member`-Struct und `getMember()` werden um `absences_public` ergänzt, damit das Profil-Toggle korrekt gelesen und angezeigt wird
- **Backend:** `GET /api/absences/calendar` bekommt optionalen Parameter `show_team=true`; bei berechtigten Rollen (trainer, sportvorstand, vorstand, admin) werden zusätzlich Abwesenheiten von Teammitgliedern zurückgegeben, die `absences_public = 1` haben — gefiltert auf die eigenen Teams des Nutzers
- **Frontend Kalender:** Neuer Toggle „Mannschaftsabwesenheiten" (nur sichtbar für berechtigte Rollen); greift in den bestehenden Team-Filter ein; Filter-State in `sessionStorage`, Default AUS
- **Darstellung:** Eigene Abwesenheiten bleiben `brand-yellow`; Team-Abwesenheiten werden in `brand-blue` dargestellt; Name + Typ nur per Tooltip und per Click → Detailansicht

## Capabilities

### New Capabilities

- `team-absences-calendar`: Trainer können Mannschaftsabwesenheiten im Kalender ein-/ausblenden (session-basierter Toggle, rollengeschützt)

### Modified Capabilities

- `member-absences`: `absences_public` wird korrekt in `getMember()` gelesen und im Profil-Toggle angezeigt (Bugfix)
- `kalender-agenda-view`: Kalender unterscheidet eigene vs. fremde Abwesenheiten farblich und zeigt Personendetails nur per Tooltip/Click

## Impact

- **Backend:** `internal/members/handler.go` (Member-Struct, getMember), `internal/absences/handler.go` (Calendar-Endpoint)
- **Frontend:** `web/src/pages/KalenderPage.tsx`
- **Keine neuen Abhängigkeiten**
- **Keine Migration** (Spalte `absences_public` existiert seit Migration 030)
