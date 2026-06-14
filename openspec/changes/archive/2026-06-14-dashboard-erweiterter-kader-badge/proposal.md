## Why

Spieler im Erweiterten Kader einer Mannschaft sehen im Dashboard dieselben Termine und denselben Teameintrag wie Stammkader-Spieler — ohne zu wissen, warum. Die fehlende Kennzeichnung erzeugt Verwirrung: „Warum sehe ich Trainings von Damen 1?" oder „Bin ich wirklich Teil dieses Teams?".

## What Changes

- `GET /api/teams/my` erhält pro Team ein neues Feld `isExtended: bool` — `true`, wenn der User das Team ausschließlich über `kader_extended_members` sieht.
- `GET /api/dashboard` → `meineTermine`: Jedes `NextEvent`-Objekt erhält `isExtended: bool` — `true`, wenn der User das Event-Team ausschließlich über `kader_extended_members` sieht.
- Dashboard „Mein Team"-Accordion: Badge „Erw. Kader" neben Teamnamen, wenn `isExtended`.
- Dashboard „Meine Termine"-Accordion: Zusatz „(Erw. Kader)" in der Teamzeile, wenn `isExtended`.

## Capabilities

### New Capabilities

- `erweiterter-kader-dashboard-badge`: Erweiterte Kader-Mitglieder sehen im Dashboard eine Kennzeichnung, die ihren Status als Nicht-Stammkader-Mitglied deutlich macht — sowohl bei Terminen als auch beim Teameintrag.

### Modified Capabilities

*(keine — rein additive Felder, bestehende Anforderungen bleiben unverändert)*

## Impact

- `internal/teams/handler.go` — `ListMyTeams`: Query erweitern, `IsExtended bool` zum Response-Struct
- `internal/dashboard/handler.go` — `queryNextEvents`: `IsExtended bool` zu `NextEvent`-Struct und Query
- `web/src/pages/DashboardPage.tsx` — `MeinTeamSection` + `MeineTermineSection`: Badge rendern
- Keine DB-Migration
- Keine Breaking Changes (additive Felder)
