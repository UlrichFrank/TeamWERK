## Context

TeamWERK hat ein vollständig implementiertes Training-RSVP-System (`training_responses`,
`/api/training-sessions/{id}/respond`, `TrainingsPage`, `TrainingsDetailPage`). Für Spiele
existiert dieses System noch nicht. Statt eine zweite parallele Struktur zu bauen, werden
Trainings und Spiele auf einer gemeinsamen `/termine`-Seite zusammengeführt.

Bestehende Datenbankstruktur für Referenz:
- `training_sessions` ← Trainingstermine
- `training_responses` ← RSVP-Einträge (training_id, member_id, status, reason)
- `games` ← Spieltermine
- `game_teams` ← Zuordnung Spiele → Teams (n:m)
- `team_memberships` ← Zuordnung Members → Teams pro Saison

## Goals / Non-Goals

**Goals:**
- Spieler/Eltern können zu Spielen RSVP abgeben (confirmed/declined/maybe + Grund)
- Trainer sehen Rückmeldungs-Übersicht für Trainings und Spiele je auf einer Detailseite
- `/termine` ersetzt `/trainings` als primäre RSVP-Oberfläche
- Chronologisch gemischte Liste (Trainings + Spiele) team-gefiltert nach eingeloggtem User

**Non-Goals:**
- Anwesenheits-Tracking für Spiele (bleibt nur für Trainings)
- Push-Benachrichtigungen bei neuen Spielterminen (separater Change)
- Öffentliche Spieler-Statistiken über Anwesenheitsquote

## Decisions

### D1: Gemeinsame Seite statt zweier separater Seiten

**Entscheidung**: `/termine` zeigt Trainings und Spiele gemischt; `/trainings` wird abgelöst.

**Rationale**: Spieler denken in „Terminen", nicht in „Trainings" vs. „Spielen". Eine
gemeinsame Liste reduziert Navigation und Code-Duplikation (RSVP-Logik und -UI sind identisch).

**Alternative**: Separate `/spiele`-Seite parallel zu `/trainings` — abgelehnt wegen
doppelter Nav-Einträge und Code-Duplikation.

### D2: Separate game_responses-Tabelle statt generischer event_responses

**Entscheidung**: Neue Tabelle `game_responses` mit identischer Struktur wie `training_responses`
(FK auf `games.id` statt `training_sessions.id`).

**Rationale**: Kein ORM-Prinzip des Projekts; FK-Constraints und CHECK-Constraints bleiben
einfach und explizit. Eine generische `event_responses`-Tabelle würde nullable FKs erfordern
und die Integrität schwächen.

**Alternative**: Polymorphe `event_responses(event_type, event_id, ...)` — abgelehnt,
da SQLite FK-Constraints nicht über nullable FK-Spalten funktionieren.

### D3: Neuer GET /api/games/my Endpoint (statt bestehenden /api/games erweitern)

**Entscheidung**: Neuer user-aware Endpoint `/api/games/my` mit Team-Filterung und RSVP-Daten.

**Rationale**: Bestehender `/api/games` ist Admin/Kalender-Endpoint ohne User-Kontext.
Erweiterung würde Breaking Change riskieren. Saubere Trennung: `my` = user-facing,
`/games` = admin.

**Route-Reihenfolge-Hinweis**: In Chi v5 muss `/api/games/my` vor `/api/games/{id}`
registriert werden, da sonst „my" als ID interpretiert wird.

### D4: Frontend-API — ein kombinierter Endpoint statt zwei parallele Calls

**Entscheidung**: `TerminePage` macht zwei parallele API-Calls (`/api/training-sessions` und
`/api/games/my`), mergt und sortiert clientseitig.

**Rationale**: Kein neuer Server-Aggregations-Endpoint nötig; beide Listen sind bereits
paginierungslos und zeitlich gefiltert. Client-Merge ist mit < 100 Einträgen trivial.

**Alternative**: Neuer `/api/termine`-Aggregationsendpoint — abgelehnt als Over-Engineering
für die Datenmenge.

### D5: Detailrouten /termine/training/:id und /termine/spiel/:id

**Entscheidung**: Zwei separate Detailseiten mit gemeinsamer RSVP-Tabellen-Komponente,
aber unterschiedlichem Kontext (Trainings haben Anwesenheits-Tracking, Spiele nicht).

**Rationale**: Unterschiedliches Daten-Modell (training_sessions vs. games) und
unterschiedliche Trainer-Aktionen (Anwesenheit erfassen nur für Trainings).

## Risks / Trade-offs

- **[Risiko] Redirect-Verwirrung** → Nutzer mit alten Bookmarks auf `/trainings` landen
  auf 404. Mitigation: React Router `<Navigate from="/trainings" to="/termine" />` einrichten.

- **[Trade-off] Doppelter fetch** → Zwei API-Calls auf `/termine` statt einem.
  Akzeptabel bei kleinen Datenmengen; beide Requests laufen parallel.

- **[Risiko] Chi-Route-Reihenfolge** → `/api/games/my` muss vor `/api/games/{id}` registriert
  sein. Wird in main.go durch Reihenfolge der Route-Registrierung sichergestellt.

## Migration Plan

1. Migration 012 ausführen (`game_responses` Tabelle anlegen)
2. Backend deployen (neue Endpoints; bestehende `/trainings`-Endpoints bleiben erhalten)
3. Frontend deployen (neue `/termine`-Seiten; alte `/trainings`-Seiten bleiben erreichbar)
4. Navigation auf „Termine" umstellen (AppShell)
5. Alte `/trainings`-Routen auf `/termine` redirecten

Rollback: Migration 012 down (Tabelle droppen); Frontend-Revert auf alten Build.
