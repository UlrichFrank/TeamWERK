## Why

Trainer, Vorstand und sportliche Leitung müssen sich vereinsweit koordinieren — heute geht das im Chat nicht. Zwei Lücken:

1. **Kein teamübergreifender Trainerkontakt.** `canContactUser` (`internal/chat/handler.go`) lässt außer admin/vorstand nur User mit **gemeinsamem Team** (`user_accessible_teams`) zueinander durch. Ein Trainer der mB2 kann einen Trainer der wA1 weder finden (`GET /chat/users`) noch anschreiben. Auch die **sportliche Leitung** ist im Kontakt-Bypass nicht enthalten (sie sieht zwar alle Team-Gruppen, kann aber teamfremde Trainer nicht kontaktieren).
2. **Keine „Alle Trainer"-Gruppe.** Die vordefinierten Standard-Gruppen (`internal/chat/team_groups.go`) sind pro Team (`Trainer mB2`, …). Eine Kachel, die **alle Trainer aller Teams** auf einmal in eine Gruppe holt, fehlt.

Bewusst **kein** neues „mitwachsendes" Konversations-Primitiv: die „Alle Trainer"-Kachel funktioniert **exakt wie die bestehenden Team-Gruppen** — ein Auswahl-Shortcut, der beim Anlegen eine normale Momentaufnahme-Gruppe erzeugt.

## What Changes

- **Neue Standard-Gruppen-Kachel „Alle Trainer"** in `GET /api/chat/team-groups`: ein synthetischer Eintrag (`teamId: 0`, `kind: "alle_trainer"`, `displayShort: "Alle Trainer"`, `count`), sichtbar für den **Zugriffskreis** (siehe unten) und admin.
- **Auflösung** über den bestehenden Endpoint `GET /api/chat/team-groups/0/alle_trainer/members` (Sentinel `teamId=0`, kein neuer Route-Eintrag): liefert **nur die Trainer aller Kader der aktiven Saison** (dieselbe Auflösung wie `kind="trainer"`, nur ohne den `k.team_id`-Filter), ohne den Caller. **Vorstand, sportliche Leitung und vorstand_beisitzer sind NICHT enthalten** — außer sie sind selbst Kader-Trainer dieser Saison. Zugriff (Auflösen) für jedes Zugriffskreis-Mitglied (nicht mehr nur vorstand/sL/admin).
- **`canContactUser` erweitert**: zusätzlich zur bestehenden Regel (admin/vorstand bzw. gemeinsames Team) dürfen sich **zwei Mitglieder des Zugriffskreises gegenseitig** kontaktieren — sowohl 1:1 (`createDirect`) als auch beim Gruppenaufbau (`createGroup`).
- **`GET /api/chat/users` erweitert**: ein Zugriffskreis-Mitglied findet in der Nutzersuche zusätzlich **alle anderen Zugriffskreis-Mitglieder** (teamübergreifend), nicht nur User mit gemeinsamem Team.
- **Frontend (`ChatPage.tsx`)**: `TeamGroup["kind"]` um `"alle_trainer"` erweitern, Label-Mapping + Tag-Rendering (Label ohne `displayShort`-Suffix für diese Kachel). Der übrige „Neue Gruppe"-Flow bleibt unverändert.

**Zwei Mengen (Zugriff ≠ Inhalt), beide DB-abgeleitet, aktive Saison:**
- **Zugriffskreis** (Kachel sehen/auflösen · 1:1-Kontakt · Nutzersuche): User mit (a) Kader-Trainer-Zuordnung der aktiven Saison (`kader_trainers`) **oder** (b) `vorstand` **oder** (c) `sportliche_leitung` **oder** (d) `vorstand_beisitzer`. `admin` stets berechtigt.
- **Mitgliedermenge** (Inhalt der „Alle Trainer"-Gruppe): **nur (a)** — Kader-Trainer der aktiven Saison. Vorstand/sL/beisitzer nur, wenn selbst Kader-Trainer.

## Capabilities

### Modified Capabilities

- `chat-team-groups`: neue synthetische Standard-Gruppe „Alle Trainer" (Listing + Auflösung teamübergreifend) mit eigener Sichtbarkeits-/Zugriffsregel (Trainerkreis).
- `chat-konversationen`: Kontaktierbarkeit (`canContactUser`) und Nutzersuche (`GET /chat/users`) erlauben teamübergreifenden Kontakt innerhalb des Trainerkreises (1:1 und Gruppe).

## Impact

- **Backend:** `internal/chat/team_groups.go` (synthetischer Eintrag in `ListTeamGroups`, `alle_trainer`-Zweig in `ResolveTeamGroup` + `countTeamGroupMembers`, `teamGroupKinds`, Helfer `isInTrainerCircle`/Trainerkreis-Query); `internal/chat/handler.go` (`canContactUser`, `Users`). **Keine neuen Routen, keine Migration.**
- **Frontend:** `web/src/pages/ChatPage.tsx` (Kind-Union, Label, Tag-Rendering).
- **SSE/Broadcast-Gate:** unberührt — nur GET-Endpoints und Verhaltensänderung an `POST /chat/conversations` (das bereits broadcastet). Keine neue Mutations-Route.
- **Bewusster Trade-off:** Die erzeugte „Alle Trainer"-Gruppe ist eine **Momentaufnahme** (wie alle Team-Gruppen). Später hinzukommende Trainer erscheinen nicht automatisch in bereits angelegten Gruppen. Kein Reconcile, keine dynamische Mitgliedschaft.
- **Keine neuen Dependencies.**
