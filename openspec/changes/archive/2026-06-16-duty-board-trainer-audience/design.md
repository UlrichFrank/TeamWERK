## Context

Die Sichtbarkeit der Dienstbörse (`GET /api/duty-board`) ist heute in `internal/duties/handler.go:333` (`Board()`) implementiert und kennt zwei orthogonale Filter-Achsen:

1. **Team-Quelle** — welche `team_id`s gelten als „meine Teams"? Aktuell:
   - `admin` und `vorstand`: alle Teams der aktiven Saison.
   - Alle anderen: Teams, in denen der Nutzer oder ein verknüpftes Familienmitglied (via `family_links`) als Spieler im Kader (`player_memberships`) eingetragen ist. Trainer-Beziehungen über `kader_trainers` werden **nicht** berücksichtigt.

2. **Audience** — auf Slot-Ebene durch `duty_slots.audiences` bzw. `duty_types.audiences` (JSON-Array mit `'eltern'`/Vereinsfunktionen). Heute gilt: für `vorstand`, `vorstand_beisitzer` und `trainer` greift ein pauschaler **Audience-Bypass** (alle Audiences sichtbar). Für alle anderen wird gefiltert: ein Slot ist sichtbar, wenn `audiences` NULL ist oder eine der eigenen Funktionen / `'eltern'` (bei Eltern) enthält.

Die View `trainer_memberships` existiert seit Migration 039 und liefert `(id, member_id, team_id, season_id)` analog zu `player_memberships`. Sie kann ohne Schema-Änderung verwendet werden.

Auf der Frontend-Seite arbeitet `web/src/pages/DutyPage.tsx` mit URL-Search-Params (`?team`, `?types`, `?mine`, `?past`). Der neue Audience-Filter folgt diesem Muster.

## Goals / Non-Goals

**Goals:**
- Trainer/sportliche Leitung sehen automatisch die Dienst-Slots **aller** Teams, die sie über `kader_trainers` trainieren.
- Privilegierte Funktionen (`vorstand`, `vorstand_beisitzer`, `trainer`, `sportliche_leitung`) sehen standardmäßig nur Slots, deren `audiences` zu einer ihrer Funktionen passt — können den Filter aber explizit deaktivieren.
- Frontend bietet einen einzigen Pill-Toggle „Nur meine Audience" für genau diese vier Funktionen, default aktiv, URL-persistiert.

**Non-Goals:**
- Kein neuer Filter für nicht-privilegierte Rollen (Spieler/Eltern). Deren Audience-Filterung war bisher und bleibt **immer aktiv** (kein Bypass möglich).
- Keine Änderung an `?view=mine`, am Team-Dropdown, an den Event-Typ-Pills oder am Past-Toggle.
- Keine Änderung am Schema, an Migrationen oder an der `trainer_memberships`-View.
- Keine Berechtigungsänderung bei Slot-Erstellung, -Claim oder -Edit.

## Decisions

### 1. Trainer-Team-Quelle: `trainer_memberships` zur UNION ergänzen

Die `whereParts`-Konstruktion in `Board()` (handler.go:354-382) wird so erweitert, dass bei nicht-admin/nicht-vorstand-Nutzern die Team-Liste zusätzlich Teams enthält, in denen der Nutzer als Trainer im Kader steht:

```sql
ds.team_id IN (
    SELECT DISTINCT tm.team_id FROM player_memberships tm
    JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
    WHERE tm.member_id IN (
        SELECT id FROM members WHERE user_id = ?
        UNION
        SELECT fl.member_id FROM family_links fl WHERE fl.parent_user_id = ?
    )
    UNION
    SELECT DISTINCT trm.team_id FROM trainer_memberships trm
    JOIN seasons s2 ON s2.id = trm.season_id AND s2.is_active = 1
    JOIN members tm_m ON tm_m.id = trm.member_id
    WHERE tm_m.user_id = ?
)
```

Analog wird die zweite Unterabfrage für game-lose Slots (über `game_teams gt`) um den Trainer-Pfad erweitert.

**Alternative verworfen:** „Trainer sehen alle Teams" (analog zu vorstand). Verworfen weil zu breit — ein Trainer von Team A soll nicht alle Slots von Team B sehen.

### 2. Audience-Self-Filter ersetzt Bypass für privilegierte Funktionen

Der bisherige `audienceBypass`-Code (handler.go:339-347, 384-403) wird umgebaut:

- **Nicht-privilegierte Nutzer** (Spieler, Eltern ohne Funktion): wie bisher — Audience-Filter immer aktiv, nicht abschaltbar.
- **Privilegierte Funktionen** (`vorstand`, `vorstand_beisitzer`, `trainer`, `sportliche_leitung`): Audience-Filter standardmäßig aktiv, aber per `?audience=all` abschaltbar.
- **`admin`** (System-Rolle): immer Bypass, unabhängig vom Query-Param. Admin-Diagnose darf nie eingeschränkt werden.

Der bestehende Audience-Match-Block (Zeile 385-401) bleibt wiederverwendbar — der einzige Unterschied ist, **wann** er angehängt wird:

