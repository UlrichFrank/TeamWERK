## Context

Das Dashboard aggregiert Daten aus mehreren Packages (dashboard, games, trainings, duties, mitfahrgelegenheiten) in einem einzigen `GET /api/dashboard`-Call. Die bisherige Trennung in "Diese Woche" (Aktionen) und "Dienstkonto" (Saldo) ist für Endnutzer nicht intuitiv. Außerdem fehlen Trainings komplett in der Terminübersicht, obwohl sie bereits über `/api/training-sessions` und `TermineDetailPage` verfügbar sind.

## Goals / Non-Goals

**Goals:**
- Vier klar abgegrenzte Dashboard-Kacheln mit eigenem Fokus
- Terminanzeige umfasst alle Event-Typen (training_sessions + games)
- Dienste-Kachel bündelt Handlungsbedarf + Saisonstand in einem Block
- Neue Seite `/mein-team` als leichtgewichtige Kontaktübersicht
- Fahrgemeinschaften-Kachel zeigt nur bestätigte Information

**Non-Goals:**
- Kein neues Echtzeit-Push für Dashboard-Updates (SSE bleibt wie bisher)
- Keine Änderung am Rollen-Modell oder an Berechtigungen
- Keine Pagination auf dem Dashboard selbst

## Decisions

### 1. Terminanzeige: tagesbasiert statt zählenbasiert

**Entscheidung:** "Meine Termine" zeigt alle Events des *nächsten Tages mit mindestens einem Termin*, nicht eine feste Zahl (z.B. "nächste 3").

**Rationale:** Mehrere Trainings am selben Abend (z.B. A- und B-Jugend eines Elternteils) würden bei einer Zählung auf verschiedene Tage auseinandergerissen. Tagesbasiert ist die natürliche Frage: "Was ist morgen/diese Woche?"

**Implementierung:** UNION-Query auf `training_sessions` und `games`, nach Datum sortiert, dann alle Rows des ersten Datums zurückgeben.

```sql
-- Pseudocode
SELECT MIN(date) AS next_date FROM (
  SELECT date FROM training_sessions WHERE team_id IN (...) AND date >= DATE('now')
  UNION ALL
  SELECT date FROM games WHERE team_id IN (...) AND date >= DATE('now')
)
-- dann alle Events WHERE date = next_date
```

### 2. Dienste-Kachel: nächstes Spiel mit Slots (nicht Trainings)

**Entscheidung:** Bezugspunkt für "Meine Dienste" ist das nächste `game` mit mindestens einem `duty_slot` — unabhängig davon, ob vorher Trainings stattfinden.

**Rationale:** Trainings haben keine Dienst-Slots. Eine Dienst-Kachel, die auf ein Training zeigt, hätte keinen darstellbaren Inhalt. Der Nutzer muss das nächste *relevante* Spiel sehen, nicht einfach den nächsten Termin.

**Anzeigelogik:**
- User hat ≥1 eigene Zusage für dieses Spiel → Liste der eigenen Slots (Typ + Zeit)
- User hat 0 Zusagen → "N offene Slots" mit Link zu /dienste
- Immer zusätzlich: Saison-Saldo (Ist/Soll + Fortschrittsbalken)

### 3. Mein Team: neuer Endpoint GET /api/teams/:id/roster

**Entscheidung:** Eigener Endpoint statt Erweiterung von `/api/members`.

**Rationale:** Die Roster-Ansicht kombiniert drei heterogene Personengruppen (Trainer via `kader_trainers`, Spieler via `kader_members`, Eltern via `family_links`). Das passt nicht in die bestehende Members-Paginierung. Ein dedizierter Endpoint ist sauberer und unabhängig.

**Berechtigung:** Alle authentifizierten User, die Zugriff auf das Team haben (via `user_accessible_teams`).

**Response-Struktur:**
```json
{
  "team": { "id": 1, "name": "A-Jugend" },
  "trainers": [{ "name": "...", "email": "...", "phone": "..." }],
  "players": [{ "name": "...", "jersey_number": 7, "status": "aktiv", "email": "...", "phone": "..." }],
  "parents": [{ "name": "...", "email": "...", "phone": "...", "children": ["Anna K."] }]
}
```

### 4. Fahrgemeinschaften: bestätigte Paare, nächste 3 Auswärtsspiele

**Entscheidung:** Dashboard zeigt nur `status='confirmed'`-Paarungen, für die nächsten max. 3 Auswärtsspiele des Users.

**Rationale:** Offene Einträge ("biete/suche") gehören auf die Mitfahrgelegenheiten-Seite. Das Dashboard ist kein Aktionsort für Fahrgemeinschaften — es soll nur zeigen "was ist schon geregelt". Die Erweiterung auf 3 Spiele gibt mehr Vorausschau ohne den Block zu überfluten.

**Backend-Änderung:** `queryCarpoolingHint` wird zu `queryCarpoolingConfirmed` — gibt ein Array von bis zu 3 Einträgen zurück (ein Eintrag pro Auswärtsspiel mit bestätigten Paaren).

## Risks / Trade-offs

**[Kein Termin im nächsten Tag] → Mitigation:** Wenn der nächste Tag mit Events weit in der Zukunft liegt, ist das korrekt — die Kachel zeigt einfach diesen Termin. Kein künstliches Limit.

**[Kein Spiel mit offenen Slots] → Mitigation:** Wenn kein kommendes Spiel mit Slots existiert (Saisonende), zeigt "Meine Dienste" nur den Saldo — kein Fehlerfall.

**[Roster-Endpoint mit sensiblen Kontaktdaten] → Mitigation:** Der Endpoint wird hinter `auth.Middleware` geschützt. Zugriff nur für User mit Team-Zugehörigkeit via `user_accessible_teams`. E-Mail und Telefon werden nur zurückgegeben wenn sie in `users`/`members` vorhanden sind (nullable).

**[UNION-Query Performance] → Mitigation:** Beide Tabellen haben Indices auf `date` und `team_id`. Bei SQLite und der geringen Datenmenge (<1000 Rows) kein Risiko.

## Migration Plan

- Keine DB-Migration erforderlich (keine Schema-Änderungen)
- Frontend und Backend werden simultan deployed (`make deploy`)
- Rollback: `git revert` + `make deploy`

## Open Questions

- Sollen Telefonnummern im Roster angezeigt werden, falls in `members.phone` vorhanden? (Noch kein `phone`-Feld in der DB sichtbar — ggf. nur E-Mail)
- Nav-Eintrag für "Mein Team" in der Sidebar: für alle Rollen oder nur Trainer/Eltern?
