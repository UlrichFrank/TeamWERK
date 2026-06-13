## Context

Der CSV-Import-Handler (`internal/members/handler.go`, Funktion `Import`, ~400 Zeilen) unterstützt aktuell zwei Modi: `append` (nur Anlegen) und `update` (Felder überschreiben wenn CSV-Wert vorhanden). Beide Modi teilen sich eine gemeinsame CSV-Parse-Schleife; der Modus steuert lediglich den Schreib-Pfad am Ende jeder Iteration.

Der `enrich`-Modus ist eine Variante von `update` mit zwei Unterschieden: kein Anlegen bei fehlendem Match, und beim Schreiben wird pro Feld geprüft ob der DB-Wert leer ist.

## Goals / Non-Goals

**Goals:**
- Dritter Import-Modus `enrich` der nur leere DB-Felder befüllt
- Kein Anlegen neuer Mitglieder im `enrich`-Modus
- Matching optional mit Geburtsdatum; ohne DOB Fallback auf Name mit Ambiguity-Check
- `not_found`-Status und Zähler im Report
- Minimale Änderung am bestehenden Code-Pfad

**Non-Goals:**
- Kein neues Matching-Konzept (kein Fuzzy-Match, kein Passnummer-Matching)
- Keine Änderung am `append`- oder `update`-Verhalten
- Kein neues API-Endpoint (derselbe `POST /api/members/import`)
- Keine DB-Migration

## Decisions

### Entscheidung: Enrich als dritte Branch in der bestehenden Import-Funktion
Der `enrich`-Modus wird als dritter `if/else`-Zweig in der vorhandenen Schleife implementiert, nicht als separate Funktion.

**Alternativen:** Neue Handler-Funktion für `enrich`. Abgelehnt, weil CSV-Parsing, BOM-Strip, Delimiter-Detection und Duplikat-Tracking identisch sind — eine eigene Funktion würde Code duplizieren.

### Entscheidung: Pro-Feld-Leer-Prüfung via Hilfsfunktion
Beim Schreiben eines Feldes im `enrich`-Modus wird eine Hilfsfunktion `isDBFieldEmpty(value interface{}) bool` genutzt, die `sql.NullString.Valid == false || == ""` sowie `sql.NullInt64.Valid == false` erkennt. Diese Funktion wird konsistent für alle Felder verwendet.

**Alternativen:** Inline-Prüfungen pro Feld. Abgelehnt wegen Wiederholung und fehleranfälliger Inkonsistenz bei ~15 Feldern.

### Entscheidung: Matching-Fallback nur wenn kein DOB-Spalte vorhanden
Der DOB-Fallback greift nur wenn die CSV keine `Geburtsdatum`-Spalte enthält (nicht wenn die Spalte leer ist). Damit bleibt das Verhalten vorhersehbar: eine CSV mit DOB-Spalte matcht immer via Name+DOB.

**Alternativen:** Fallback auch bei leerem DOB-Wert. Abgelehnt, weil ein leeres DOB-Feld bei expliziter DOB-Spalte eher ein Datenfehler ist als ein „kein DOB bekannt"-Signal.

### Entscheidung: `not_found` ist kein Fehler
`not_found`-Zeilen erhöhen `report.NotFound`, nicht `report.Errors`. Fehler sind Validierungsprobleme (ungültige Daten); kein Match ist ein erwartetes Ergebnis im `enrich`-Modus.

## Risks / Trade-offs

- **Name-Kollision ohne DOB:** Bei gleichnamigen Mitgliedern ohne Geburtsdatum in der CSV wird `error: mehrdeutig` zurückgegeben. Das ist korrekt aber kann für Nutzer überraschend sein. → Mitigation: klare Fehlermeldung mit Anzahl der Treffer.
- **Bestehende `update`-Semantik unverändert:** `update` überschreibt weiterhin. Nutzer könnten verwirrt sein welchen Modus sie wählen sollen. → Mitigation: klare Radio-Button-Labels und Hilfetexte im UI.
- **`ImportReport`-Struct-Erweiterung ist API-additiv:** Frontend muss `not_found`-Feld kennen. Da beide zeitgleich deployed werden (embedded Binary), kein Kompatibilitätsproblem.
