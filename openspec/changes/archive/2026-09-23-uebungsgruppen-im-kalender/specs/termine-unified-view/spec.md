## ADDED Requirements

### Requirement: Ladefenster der Terminliste

Das System SHALL die Terminliste über ein Datumsfenster laden, dessen obere Grenze das
**spätere** von Saisonende und „heute + 365 Tage" ist.

Die Saison begrenzt das Fenster nach oben nur, solange sie weiter reicht als das rollierende
Jahr. Ein Termin, dessen Datum jenseits des Endes seiner Saison liegt — zulässig, weil
`training_sessions` die Saison als Spalte und nicht als Datumsbedingung führt, und in der
Praxis bei über das Saisonende hinauslaufenden Übungsgruppen-Serien der Fall — MUSS in der
Liste erscheinen.

Die untere Grenze bleibt unverändert: ohne den Toggle „Vergangene" ist es der heutige Tag,
mit Toggle der frühere von Saisonstart und „heute − 365 Tage".

#### Scenario: Termin nach dem Saisonende

- **WHEN** die aktive Saison am 30.06.2027 endet und ein Trainingstermin dieser Saison auf
  den 03.07.2027 fällt
- **THEN** wird er von der Terminliste geladen und angezeigt

#### Scenario: Saison reicht über das rollierende Jahr hinaus

- **WHEN** die aktive Saison später als „heute + 365 Tage" endet
- **THEN** reicht das Ladefenster bis zum Saisonende

### Requirement: Zeilenlimit der Termin-Abfrage

Das System SHALL `GET /api/training-sessions` mit einem Limit von höchstens 1000 Zeilen pro
Abfrage beantworten (Default 100).

Das frühere Limit von 200 lag unter der Zahl der Termine, die ein Trainer mit mehreren Kadern
oder ein Funktionsträger (admin/vorstand/sportliche_leitung, die alle Termine sehen) in einer
Saison hat; die Liste wurde ohne Hinweis abgeschnitten.

#### Scenario: Mehr als 200 sichtbare Termine

- **WHEN** ein Nutzer im geladenen Fenster 285 sichtbare Trainingstermine hat und die Liste
  mit `limit=500` abgefragt wird
- **THEN** enthält die Antwort alle 285 Termine
