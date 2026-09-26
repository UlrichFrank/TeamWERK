## ADDED Requirements

### Requirement: Dienstbörse lädt unabhängig von Zusagen- und Termin-Team-Anzahl
`GET /api/duty-board` SHALL die Team-Zugehörigkeit eines Dienst-Slots (für Sichtbarkeit,
Eltern-Zielgruppe und Aushilfe-Kennzeichen von Gruppe und Eingetragenen) je Slot über die
Teams **seines** Termins prüfen. Der Aufwand je Slot bzw. je Zusage SHALL nicht mit der
Gesamtzahl der Termin-Team-Zuordnungen der Saison wachsen, und die Antwortzeit SHALL nicht
davon abhängen, ob die Datenbank Planer-Statistiken (`ANALYZE`) oder zusätzliche Indizes
trägt. Dieselbe Anforderung gilt für das Aushilfe-Prädikat des Dashboards („Meine Dienste",
Block „Aushilfe"), das dieselbe Regel auswertet.

Die fachliche Antwort SHALL unverändert bleiben: dieselben Gruppen, Slots, Eingetragenen und
dieselben `aushilfe`-Kennzeichen wie vor der Änderung, für jede Persona.

#### Scenario: Aushilfe-Kennzeichen der Eingetragenen durchsucht nicht alle Termin-Teams
- **WHEN** der Ausführungsplan der Abfrage erstellt wird, die die Eingetragenen der
  Dienstbörse samt Aushilfe-Kennzeichen liefert
- **THEN** enthält er keinen vollständigen Scan der Termin-Team-Zuordnungen
  (`game_teams`), sondern greift je Slot über dessen Termin darauf zu

#### Scenario: Plan bleibt nach Statistik-Erhebung gleich
- **WHEN** auf der Datenbank `ANALYZE` ausgeführt wurde und der Ausführungsplan derselben
  Abfrage erneut erstellt wird
- **THEN** enthält er weiterhin keinen vollständigen Scan von `game_teams`

#### Scenario: Ergebnis unverändert
- **WHEN** Admin, Trainer, ein Elternteil mit Kind im Stammkader und ein Elternteil mit
  Kind nur im erweiterten Kader die Dienstbörse abrufen
- **THEN** erhalten sie dieselben Gruppen, Slots, Eingetragenen und `aushilfe`-Kennzeichen
  wie mit der bisherigen Formulierung des Team-Prädikats
