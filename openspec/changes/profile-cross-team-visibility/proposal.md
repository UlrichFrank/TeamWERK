## Why

Bei generischen Terminen mit mehreren Mannschaften (z.B. Vereinsfest, gemeinsame Saisoneröffnung) zeigt die Termin-Detailseite `/termine/ereignis/{id}` heute jedem Teilnehmer **alle** Spieler **aller** beteiligten Teams mit Namen und RSVP-Status. Das ist für viele Mitglieder zu viel Sichtbarkeit (Datenschutz, „Schulfreunde im Nachbarverein sollen meine Spielzusagen nicht sehen").

Die Anzeige der eigenen Team-Mitglieder bleibt unverändert; die Sichtbarkeit der Mitglieder **fremder** Teams im selben Event soll auf **Opt-In pro Mitglied** umgestellt werden.

## What Changes

- **Migration:** Neue Spalte `members.cross_team_visible INTEGER NOT NULL DEFAULT 0`.
- **Backend `GET /api/games/{id}/participants`:** Bei Multi-Team-Events filtert die Antwort für Caller ohne Funktion (admin/trainer/sportliche_leitung/vorstand) auf:
  1. alle Teilnehmer aus den Teams, in denen der Caller selbst (oder eines seiner Kinder) im Kader oder erweiterten Kader steht (= "meine Teams im Event"), plus
  2. Teilnehmer aus fremden Teams, deren Member `cross_team_visible=1` hat.
- **Funktionsträger (admin / trainer / sportliche_leitung / vorstand) sehen weiterhin alles** (für Aufstellung, Organisation, Lineup-Management). Die bisherige Logik bleibt für sie unverändert.
- **Neuer Profil-Tab "Datenschutz"** in `/profil`, einsortiert **vor "Sonstiges"**, sichtbar nur wenn der eingeloggte Nutzer ein eigenes Member hat:
  - Toggle „Sichtbarkeit für Mitglieder" für das eigene Member (direktes Speichern, kein Draft-Workflow — es ist eine persönliche Privacy-Präferenz).
  - Read-only Anzeige der DSGVO-Einwilligungen (Verarbeitung / Weitergabe inkl. Datum), gleiches visuelles Control wie `MemberDatenschutzTab` im Admin (`/mitglieder/{id}`), aber gesperrt.
- **Member-Detailseite `/mitglieder/{id}` (Admin / Familienzugang):** Der bestehende `MemberDatenschutzTab` wird um den `cross_team_visible`-Toggle ergänzt — so können Eltern die Sichtbarkeit ihrer Kinder dort einstellen (Eltern haben über den Familienzugang auf das Kind-Profil Zugriff).
- **Frontend `TermineDetailPage`:**
  - Counter-Badges (✓ / ✗ / ?) aggregieren nur über die **sichtbaren** Zeilen.
  - Wenn in einer Team-Sektion gefilterte Mitglieder weggelassen wurden, erscheint ein dezenter Hinweis am Ende der Sektion: **„Weitere Mitglieder nicht sichtbar"** (ohne Zahl — die Anzahl würde wieder Information preisgeben).
  - Sektionen, in denen **kein** Mitglied sichtbar ist, werden weggelassen — kein leerer Header.

## Capabilities

### Added Capabilities

- `profile-datenschutz-tab`: Neuer Profil-Tab „Datenschutz" mit Sichtbarkeits-Toggle und read-only DSGVO-Anzeige.

### Modified Capabilities

- `spiel-teilnahme`: `GET /api/games/{id}/participants` filtert bei Multi-Team-Events fremde Team-Mitglieder ohne Opt-In für Caller ohne Funktion.

## Test-Anforderungen

| Route / Capability | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `GET /api/games/{id}/participants` | `TestGetParticipants_MultiTeam_SpielerSiehtNurEigenesTeam` | Spieler in Team A bei Event mit Teams A+B: sieht nur Member aus Team A in der Response. |
| `GET /api/games/{id}/participants` | `TestGetParticipants_MultiTeam_OptInMacht FremdSichtbar` | Member aus Team B mit `cross_team_visible=1` erscheint zusätzlich für Spieler aus Team A. |
| `GET /api/games/{id}/participants` | `TestGetParticipants_MultiTeam_ElternSehenTeamsDerKinder` | Elternteil eines Kindes in Team A sieht Team-A-Mitglieder (analog Spieler), plus Opt-In-Mitglieder fremder Teams. |
| `GET /api/games/{id}/participants` | `TestGetParticipants_MultiTeam_KindIn2TeamsSiehtBeide` | Member, das im Kader von Team A UND Team B steht, sieht beide Sektionen vollständig. |
| `GET /api/games/{id}/participants` | `TestGetParticipants_MultiTeam_TrainerSiehtAlles` | Caller mit `trainer`-Funktion sieht alle Member aller Teams (kein Filter). |
| `GET /api/games/{id}/participants` | `TestGetParticipants_MultiTeam_VorstandSiehtAlles` | Caller mit `vorstand`-Funktion sieht alle Member aller Teams. |
| `GET /api/games/{id}/participants` | `TestGetParticipants_SingleTeam_KeinFilter` | Bei nur einem Team in `game_teams` erfolgt KEIN Filter, unabhängig von `cross_team_visible`. |
| `PUT /api/members/{id}` | `TestUpdateMember_CrossTeamVisible_EigenesMember` | Eingeloggter Nutzer kann `cross_team_visible` auf seinem eigenen Member direkt setzen (kein Draft). |
| `PUT /api/members/{id}` | `TestUpdateMember_CrossTeamVisible_EigenesKindAlsElternteil` | Elternteil kann `cross_team_visible` auf dem Kind-Member direkt setzen. |
| `PUT /api/members/{id}` | `TestUpdateMember_CrossTeamVisible_Fremd_403` | Standard-Nutzer ohne Eltern-Beziehung kann das Flag auf einem fremden Member NICHT setzen → 403. |

**Garantierte Invariante:** In `GET /api/games/{id}/participants` ist ein Member aus einem Fremdteam genau dann enthalten, wenn (a) `cross_team_visible=1` **ODER** (b) der Caller selbst Funktion `admin|trainer|sportliche_leitung|vorstand` hat **ODER** (c) der Caller bzw. eines seiner Kinder Mitglied desselben Teams ist. Eigene Team-Member sind immer sichtbar.

## Impact

- **Migration:** `internal/db/migrations/003_member_cross_team_visible.up.sql` (+ `.down.sql`) — Spalte `members.cross_team_visible INTEGER NOT NULL DEFAULT 0`.
- **Backend:**
  - `internal/games/handler.go` (`GetParticipants`) — Query um Funktions-/Team-Filter erweitern, Fremd-Team-Member nur bei `cross_team_visible=1`.
  - `internal/members/handler.go` (`UpdateMember`) — `cross_team_visible` ins Update-DTO und ins SQL aufnehmen, ohne Draft.
- **Frontend:**
  - `web/src/pages/ProfilePage.tsx` — neuer Tab `'datenschutz'`, Reihenfolge `… banking, kalender, datenschutz, misc`.
  - `web/src/components/profile/ProfileDatenschutzTab.tsx` (NEU) — Toggle + DSGVO-Read-Only-Block (Komponente aus `MemberDatenschutzTab` als read-only Variante extrahiert oder dupliziert).
  - `web/src/components/admin/MemberDatenschutzTab.tsx` — bestehenden Tab um Toggle ergänzen (gleiche Komponente wiederverwendbar).
  - `web/src/pages/TermineDetailPage.tsx` — Counter über sichtbare Rows aggregieren, „Weitere Mitglieder nicht sichtbar"-Footer pro Sektion, leere Sektionen weglassen.
- **Default `0` für Bestandsmitglieder:** Mit dem Deploy werden alle Cross-Team-Sektionen schlagartig auf "nur eigenes Team" reduziert. Bewusst akzeptiert (Privacy-by-default).
- **Counter divergieren** zwischen Funktionsträgern und Spielern. Bewusst akzeptiert.
- **Push-Notifications:** Unverändert in diesem Proposal. Die Push-Empfänger werden bereits per Team-Kader bestimmt; eine grundlegende Synchronisation kommt mit `event-team-visibility` (separates Proposal).
- **Tests:** Fixtures vorhanden (`CreateMember`, `CreateKader`, `CreateGame`, …); neue Fixture-Helper `CreateFamilyLink` evtl. nötig (prüfen).
