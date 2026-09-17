## Why

Der Mannschafts-Filter (Dropdown mit Checkboxen, `web/src/components/TeamFilter.tsx`)
auf `/kalender` und `/termine` listet ausschließlich Mannschaften (`teams`-Zeilen aus
`GET /api/teams`). Übungsgruppen (`kader.kind='practice'`, eigene Trainingstermine ohne
`teams`-Zwilling) tauchen dort nicht auf — wer nur die Termine einer Übungsgruppe sehen
will (deren Trainer, Mitglied oder ein Elternteil), kann nicht danach filtern und muss
durch die ungefilterte Liste aller Termine scrollen.

## What Changes

- **Neuer Endpoint `GET /api/practice-groups/my`** (Authenticated-Tier, jeder eingeloggte
  Nutzer): liefert die für den Aufrufer sichtbaren Übungsgruppen der aktiven Saison als
  `[{id, name}]`. Sichtbarkeit spiegelt die bestehende Rollenlogik von `GET /api/teams`
  (`ListTeamsForUser`), angewandt auf `kader.kind='practice'` statt auf `teams`:
  admin/vorstand/sportliche_leitung sehen alle, Trainer ihre eigenen (`kader_trainers`),
  Spieler/Elternteile ihre eigenen (`kader_members`/`kader_extended_members`/
  `family_links`). Die bestehende Vorstand/Trainer/sportliche_leitung-CRUD-Route
  `GET /api/practice-groups` bleibt unverändert und unangetastet — dieser neue Endpoint
  ist bewusst separat, weil er (anders als die CRUD-Liste) für alle Rollen erreichbar
  sein muss und nutzerspezifisch statt vollständig filtert.
- **`/kalender` und `/termine` laden diese Liste zusätzlich zu `/teams`** und zeigen die
  Übungsgruppen als weitere Checkboxen im selben `TeamFilter`-Dropdown — keine separate
  UI, keine visuelle Trennung von Mannschaften nötig.
- **ID-Kodierung im Filter-State:** Übungsgruppen-IDs (`kader.id`) und Mannschafts-IDs
  (`teams.id`) sind unterschiedliche ID-Räume und können kollidieren. Der Filter kodiert
  Übungsgruppen deshalb als **negative** Zahlen (`-kader_id`) in derselben
  `Set<number>` — `lib/teamFilter.ts` bleibt dadurch unverändert wiederverwendbar (die
  Bibliothek kennt nur „numerische ID", keine Bedeutung dahinter).
- **Trainings-Matching wird um den Übungsgruppen-Fall erweitert:** Trainingstermine mit
  `team_id=0` (Server-Projektion für „gehört einer Übungsgruppe", siehe Gotcha) matchen
  ab jetzt über `-kader_id` statt structurell nie einem Filter zu entsprechen.
- **Spiele bleiben unberührt** — Übungsgruppen haben strukturell keine Spiele
  (`teams`-Zwilling fehlt), eine Übungsgruppen-Auswahl blendet Spiele deshalb weiterhin
  komplett aus (erwartetes Verhalten, kein Sonderfall in der Filterlogik nötig).

## Capabilities

### New Capabilities
- `uebungsgruppen-termin-filter`: Übungsgruppen als wählbare Filteroption im
  Mannschafts-Filter von `/kalender` und `/termine`, inklusive des dafür nötigen
  Sichtbarkeits-Endpoints.

### Modified Capabilities
- `termine-unified-view`: Die Team-Filter-Requirement wird um Übungsgruppen als
  zusätzliche, gleichwertig wählbare Filteroption erweitert (Matching über `kader_id`
  statt `team_id` für Termine ohne Mannschaft).
- `permissions`: neue Requirement für `GET /api/practice-groups/my` im
  Authenticated-Tier (analog zur bestehenden „Dienst-Rangliste im Authenticated-Tier
  mit Handler-Scope"-Requirement) — nötig, damit die mechanische Permission-Matrix
  (`internal/permissions/matrix_test.go`) die neue Route kennt.

## Impact

- **Backend:** `internal/practicegroups/handler.go` (neuer Handler `ListMine`),
  `internal/app/router.go` (neue Route im Authenticated-Tier).
- **Frontend:** `web/src/pages/KalenderPage.tsx`, `web/src/pages/TerminePage.tsx`
  (zusätzlicher Fetch + Merge in die Filter-Optionen), `web/src/lib/teamFilter.ts`
  (neue Hilfsfunktion für die Trainings-Filter-ID-Ableitung).
- **Tests:** neuer Go-Handler-Test (Happy-Path + Rollen-Sichtbarkeit + 401), Vitest-Tests
  für die neue `teamFilter.ts`-Hilfsfunktion und die beiden Seiten
  (`KalenderPage.teamfilter.test.tsx`, `TerminePage.teamfilter.test.tsx`).
