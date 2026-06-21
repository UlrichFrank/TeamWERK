## Why

Heute sind Events (Spiele, generische Termine) für alle eingeloggten Nutzer in Listen, Kalender und Dashboard sichtbar — auch wenn weder der Nutzer noch eines seiner Kinder zu einem der beteiligten Teams gehört. Konsequenz:

- Spieler aus Team A sehen den Vereinsfest-Termin von Teams B+C+D im Kalender.
- Push-Notifications zu Events fremder Teams erreichen Nutzer, die nichts damit zu tun haben.
- Direkter URL-Zugriff auf `/api/games/{id}` liefert Details fremder Events.

Sichtbarkeit von Events SHALL strikt an Team-Zugehörigkeit gekoppelt werden. Funktionsträger (admin/trainer/sportliche_leitung/vorstand) bleiben global sichtbar — sie brauchen die organisatorische Übersicht.

**Voraussetzung:** Dieses Proposal baut auf `profile-cross-team-visibility` auf (gleiche „Meine Teams im Event"-Definition).

## What Changes

- **Backend-Filter** in allen Listen- und Detail-Routen, die Events ausliefern:
  - `GET /api/games` — nur Games, deren `game_teams` ∩ „meine Teams in Saison" ≠ ∅.
  - `GET /api/games/{id}` — 404, wenn Game nicht in „meine Events".
  - `GET /api/dashboard` (Block „Nächste Termine", „Offene Mitfahr-Gesuche", …) — Events nur in eigener Auswahl.
  - `GET /api/calendar` (bzw. Quelle des Kalender-Views) — analog.
  - `GET /api/games/{id}/participants` — 404, wenn Game nicht sichtbar (zusätzlich zum Cross-Team-Filter aus `profile-cross-team-visibility`).
  - `GET /api/games/{id}/duty-slots`, `GET /api/games/{id}/lineup` etc. — analog, 404.
- **Mitfahrgelegenheiten:** `POST/GET /api/mitfahrgelegenheiten` und Paarungs-Routen werfen 404, wenn das referenzierte Game nicht sichtbar ist.
- **Push-Notifications:** `notifications`-Empfänger-Auswahl SHALL über dieselbe Funktion `usersWithAccessToGame(gameID)` laufen. Nutzer, die das Event nicht sehen dürfen, erhalten KEINE Push zum Event (Erstellung, Änderung, Absage, Carpooling).
- **Funktionsträger-Bypass:** Caller mit `admin`, `trainer`, `sportliche_leitung` oder `vorstand` sehen alle Events ohne Filter — sowohl in Listen als auch per Direkt-ID.
- **Trainings:** Bleiben unverändert (per Definition single-team, bereits team-gefiltert).
- **Gemeinsame Helper-Funktion** `auth.UserCanSeeGame(ctx, db, userID, gameID) (bool, error)` und `auth.GameIDsVisibleToUser(ctx, db, userID, seasonID) ([]int, error)` zur Wiederverwendung über alle Domains.

## Capabilities

### Added Capabilities

- `event-team-visibility`: Globale Sichtbarkeitsregel für Events basierend auf Team-Zugehörigkeit (mit Funktionsträger-Bypass), inkl. Push-Synchronisation.

### Modified Capabilities

- `spiel-teilnahme`: `GET /api/games/{id}/participants` liefert 404 statt 200 mit leerer Liste, wenn der Caller das Game nicht sehen darf.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `GET /api/games` | `TestListGames_FilterEigeneTeams` | Nur Games mit Schnittmenge zu „meine Teams in Saison". |
| `GET /api/games` | `TestListGames_TrainerSiehtAlle` | Caller mit `trainer` sieht alle Games. |
| `GET /api/games` | `TestListGames_ElternSiehtTeamsDerKinder` | Elternteil sieht Games der Teams seiner Kinder. |
| `GET /api/games/{id}` | `TestGetGame_FremdEvent_404` | Standard-Nutzer ohne Team-Bezug → 404. |
| `GET /api/games/{id}` | `TestGetGame_EigenesEvent_200` | Standard-Nutzer mit Team-Bezug → 200. |
| `GET /api/games/{id}/participants` | `TestGetParticipants_FremdEvent_404` | Standard-Nutzer ohne Team-Bezug → 404 (statt 200 + leere Liste). |
| `GET /api/dashboard` | `TestDashboard_NaechsteTermine_Filter` | Nur Events mit Team-Bezug erscheinen im Dashboard. |
| `GET /api/calendar` | `TestCalendar_Filter` | Nur Events mit Team-Bezug erscheinen im Kalender. |
| `POST /api/mitfahrgelegenheiten` | `TestCarpooling_FremdGame_404` | Anlegen eines Gesuchs zu fremdem Game → 404. |
| `notifications.SendToUsersForGame` | `TestPush_FremdEventKeinEmpfaenger` | Nutzer ohne Team-Bezug ist KEIN Push-Empfänger. |
| `notifications.SendToUsersForGame` | `TestPush_TrainerImmerEmpfaenger` | Trainer ist Push-Empfänger auch ohne Team-Bezug. |

**Garantierte Invariante:** Ein Nutzer N sieht ein Event E (in jeder Liste, jedem Detail, jeder Push-Notification) genau dann, wenn (a) N selbst oder ein Kind von N Mitglied eines Teams in `game_teams(E)` für die `season_id` von E ist, ODER (b) N hat Funktion `admin|trainer|sportliche_leitung|vorstand`.

## Impact

- **Neuer Helper:** `internal/auth/event_visibility.go` mit `UserCanSeeGame` und `GameIDsVisibleToUser`. Tests in `event_visibility_test.go`.
- **Backend-Routen:**
  - `internal/games/handler.go` — alle `GET`-Routen filtern; alle `GET /games/{id}/...`-Routen schicken 404 statt 200 bei fehlender Sichtbarkeit.
  - `internal/dashboard/handler.go` — alle Game-bezogenen Blöcke nutzen `GameIDsVisibleToUser`.
  - `internal/calendar/handler.go` (falls eigene Datei) — analog.
  - `internal/carpooling/handler.go` (bzw. `mitfahrten/handler.go`) — Game-Existenz-Check ersetzt durch Sichtbarkeits-Check.
  - `internal/notifications/` bzw. push-Caller in `games`, `carpooling`, `duties`: Empfänger-Set kommt aus zentralem Helper.
- **Datenmenge:** Listen werden kleiner (für Nicht-Funktionsträger). Performance-Risiko gering, da Filter über bereits indizierte FKs (`game_teams`, `kader`, `kader_members`).
- **Keine Frontend-Änderungen** zwingend nötig — der Filter passiert serverseitig. Optional: UI-Hinweis "Keine Events gefunden" verbessern.
- **Schema:** Keine neue Migration.
- **Bypass für Funktionsträger** ist organisatorisch notwendig — wird in `design.md` begründet.
- **Tests:** Querschnitt durch viele Pakete. Erwartet ~10–15 neue Tests; einige bestehende Tests müssen Fixtures um Team-Mitgliedschaft des Test-Nutzers ergänzen, damit sie weiterhin grün bleiben (sonst 404).
