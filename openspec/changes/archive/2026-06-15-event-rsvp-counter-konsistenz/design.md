## Context

Aktueller Stand (Stand: nach `event-rsvp-config-bearbeitbar`):

| Endpoint | Opt-Out-Aware? | Quelle für „implizite Zusage" |
|---|---|---|
| `ListGames` (`/api/games`) | ✗ — nur explizite Counts | n/a |
| `GetGame` (`/api/games/{id}`) | ✗ — gibt gar keine Counts zurück | n/a |
| `ListMyGames` (`/api/games/my`) | ✓ | `team_memberships` (Count) + `kader_members` (`inRegularKader`) — **intern inkonsistent** |
| `GetParticipants` (`/api/games/{id}/participants`) | ✗ — `rsvp_status` ist NULL bei Nicht-Response | n/a |
| `ListSessions` / `GetSession` (Trainings) | ✓ | `player_memberships` (= View über `kader_members`) |

Spec sagt seit `event-rsvp-config-bearbeitbar`: „`rsvp_opt_out` + `rsvp_require_reason` sind die einzigen Steuerungs-Flags". Das Implementierungsbild dazu hat noch Lücken — dieser Change schließt sie.

## Goals / Non-Goals

**Goals:**
- Identische Opt-Out-Logik in allen vier Game-Endpoints.
- Eine einheitliche Definition für „implizit zugesagt": regulärer Kader (`kader_members`), nicht `team_memberships` und nicht `kader_extended_members`.
- `GetGame` liefert Counts mit, damit Detail-Seite keine eigene Filter-Logik braucht (Frontend wird in diesem Change aber noch nicht umgestellt — additiv).

**Non-Goals:**
- Keine Refactorings am Frontend in diesem Change (`TermineDetailPage` darf weiterhin clientseitig zählen — funktioniert nach Backend-Fix automatisch korrekt).
- Keine Migration oder Datenanpassung.
- Trainings-Endpoints sind nicht im Fokus — die sind heute bereits konsistent (alle nutzen `player_memberships`).
- Keine UI-Änderung an der Detail-Page (Badges sind im Vor-Change schon eingebaut).

## Decisions

### Entscheidung 1 — „Implizit zugesagt" = reguläre Kader-Mitglieder

Der Auswahlraum „Spieler, die ohne Response als zugesagt gelten" sind die regulären Kader-Mitglieder (`kader_members`), nicht alle `team_memberships`. Begründung:

- Konsistenz mit Trainings-Endpoints, die bereits `player_memberships` (= View auf `kader_members`) nutzen.
- Die Detail-Page (`GetParticipants`) listet nur Kader-Mitglieder + Extended-Members — wenn der Count anhand `team_memberships` ginge, könnte er größer sein als die Liste der angezeigten Personen. Inkonsistent für Trainer/Vorstand.
- `kader_extended_members` werden bewusst NICHT als implizit zugesagt gewertet — diese Erweiterung dient als „Reserve", die explizit zusagen muss.

**Alternative verworfen:** `team_memberships` (wie bisher in `ListMyGames`). Würde Spieler einbeziehen, die nicht im aktuellen Spiel-Kader stehen — fachlich falsch.

### Entscheidung 2 — `GetParticipants` setzt `rsvp_status` server-seitig

Statt das Frontend die Opt-Out-Logik nachträglich anwenden zu lassen (z.B. „wenn `rsvp_status==null` und `rsvp_opt_out==1`, dann zähle als confirmed"), gibt der Endpoint direkt `rsvp_status='confirmed'` zurück. Damit funktioniert die bereits vorhandene Filter-Logik `participants.filter(p => p.rsvp_status === 'confirmed')` ohne Frontend-Änderung.

SQL-Pattern: `COALESCE(gr.status, CASE WHEN g.rsvp_opt_out=1 AND is_extended=0 THEN 'confirmed' ELSE NULL END)`.

**Alternative verworfen:** Zusätzliches Feld `is_implicit` zur Unterscheidung „aktiv zugesagt" vs. „implizit zugesagt". Wirft semantische Fragen auf (sollte die Tabelle ein anderes Icon zeigen?) — out of scope.

### Entscheidung 3 — `GetGame` liefert Counts mit

Auch wenn die Detail-Page heute über `GetParticipants` zählt: Counts gehören in das Game-Objekt selbst, damit künftige Konsumenten (z.B. mobile App, Embeddings) nicht extra die Participants-Liste laden müssen. Additiv, kein Breaking.

### Entscheidung 4 — `ListMyGames` Cleanup beibehalten, aber harmonisieren

`ListMyGames` muss in diesem Change auf `kader_members` umgestellt werden, damit ALLE Endpoints dieselbe Definition nutzen. Das ändert die im Kalender angezeigte Zahl ggf. für Spiele, bei denen `team_memberships` und `kader_members` verschieden waren. Für Spiel 39 (`rsvp_opt_out=1`, 19 angezeigt) gehe ich davon aus, dass Kader = Team-Members für diesen Saison-Snapshot ist und sich die Zahl nicht ändert. Falls doch — bewusst akzeptierte Korrektur.

### Entscheidung 5 — declined und maybe bleiben nur-explizit

Opt-Out heißt „du bist dabei, wenn du nichts sagst". Es gibt kein semantisches Pendant „du sagst implizit ab". Daher zählen `declined_count` und `maybe_count` weiterhin nur explizite Responses. Spec ergänzt diese Klarstellung.

## Risks / Trade-offs

**[Risiko]** `ListMyGames`-Umstellung kann den angezeigten Wert für Spiele leicht ändern, bei denen `team_memberships ≠ kader_members`. Vorstand könnte verwirrt sein.
**→ Mitigation:** Bewusst akzeptiert. Die alte Zahl war auf einer brüchigen Definition gebaut. Wenn der Vorstand fragt: „warum ist die Zahl jetzt anders" → Antwort: „weil sie jetzt überall gleich ist und dem Kader entspricht". Im Commit-Body dokumentiert.

**[Risiko]** Bestehende Tests könnten auf den alten Werten basieren.
**→ Mitigation:** Volle Test-Suite laufen lassen, betroffene Tests anpassen. Die meisten Tests setzen Kader = Team auf, dann ändert sich nichts.

**[Trade-off]** Wir aktualisieren das Frontend noch nicht — die Detail-Page rechnet weiterhin via Filter. Funktioniert, aber nicht ideal aus Architektur-Sicht.
**→ Akzeptiert:** Kein Bedarf, weil die Daten jetzt korrekt sind. Refactor in einem späteren Change.

## Migration Plan

1. `GetParticipants` SQL umbauen — opt-out wird in der Query selbst aufgelöst.
2. `ListGames` SELECT um CASE für `confirmed_count` ergänzen (declined/maybe bleiben simple counts).
3. `GetGame` SELECT um die drei Counts ergänzen, Response-Struct erweitert.
4. `ListMyGames` SQL: in der CASE-Logik `team_memberships` → `kader_members`. Außerdem `inRegularKader`-EXISTS bleibt unverändert (nutzt schon `kader_members`) — Konsistenz zwischen Count und `my_rsvp` ist nach Umstellung garantiert.
5. Tests anpassen / hinzufügen.
6. `make build && pnpm tsc --noEmit` als Sanity-Check.

**Rollback:** Reverter-Commit. Keine DB-Änderung.

## Open Questions

Keine bekannten offenen Punkte. Falls die Umstellung der `ListMyGames`-Zahl in der Praxis Verwirrung stiftet, wird das vom Live-Smoke-Test sichtbar — dann kann ein Folge-Commit eine fachliche Erklärung im UI ergänzen.
