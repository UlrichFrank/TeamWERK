## Why

Die erste Kader-Implementation hat drei grundlegende Lücken: die Jahrgangszuweisung ist falsch berechnet (überlappende Altersklassen statt DHB-konformer 2-Jahres-Sprünge), einzelne Kader können weder angelegt noch gelöscht werden, und das Konzept der „Jahrgangsmischung vs. dedizierter Jahrgänge" fehlt vollständig — was die Kaderplanung für den tatsächlichen Spielbetrieb unbrauchbar macht.

## What Changes

- **Jahrgangskalkulation korrigieren**: `ageBracketRef2025` erhält korrekte, nicht-überlappende Basiswerte (A=2007/08, B=2009/10, C=2011/12, D=2013/14); die Season-Offset-Formel bleibt erhalten
- **Kader-Schema erweitern**: neue Spalten `team_number` (1 oder 2) und `dedicated_birth_year` (NULL = gemischt, Jahreszahl = dediziert); UNIQUE-Constraint auf `(season_id, age_class, gender, team_number)` aktualisiert
- **Neuer POST-Endpoint** für das Anlegen eines einzelnen Kaders mit explizitem `team_number` und `dedicated_birth_year`
- **Neuer DELETE-Endpoint** für das Löschen eines Kaders; schlägt mit 409 fehl wenn noch Mitglieder zugeordnet sind
- **Filterlogik angepasst**: `suggestMembers` und `autoAssignMembers` verwenden `dedicated_birth_year` wenn gesetzt, sonst den vollen Bracket
- **Frontend-Kachel**: zeigt zugeordnete Jahrgänge an, enthält Modus-Umschalter (gemischt / dediziert) und Löschen-Button
- **Frontend-Neuanlage**: Button „+ Mannschaft anlegen" pro Altersklasse/Geschlecht-Gruppe öffnet Dialog mit Auswahl von Nummer und Jahrgang

## Capabilities

### New Capabilities

- `kader-year-mode`: Konfiguration ob ein Kader alle Jahrgänge der Altersklasse umfasst (Jahrgangsmischung) oder einem einzelnen Jahrgang gewidmet ist (dediziert); Filtervorschläge und Auto-Assign berücksichtigen den gewählten Modus
- `kader-lifecycle`: Einzelne Kader anlegen und löschen (Löschen nur wenn leer)

### Modified Capabilities

- `kader-age-brackets`: Korrekte DHB-konforme Jahrgangszuweisung; bisherige Berechnung war falsch (überlappend, falsche Basisjahre)

## Impact

- **DB**: neue Migration für `kader`-Tabelle (2 Spalten + neuer Unique-Index)
- **Backend**: `internal/kader/age_brackets.go`, `internal/kader/handler.go`, `internal/kader/suggestions.go`, `internal/kader/copy.go`
- **Frontend**: `web/src/pages/AdminKaderPage.tsx`, `web/src/components/KaderMemberSearch.tsx`
- **Tests**: `internal/kader/age_brackets_test.go` muss aktualisiert werden (neue Referenzwerte)
- Keine neuen Dependencies; kein Breaking Change an bestehenden API-Clients (team_number=1 ist Default)
