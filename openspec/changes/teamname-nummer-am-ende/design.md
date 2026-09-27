# Design: Teamname mit Nummer am Ende

## Context

Motivation siehe proposal.md. Heute existieren drei Quellen für den Langnamen einer Mannschaft:

1. **`teams.name`** (gespeichert, saisonunabhängig): `kader.teamLabel` baut ihn beim Anlegen — „C-Jugend männlich" für Nummer 1, „C-Jugend männlich 2" ab Nummer 2. Er ist Identitätsschlüssel in `kader.ensureTeam` (`WHERE name=? AND age_class=? AND gender=?`), der H4A-Import liest die Nummer daraus (`h4aimport.TeamNumberFromAlias`, `games/h4aimport_teamguess.go`), und `AppShell` leitet aus ihm den URL-Slug ab.
2. **`appdb.TeamDisplayName`** (SQL-Fragment, saisonabhängig): `<age_class> <n> <Geschlecht>` bei mehreren Kadern der Kombination, sonst `<age_class> <Geschlecht>`. Wird als `display_long` von Roster, Spielen und Trainings ausgeliefert.
3. **Client-seitig** in `AdminKaderPage` (wie 2), Lösch-Bestätigung (Nummer nur ab 2, vor dem Geschlecht), `CopyKaderModal` und `AutoAssignModal` (ohne Nummer).

„Mein Team" rendert `roster?.team.display_long || team.name`: vor dem Laden des Kaders greift (1), danach (2) — daher der Wechsel beim Aufklappen.

## Goals / Non-Goals

**Goals:**
- Ein Schema, `<Altersklasse> <Geschlecht>[ <Nummer>]`, für alle *angezeigten* Langnamen.
- Die Nummer hängt an der Saison-Konstellation (mehrere Kader derselben Kombination), nicht an der Nummer selbst — so trägt auch Mannschaft 1 ihre „1", sobald eine zweite existiert.

**Non-Goals:**
- Keine Änderung an `teams.name` (Wert, Schreibweise, Anlage-Logik).
- Keine Änderung an der Kurzform (`display_short`).
- Kein Durchforsten sämtlicher Endpoints, die heute `teams.name` roh ausliefern (z. B. Admin-Teamlisten). Diese Namen folgen bereits dem Muster „Nummer am Ende" — sie können lediglich bei Mannschaft 1 die „1" weglassen. Erfasst werden die gemeldete Stelle („Mein Team") und alle Stellen, die heute eine *abweichende Wortreihenfolge* erzeugen.

## Decisions

### 1. Reihenfolge im SQL-Helfer drehen, statt `teams.name` umzuschreiben

`TeamDisplayName` wird zu `age_class || ' ' || <Geschlecht> || ' ' || team_number` (Mehrfall) bzw. unverändert `age_class || ' ' || <Geschlecht>`. Damit ändern sich alle `display_long`-Ausgaben an einer Stelle.

*Alternative:* `teams.name` per Migration auf „… männlich 1" normalisieren und überall `name` anzeigen. Verworfen: `name` ist saisonunabhängig und kann nicht wissen, ob es in der aktuellen Saison eine zweite Mannschaft gibt; außerdem hängen `ensureTeam`-Lookup, H4A-Nummernerkennung und URL-Slugs an der heutigen Schreibweise — eine Umbenennung erzeugte beim nächsten Kader-Anlegen Dubletten-Teams und brüche Bookmarks.

### 2. `GET /api/teams/my` liefert `display_short`/`display_long`

`ListMyTeams` selektiert zusätzlich `COALESCE(TeamDisplayShort, t.name)` und `COALESCE(TeamDisplayName, t.name)` — dasselbe COALESCE-Muster wie `GetRoster`. Das `Team`-Struct trägt die Felder bereits. Die `ORDER BY t.name` bleibt (sortiert „… männlich" vor „… männlich 2", also korrekt nach Nummer).

„Mein Team" rendert dann `roster?.team.display_long || team.display_long || team.name`. Da beide aus derselben SQL-Funktion stammen, sind sie identisch; der Roster-Wert bleibt nur als erster Kandidat, damit sich nach einem Live-Reload nichts verschiebt.

*Alternative:* Namen client-seitig aus `buildTeamShortNames`-ähnlicher Logik bauen. Verworfen: `/teams/my` liefert keine Kader-Felder, und die Spec verlangt, dass der Server die Display-Strings liefert.

### 3. Ein Client-Helfer für Kader-basierte Titel

`web/src/lib/teamName.ts` bekommt `buildTeamLongName(k, groupCount)` (bzw. eine Map-Variante analog `buildTeamShortNames` über eine Kader-Liste) mit der identischen Regel. `AdminKaderPage` (Titel + Lösch-Bestätigung), `CopyKaderModal` und `AutoAssignModal` nutzen ihn; die drei lokalen `GENDER_LABEL`-Kopien entfallen zugunsten des exportierten `GENDER_LABEL`. `groupCount` wird aus der jeweils angezeigten Kader-Liste derselben Saison gezählt (alle drei Oberflächen laden `/api/kader?season_id=…` und haben die Liste). `team_number` liefert `/api/kader` bereits; die lokalen Interfaces der Modals werden um das Feld ergänzt.

Diese Oberflächen zeigen Kader einer *gewählten* Saison (nicht zwingend der aktiven) — deshalb dort client-seitige Zählung statt `display_long`, das an die aktive Saison gebunden ist.

## Risks / Trade-offs

- [Externe Konsumenten parsen `display_long`] → Nur das eigene Frontend liest das Feld; iCal-Feed und Push-Texte verwenden eigene Pfade. Vor der Umsetzung per `grep display_long|team_display_long_csv|TeamDisplayName` bestätigen; betroffen sind laut Stand nur Tests in `internal/games`.
- [Uneinheitliche Anzeige an Stellen mit rohem `teams.name`] → Dort fehlt höchstens die „1" bei Mannschaft 1, die Reihenfolge stimmt. Bewusst außerhalb des Umfangs (Non-Goals); folgt ggf. als eigener Change.
- [Sortierung nach `display_long`] → Stellen, die nach der Langform sortieren, bleiben stabil: „C-Jugend männlich 1" < „C-Jugend männlich 2", und verschiedene Altersklassen sortieren weiterhin nach dem Präfix.

## Migration Plan

Keine DB-Migration. Deploy wie üblich; Rollback per `make deploy-rollback` ist ohne Datenfolgen, da nur berechnete Ausgaben betroffen sind.
