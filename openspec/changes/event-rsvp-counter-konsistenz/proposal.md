## Why

Der RSVP-Modus `rsvp_opt_out` ist auf Spielen heute nur in `ListMyGames` (Kalender) implementiert — alle anderen Game-Endpoints (`ListGames`, `GetGame`, `GetParticipants`) zählen weiterhin nur explizite Responses. Konkret: Spiel 39 mit `rsvp_opt_out=1` zeigt im Kalender 19 Zusagen, in der Detail-Ansicht aber 0. Außerdem nutzt `ListMyGames` zur Bestimmung der „impliziten Zusagen" `team_memberships`, während Trainings (`GetSession`) und Detail-Ansicht (`GetParticipants`) `kader_members` (via View `player_memberships`) nutzen — zwei unterschiedliche Definitionen, was bei selber RSVP-Konfiguration zu unterschiedlichen Zahlen führen kann. Ziel: ein einheitliches Verhalten, einzig durch `rsvp_opt_out` / `rsvp_require_reason` gesteuert.

## What Changes

- `GetParticipants` (`GET /api/games/{id}/participants`): bei `rsvp_opt_out=1` wird `rsvp_status='confirmed'` für reguläre Kader-Mitglieder (`is_extended=0`) ohne Response zurückgegeben. Extended-Member-Logik unverändert (kein implizit-confirmed).
- `ListGames` (`GET /api/games`): `confirmed_count` wendet dieselbe CASE-Logik an wie der bestehende Trainings-Pfad — explizit confirmed + implizit confirmed via `kader_members`.
- `GetGame` (`GET /api/games/{id}`): liefert neu `confirmed_count`, `declined_count`, `maybe_count` (opt-out-aware), damit Frontend nicht selbst rechnen muss.
- `ListMyGames`: SQL für `confirmed_count` und `inRegularKader` einheitlich auf `kader_members` umgestellt — vorher mischte das Statement `team_memberships` (Count) und `kader_members` (my_rsvp implicit), das war intern inkonsistent.
- **BREAKING (für Frontend-Konsumenten):** `confirmed_count` aus `ListGames` und `ListMyGames` kann sich nach diesem Change für bestehende Spiele leicht ändern, wenn `team_memberships` und `kader_members` für dasselbe Team unterschiedliche Zählwerte ergaben. Verschwindet im Normalfall, weil Kader-Mitglieder eine Teilmenge der Team-Mitglieder sind — bewusst akzeptiert.

## Capabilities

### New Capabilities
*(keine)*

### Modified Capabilities
- `rsvp-event-config`: präzisiert das bestehende Scenario `confirmed_count bei Opt-Out` (verwendet `kader_members` als Quelle der „Team-Mitglieder") und fügt neue Scenarios für Detail-Endpoints (`GetParticipants`, `GetGame`) hinzu.

## Impact

**Backend:**
- `internal/games/handler.go` — `ListGames` (SELECT + Scan), `GetGame` (SELECT + Response-Struct), `GetParticipants` (SQL + Scan), `ListMyGames` (SQL für Count und inRegularKader vereinheitlichen).

**Tests:**
- `TestListGames_OptOutCountsKaderImplicit` — Spiel mit `rsvp_opt_out=1`, 0 explizite Responses, 3 Kader-Members → `confirmed_count=3`.
- `TestGetParticipants_OptOutMarksKaderConfirmed` — Spiel mit `rsvp_opt_out=1`, Kader-Member ohne Response → `rsvp_status='confirmed'`.
- `TestGetParticipants_OptOutExtendedRemainsNull` — Extended-Member ohne Response bleibt `rsvp_status=null`.
- `TestGetGame_ReturnsCounts` — `confirmed_count`, `declined_count`, `maybe_count` im Response.

**Frontend:** keine zwingenden Änderungen, da `TermineDetailPage.tsx` bereits clientseitig `participants.filter(p => p.rsvp_status === 'confirmed')` aufruft — das funktioniert nach dem Backend-Fix automatisch. Optional könnte die Detail-Page künftig die Counts direkt aus `GetGame` lesen, um die Client-Filter-Logik einzusparen — out of scope hier.

**Datenbank:** keine neue Migration.

**API-Kompatibilität:** Additiv (`GetGame` neue Felder), counts können geringfügig verändert sein (siehe BREAKING-Hinweis).
