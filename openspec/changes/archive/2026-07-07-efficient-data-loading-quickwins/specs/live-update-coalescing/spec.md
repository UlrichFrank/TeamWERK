## ADDED Requirements

### Requirement: Coalescing von Live-Update-Events

Das Frontend SHALL eingehende SSE-Events aus `useLiveUpdates` über ein kurzes Zeitfenster bündeln, sodass ein Burst gleichartiger Events genau eine Reload-Aktion je eindeutigem Event-Typ auslöst. Kein Event-Typ SHALL dabei verloren gehen — die im Fenster gesammelten, deduplizierten Typen werden nach Fensterablauf gemeinsam ausgeliefert. Der bestehende `__version:`-Sonderfall (Deploy-Erkennung) bleibt unberührt.

#### Scenario: Burst gleicher Events löst einen Reload aus

- **WHEN** innerhalb des Coalescing-Fensters mehrere Events desselben Typs (z. B. `duties`) eintreffen
- **THEN** wird die registrierte Callback für diesen Typ genau einmal aufgerufen

#### Scenario: Verschiedene Event-Typen bleiben erhalten

- **WHEN** innerhalb des Fensters Events unterschiedlicher Typen (z. B. `games` und `trainings`) eintreffen
- **THEN** wird die Callback für jeden eindeutigen Typ genau einmal aufgerufen

#### Scenario: Versions-Event umgeht das Coalescing

- **WHEN** ein `__version:`-Event eintrifft
- **THEN** wird es unmittelbar wie bisher verarbeitet und nicht in das Coalescing-Fenster einbezogen