```go
audienceBypass := claims.Role == "admin"
audienceMode := r.URL.Query().Get("audience") // "", "mine", "all"
isPrivileged := hasAnyFunction(claims, "vorstand", "vorstand_beisitzer", "trainer", "sportliche_leitung")

if !audienceBypass && !(isPrivileged && audienceMode == "all") {
    // bisheriger Audience-Filter-Block
}
```

Default-Verhalten ohne `?audience`-Param ist **mine** für Privilegierte (= Filter aktiv). Wer explizit `?audience=all` sendet UND privilegiert ist, sieht alle Audiences. Spieler/Eltern können den Filter nicht abschalten (Query-Param wird ignoriert).

**Alternative verworfen:** Separater Query-Param `?audience_all=1`. Verworfen zugunsten von `?audience=mine|all` — symmetrischer, erweiterbar (z.B. später `?audience=eltern` für „nur Eltern-Slots zeigen"), passt zum bestehenden `?view=mine`.

### 3. Frontend-Pille mit Sichtbarkeitsbedingung

`DutyPage.tsx` erhält eine neue Pille „Nur meine Audience" mit `Filter`-Icon (Lucide). Die Pille:

- Ist **nur sichtbar**, wenn `hasFunction(user, fn)` für mindestens eine der vier privilegierten Funktionen true ist (Helper in `AuthContext.tsx` existiert bereits).
- Default-Zustand: **aktiv** (= „nur meine Audience" / kein URL-Param).
- Bei Deaktivierung wird `?audience=all` in die URL geschrieben und an `/api/duty-board?audience=all` weitergereicht.
- URL-Persistenz analog zu `?mine`/`?past`: `parseFilters` wird um `audience` erweitert; `updateFilter` schreibt/löscht den Param.

Position in der Pill-Leiste: in der „Toggle"-Gruppe rechts neben „Meine"/„Vergangene" — visuell als weiterer Schalter.

**Alternative verworfen:** Versteckter Default-Param. Konvention der `DutyPage`: Default-State erzeugt keine URL-Params. Konsistent damit ist die „Filter aktiv" = leere URL, „Filter aus" = `audience=all`.

### 4. Hilfsfunktion `hasAnyFunction` im auth-Package

Aktuelle `claims.HasFunction(string)` prüft eine Funktion. Wir ergänzen `func (c *Claims) HasAnyFunction(fns ...string) bool` für die Privileg-Prüfung in `Board()` und potenziell anderswo. Trivialimplementierung.

## Risks / Trade-offs

- **Risiko: Trainer der gleichzeitig Spieler eines anderen Teams ist sieht plötzlich mehr.** → Mitigation: Das ist gewünscht. Vorher waren beide Teams sichtbar, soweit Spieler — Trainer-Beziehung wurde nur ignoriert. Nach der Änderung kommen Trainer-Teams dazu, das Spieler-Team bleibt. Keine Sichtbarkeitsregression.

- **Risiko: Vorstand mit Funktion `vorstand` verliert Sichtbarkeit auf Spieler-/Eltern-Slots, wenn Filter aktiv ist.** → Mitigation: Vorstand-Audience ist eine eigene Funktion. Slots, die für Eltern gemeint sind, sind für den Vorstand normalerweise auch nicht zum Übernehmen gedacht — wenn er sie sehen will, schaltet er den Filter aus. Default ist das gewünschte „nur was mich angeht".

- **Risiko: Erweiterte SQL-Query wird teurer.** → Mitigation: `trainer_memberships` ist eine View über indexierte Spalten (`kader_trainers.member_id`, `kader.team_id`). Bei der typischen Vereinsgröße (≪ 1000 Member, ≪ 50 Teams) vernachlässigbar.

- **Risiko: Frontend-Pille wird für privilegierte Nutzer angezeigt, obwohl Backend `?audience=all` aus anderen Gründen ignoriert.** → Mitigation: Die Bedingung ist symmetrisch (gleiche Funktions-Liste), und für `admin` wird zusätzlich Bypass im Backend erzwungen — Pille hat dann keinen Effekt, aber das ist ein No-Op, kein Fehler.

- **Trade-off: Default-aktiv vs. default-aus.** Default-aktiv reduziert visuelle Last und passt zum Wunsch des Nutzers, schafft aber eine implizite Vorannahme „du willst nur deine Audience sehen". Wird durch Sichtbarkeit der Pille jederzeit transparent (Nutzer sieht: „Filter ist an, ich kann ihn deaktivieren").

## Migration Plan

Keine Datenbank-Migration nötig. Deployment-Schritte:

1. Backend-Patch + Tests → `go test ./internal/duties/...` grün.
2. Frontend-Patch → `pnpm build` grün.
3. `make build && make deploy`.
4. Rollback: einfache Git-Reverts; keine Datenseitigen Effekte.

## Open Questions

- Soll die Pille auch dann sichtbar sein, wenn der Nutzer aktuell `view=mine` aktiviert hat? → Vorschlag: Ja, kombinierbar lassen. „Meine übernommenen Dienste" + „Audience-Filter" wirken multiplikativ und sind beide sinnvoll, auch wenn `mine` in den meisten Fällen den Audience-Filter redundant macht.
