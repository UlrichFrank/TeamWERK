## Context

`GET /api/mitfahrgelegenheiten` gibt derzeit alle zukünftigen Spiele zurück, unabhängig von der Team-Zugehörigkeit des anfragenden Nutzers. Die Dashboard-Logik (`internal/dashboard/handler.go`) löst dasselbe Problem bereits korrekt mit einer rollenabhängigen `teamQueryForUser()`-Hilfsfunktion. Diese Logik muss in den Carpooling-Handler übertragen werden.

## Goals / Non-Goals

**Goals:**
- `GET /api/mitfahrgelegenheiten` filtert Spiele nach Rollen-Logik: elternteil/spieler sehen nur ihre Teams, trainer nur ihre Kaderteams, admin/vorstand sehen alle
- Frontend-Toggle "Alle" → "Team" umbenennen (kein Logik-Change)
- Dashboard-Carpooling-Hint funktioniert korrekt, sobald family_links und kader_members für Testnutzer befüllt sind

**Non-Goals:**
- Keine DB-Migration
- Kein neues Package für gemeinsame Team-Query-Logik (Duplizierung akzeptabel bei zwei Callpoints)
- Dashboard-Hint-Query bleibt unverändert

## Decisions

### Team-Query-Logik duplizieren statt extrahieren

Die `teamQueryForUser()`-Logik aus `dashboard/handler.go` wird als private Hilfsfunktion in `carpooling/handler.go` kopiert, anstatt in ein gemeinsames `internal/teamquery`-Package ausgelagert zu werden.

**Rationale:** Nur zwei Callpoints (dashboard + carpooling). Ein eigenes Package für zwei Verwender ist premature abstraction. Falls ein dritter Caller entsteht, kann extrahiert werden.

**Alternative verworfen:** `internal/teamquery`-Package — zu früh, erhöht Komplexität ohne echten Mehrwert.

### Aktive Saison für Filter verwenden

Der Carpooling-Filter nutzt die aktive Saison (wie der Dashboard-Hint), nicht alle Saisons. Spiele aus inaktiven Saisons werden für reguläre Nutzer ausgeblendet.

**Rationale:** Konsistenz mit Dashboard; inaktive Saisons sind abgeschlossen und nicht relevant für laufende Carpooling-Planung.

### Admin/Vorstand sehen alle Spiele (kein Filter)

Rollen `admin` und `vorstand` erhalten keine Team-Einschränkung und sehen alle Spiele aller Saisons (bisheriges Verhalten).

## Risks / Trade-offs

- **Risiko: family_links oder kader_members unvollständig** → Elternteile/Spieler ohne Datenzuordnung sehen leere Liste. Mitigation: Beim Leeren der Liste wird kein Fehler geworfen; ein erklärender Leerstate im Frontend hilft.
- **Duplizierung der teamQueryForUser-Logik** → bei zukünftigen Schema-Änderungen (z.B. neue Rollen) muss an zwei Stellen angepasst werden. Akzeptabel bei aktuellem Scope.
