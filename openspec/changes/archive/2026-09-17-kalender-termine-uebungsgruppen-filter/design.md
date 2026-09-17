## Context

`/kalender` (`web/src/pages/KalenderPage.tsx`) und `/termine` (`web/src/pages/TerminePage.tsx`)
laden ihre Filter-Optionen über `GET /api/teams` (`ListTeamsForUser` in
`internal/games/handler.go`) und bauen daraus mit `buildTeamOptions` (`lib/teamName.ts`)
die Checkbox-Liste für `<TeamFilter>`. `ListTeamsForUser` ist bereits pro Nutzer
gescoped (admin/vorstand/sportliche_leitung: alle Teams; Trainer: eigene via
`kader_trainers`; Spieler/Eltern: eigene via `user_accessible_teams`).

Übungsgruppen (`kader.kind='practice'`, Migration `057`) sind Kader **ohne**
`teams`-Zwilling — genau diese Abwesenheit hält Spiele, Dienste, Kasse etc. strukturell
von ihnen fern (siehe Gotcha „Übungsgruppen" in `docs/agent/06-gotchas.md`). Ihre
Trainingstermine tragen `training_sessions.kader_id` (immer gesetzt) und
`team_id IS NULL`, was die API als `team_id: 0` projiziert
(`internal/trainings/handler.go:1129`, Kommentar: „0 heißt gehört einer Übungsgruppe").
Weil **alle** Übungsgruppen-Trainings denselben `team_id=0` tragen, kann der bestehende
Filter (der nach `team_id` matcht) sie nicht einzeln unterscheiden — er müsste stattdessen
nach `kader_id` matchen, das er heute nicht kennt.

Die einzige bestehende Übungsgruppen-Leseroute (`GET /api/practice-groups`,
`internal/practicegroups/handler.go`) sitzt im Vorstand/Trainer/sportliche_leitung-Tier
(`internal/app/router.go:719`) und liefert zudem Mitglieder-/Trainer-Details, die für
einen reinen Filter nicht gebraucht werden und für Spieler/Eltern nicht zugänglich sein
sollen (Datenschutz von Mitgliederlisten). Sie ist für diesen Zweck nicht direkt
wiederverwendbar.

## Goals / Non-Goals

**Goals:**
- Übungsgruppen als zusätzliche, gleichwertige Filteroption in der bestehenden
  `<TeamFilter>`-Dropdown auf `/kalender` und `/termine`.
- Sichtbarkeit der Übungsgruppen-Option folgt derselben Rollenlogik wie bei Mannschaften
  (jeder sieht nur, wozu er Zugang hat; Vorstand/Trainer-Rollen sehen mehr/alles).
- Keine Änderung an `GET /api/teams`, `GET /api/practice-groups` oder an anderen
  Konsumenten von `lib/teamFilter.ts` (Dienstbörse, Mitfahrgelegenheiten, Dashboard) —
  die haben strukturell keine Übungsgruppen-Bezüge (Dienste/Mitfahrten/Dashboard-Kacheln
  existieren für Übungsgruppen nicht) und sollen unverändert bleiben.

**Non-Goals:**
- Keine Übungsgruppen-Sichtbarkeit in `/dienste`, `/mitfahrten` oder dem Dashboard-Filter.
- Kein Update der Übungsgruppen-CRUD-Seite (`/admin/uebungsgruppen`) — die Sichtbarkeits-
  logik entsteht als neuer, separater Endpoint.
- Keine visuelle Trennung („Mannschaften" vs. „Übungsgruppen" als Gruppen-Header) im
  Dropdown — eine flache Liste reicht für die erwartete Größenordnung (wenige
  Übungsgruppen pro Verein).

## Decisions

### 1. Neuer Endpoint statt Erweiterung von `GET /api/teams` oder `GET /api/practice-groups`

`GET /api/practice-groups/my` (neu, `internal/practicegroups`, Authenticated-Tier) statt:
- **`GET /api/teams` erweitern**: würde Übungsgruppen auch in `/dienste`,
  `/mitfahrten` und dem Dashboard-Filter erscheinen lassen, die alle denselben Endpoint
  konsumieren — dort sind Übungsgruppen strukturell fehl am Platz (keine Dienste, keine
  Mitfahrten, keine Konten). Ein Discriminator-Feld plus Filterung an vier Call-Sites
  wäre fehleranfälliger als ein eigener, kleiner Endpoint.
- **`GET /api/practice-groups` für alle öffnen**: liefert Mitglieder- und Trainerlisten
  jeder Gruppe — für einen reinen Namens-Filter unnötige Daten, und die Route müsste
  ihre Autorisierung (aktuell vorstand/trainer/sportliche_leitung, ungefiltert) auf
  „jeder sieht nur seine eigenen Gruppen, außer privilegierte Rollen" umbauen, was die
  bestehende Admin-Oberfläche (`UebungsgruppenPage.tsx`, die alle Gruppen zum Verwalten
  braucht) verkomplizieren würde.

Der neue Endpoint spiegelt exakt die Rollenverzweigung aus `ListTeamsForUser`
(admin/vorstand/sportliche_leitung: alle; Trainer: eigene via `kader_trainers`;
Spieler/Eltern: eigene via `kader_members`/`kader_extended_members`/`family_links`),
aber auf `kader.kind='practice'` direkt — ohne den Umweg über `teams`/
`user_accessible_teams`, die für Übungsgruppen ohnehin nicht greifen (siehe Gotcha).

### 2. ID-Kodierung: negative Zahlen statt eines zweiten Sets oder eines präfixierten Strings

Übungsgruppen-IDs (`kader.id`) und Mannschafts-IDs (`teams.id`) sind unabhängige
Primärschlüssel-Räume und können denselben Zahlenwert tragen. `lib/teamFilter.ts`
arbeitet aber durchgängig mit `Set<number>` (URL-Parsing, Serialisierung, Toggle,
Matching) und wird von vier Seiten geteilt. Optionen:
- **Zweites `Set<number>` für Übungsgruppen** (getrennter State, getrennter Query-Param
  `practice=`): verdoppelt jede Funktion in `teamFilter.ts` (parse/serialize/toggle/match)
  und jede Aufrufstelle in `KalenderPage.tsx`/`TerminePage.tsx`; die „leere Auswahl heißt
  kein Filter"-Semantik müsste zwischen zwei Mengen synchron gehalten werden.
- **String-IDs** (`"team-3"`, `"practice-5"`): bricht die bestehende
  `Set<number>`-Schnittstelle und alle IDs, die aus der URL geparst werden
  (`parseTeamIds` nutzt `parseInt`).
- **Gewählt: negative Zahlen** (`-kader_id`) im selben `Set<number>`. Reale Team-IDs
  und Kader-IDs sind SQLite-`INTEGER PRIMARY KEY`s, beginnend bei 1 — niemals negativ oder
  0. `-kader_id` ist damit eindeutig von jeder positiven Team-ID unterscheidbar, ohne dass
  `teamFilter.ts` etwas über die Bedeutung einer ID wissen muss: „ist er drin, matcht er"
  bleibt exakt der bestehende Code. Konvention ist bereits im Projekt etabliert
  (`training_sessions.team_id=0` bedeutet ebenfalls „lies das anders" für Übungsgruppen).

### 3. Trainings-Matching: `kader_id` statt `team_id` für Übungsgruppen-Trainings

Die Frontend-`Training`/`Session`-Interfaces (`KalenderPage.tsx`, `TerminePage.tsx`)
bekommen ein zusätzliches Feld `kader_id: number` (vom Server bereits geliefert, bisher
nur nicht typisiert/genutzt). Eine neue Hilfsfunktion in `lib/teamFilter.ts`,
`trainingFilterId({team_id, kader_id})`, liefert `team_id > 0 ? team_id : -kader_id` — der
Wert, gegen den `matchesTeamFilter` geprüft wird. Games bleiben unverändert (`team_ids`),
weil sie strukturell nie zu einer Übungsgruppe gehören.

### 4. Kein `scope`-Parameter am neuen Endpoint

`ListTeamsForUser` kennt `?scope=duties`/`attendance-stats`/`diary-stats` für
Sonderfälle, die für Übungsgruppen nicht existieren (keine Dienstpflicht, keine
Anwesenheitsstatistik, siehe Gotcha). Der neue Endpoint braucht deshalb nur die eine,
ungescopte Sichtbarkeitsregel — kein Parameter, keine Verzweigung.

## Risks / Trade-offs

- **Negative IDs als impliziter Vertrag** → Nur `lib/teamFilter.ts` und die beiden
  Seiten müssen die Konvention kennen; ein Kommentar an `trainingFilterId` und an der
  Stelle, die die Übungsgruppen-Optionen baut, macht sie sichtbar. Ein Vitest-Test
  (`teamFilter.test.ts`) hält die Kodierung fest, damit eine künftige Änderung sie nicht
  versehentlich bricht.
- **Ein weiterer Endpoint mit ähnlicher Sichtbarkeitslogik wie `ListTeamsForUser`**
  → bewusst in Kauf genommen (siehe Decision 1); die Alternative hätte vier bestehende,
  gut getestete Call-Sites angefasst. Beide Implementierungen dürften künftig
  unabhängig voneinander driften — das ist der Trade-off für die Isolation.
- **`GET /api/practice-groups/my` liegt im Authenticated-Tier, nicht im
  Vorstand/Trainer/sportliche_leitung-Tier wie die CRUD-Route** → beabsichtigt (jeder
  Nutzer muss seine eigenen Gruppen zum Filtern sehen können), aber die
  Objektrechte-Matrix (`internal/permissions/object_matrix_test.go`) sollte den Endpoint
  in `openByDesign`/passende Kategorie aufnehmen, da er absichtlich weiter offen ist als
  die CRUD-Route.

## Migration Plan

Rein additiv: neue Route, neue Frontend-Fetches, erweiterte Interfaces. Kein
DB-Schema-Wechsel, keine Migration nötig. Rollback ist ein einfacher Revert (kein
Datenverlust, keine Bestandsdaten betroffen).
