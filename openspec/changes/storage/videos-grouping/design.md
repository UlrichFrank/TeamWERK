## Context

`videos` enthält bereits heute mehrere Zeilen pro `game_id` (nullable Foreign Key, keine UNIQUE-Constraint). Die Anzeige in der Web-UI behandelt jedes Video aber als eigenständige Karte — Nutzer sehen nicht, dass „1. Halbzeit" und „2. Halbzeit" zusammengehören, und beim Hochladen des zweiten Clips fehlt der Hinweis, dass schon einer existiert. Der Upload-Flow erlaubt seit Commit `8d4f469` Titel **oder** Spiel als Pflichtfeld, was die Geltung der Gruppierung auf reine „Titel-Videos" (Auswärtsfahrten, Trainingsmitschnitte) ausdehnt.

Lese-Berechtigungen pro Video sind durch `videos.team_id` und die rollenbasierte Filterung in `internal/videos/access.go` geregelt — die Gruppierung darf darauf nicht ausweichen, sondern muss auf der bereits gefilterten Ergebnismenge der Liste arbeiten.

## Goals / Non-Goals

**Goals:**

- Mehrere Videos pro Spiel/Titel sind in Liste und Detail als zusammengehörig erkennbar.
- Beim Upload erkennt das Frontend bestehende Gruppen und schlägt einen sinnvollen Folge-Titel vor (Pfadlenker, kein hartes Validierungs-Gate).
- Keine Schema-Änderung, keine neue Berechtigungs-Achse, keine neue serverseitige State-Maschine.
- Bestehende Permission-Filter wirken unverändert: gruppiert wird **nach** dem Auth-Filter.

**Non-Goals:**

- Keine separate „Sammlung"/„Playlist"-Entität (war Alternative; verworfen).
- Kein manuelles Umsortieren via Drag-and-Drop in dieser Iteration — Sortierung ergibt sich aus `created_at`.
- Keine Cross-Spiel-Gruppierung (z. B. „alle Pokalvideos der Saison" — wäre eine echte Sammlung, bewusst out of scope).
- Keine SSE-Live-Updates speziell für die Gruppierung — die bestehende `videos`-Liste lädt sowieso bei Bedarf nach.

## Decisions

### 1. Gruppen-Schlüssel: `game_id` mit Title-Fallback (statt neuem Feld)

**Wahl:** Gruppen-Key ist `game_id` (wenn gesetzt) oder die normalisierte `title`-Zeichenkette (`title.trim()`, case-sensitive).

**Rationale:** Vermeidet Migration und neues Pflichtfeld im Upload. Der Title-Fallback ist eine ehrliche „Best-Effort"-Heuristik; deckt den 99-%-Fall „Nutzer benennt zwei Clips gleich" ab.

**Alternativen erwogen:**

- **Neue Spalte `group_key TEXT`** — wäre sauberer (z. B. UUID pro Gruppe), erzwingt aber Upload-UX-Änderungen (Gruppen-Picker), Migration für Bestand und neue Validierungen serverseitig. Übertrieben für rein darstellende Funktion.
- **Neue Tabelle `video_collections`** — maximale Flexibilität (auch Cross-Spiel), aber doppelte Strukturen (Spiel vs. Sammlung) und CRUD-UI nötig. Vom Nutzer explizit abgewählt.

### 2. Gruppierung clientseitig, nicht serverseitig

**Wahl:** Der bestehende `GET /api/videos` liefert die flache, permission-gefilterte Liste; das Frontend gruppiert im Speicher.

**Rationale:** Die Liste ist klein (≤ einige Dutzend Videos pro Saison/Team), Gruppierung ist O(n) und vermeidet einen neuen Endpoint-Vertrag samt Pagination. Lese-Berechtigungen bleiben in einer einzigen Codepfad-Stelle (`videos.access`).

**Alternativen erwogen:** Backend liefert bereits gruppierte Antwort — bricht die bestehende Listen-API, mehr Komplexität für minimalen Mehrwert.

### 3. `GET /api/games/{id}/videos` nur optional / on-demand

**Wahl:** Wird **nicht** in der ersten Iteration implementiert. Die Geschwister-Liste auf der Video-Detailseite filtert clientseitig auf dem Ergebnis von `GET /api/videos` (oder einem Aufruf von `GET /api/videos?game_id=…`, falls die bestehende Liste das Query-Param schon kennt).

**Rationale:** Erst implementieren, wenn das Datenvolumen es rechtfertigt; sonst Code ohne aktuellen Nutzen.

**Trigger zur Nachrüstung:** sobald ein Team > 50 Videos in der gleichen Saison hat oder die Detailseite spürbar zu viel überträgt.

### 4. Default eingeklappt, Anzahl + erstes Video als Vorschau

**Wahl:** Gruppen-Karten mit > 1 Video sind initial eingeklappt; sichtbar sind Spiel-/Titel-Header + Anzahl + Thumbnail/Titel des ersten Videos.

**Rationale:** Tabellen-Lookalike, schnell überscrollbar; Detail-Klick auf ein einzelnes Video bleibt der primäre Weg ins Video.

### 5. Upload-Hinweis ist nicht-blockierend

**Wahl:** Beim Upload zeigt das Frontend einen Info-Hinweis (Alert, kein Modal, kein Disable), wenn Spiel oder Titel matchen. Der Titel-Vorschlag landet als Placeholder/Default, der User kann ihn überschreiben.

**Rationale:** Wir wissen nicht sicher, ob es wirklich „dieselbe Gruppe" ist (Titel-Heuristik) — den User nicht aussperren.

## Risks / Trade-offs

- **Title-Heuristik produziert Phantomgruppen**: zwei Videos mit zufällig identischem Titel werden gruppiert, obwohl gemeinsamer Kontext fehlt. → Mitigation: nur clientseitig dargestellt, keine Konsequenzen für Lese-/Schreibrechte; User kann den Titel ändern und entkoppelt damit die Gruppe.
- **Sortierung per `created_at` passt nicht immer zur Halbzeit-Reihenfolge** (Upload kann nachträglich erfolgen). → Mitigation: User kann durch Titelpräfix („1. ", „2. ") visuell führen; späteres Sort-Feld bleibt nachrüstbar (additiv, nicht breaking).
- **Performance bei sehr großen Listen** (>500 Videos): clientseitige Gruppierung wird langsam, Render der Karten dauert. → Aktuell unrealistisch (Retention 90 Tage); falls Schwelle erreicht, Backend-Endpoint nachziehen (Decision 3).
- **Eingeklappte Karten verstecken Inhalte vor Live-Updates**: ein neu hochgeladenes Video in einer eingeklappten Gruppe ist nicht sofort sichtbar. → Mitigation: Anzahl-Badge in der Karte aktualisiert sich; User klickt auf, wenn relevant.

## Migration Plan

1. Code-Änderung deployen — keine DB-Migration nötig.
2. Bestand: alle vorhandenen Videos werden automatisch nach den neuen Regeln gruppiert (rein darstellend).
3. Rollback: einfacher Code-Revert; keine Datenrückführung erforderlich.

## Open Questions

_keine — Datenmodell, Gruppierungs-Logik, Upload-UX und Backend-Scope sind im Proposal mit dem Nutzer abgestimmt._
