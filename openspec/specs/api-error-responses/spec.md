# api-error-responses Specification

## Purpose
Fehlerantworten der API sind maschinenlesbar, verraten keine internen Details und hinterlassen bei Serverfehlern eine Log-Spur.
## Requirements
### Requirement: Einheitliche Fehlerantworten mit Log-Spur

Handler der Domänen `games`, `trainings` und `kader` MUST Fehler als JSON `{"error": "<code>"}` beantworten. Der Body MUST frei von Datenbank- oder Laufzeitfehlertexten sein. Jede Antwort mit Status 5xx MUST eine strukturierte Log-Zeile mit Pfad, Methode, Nutzer-ID und Fehlerursache erzeugen. Listen-Routen MUST `limit` auf höchstens 200 deckeln.

#### Scenario: Serverfehler wird geloggt, nicht verraten
- **WHEN** eine Datenbankabfrage in einem migrierten Handler fehlschlägt
- **THEN** antwortet der Server mit 500 und `{"error":"internal"}` und schreibt eine Log-Zeile mit der Fehlerursache

#### Scenario: Paginierung ist gedeckelt
- **WHEN** `GET /api/games?limit=100000` aufgerufen wird
- **THEN** liefert der Server höchstens 200 Einträge

