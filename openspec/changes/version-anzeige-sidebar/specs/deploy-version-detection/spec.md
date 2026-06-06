## MODIFIED Requirements

### Requirement: SSE-Versionscheck erkennt neuen Deployment-Stand

Das Frontend SHALL beim Verbindungsaufbau und bei jedem SSE-Reconnect die empfangene Versionsinformation mit der zuerst bekannten Version vergleichen. Bei Abweichung SHALL der Update-Banner angezeigt werden. Der Hook SHALL neben `updateAvailable` auch die aktuell bekannte `version` (erster empfangener Hash, oder `null` vor dem ersten Event) zurückgeben. Die SSE-Verbindung SHALL nach Verbindungsabbruch automatisch reconnecten; der Hook SHALL `EventSource.close()` NICHT im `onerror`-Handler aufrufen.

#### Scenario: Erste Verbindung speichert Baseline-Version

- **WHEN** der SSE-Client zum ersten Mal verbindet und ein `__version:`-Event empfängt
- **THEN** wird dieser Hash als bekannte Version gespeichert
- **THEN** gibt der Hook `{ updateAvailable: false, version: "<hash>" }` zurück

#### Scenario: SSE-Reconnect nach Server-Neustart zeigt Banner

- **WHEN** die SSE-Verbindung nach einem Server-Neustart (deploy) neu aufgebaut wird
- **WHEN** der neue Server einen anderen Hash als die gespeicherte Version sendet
- **THEN** gibt der Hook `{ updateAvailable: true, version: "<alter-hash>" }` zurück
- **THEN** wird der Update-Banner angezeigt

#### Scenario: Reconnect ohne Versionsänderung zeigt keinen Banner

- **WHEN** die SSE-Verbindung kurzzeitig unterbrochen und wieder hergestellt wird
- **WHEN** der Server denselben Hash sendet wie die gespeicherte Version
- **THEN** gibt der Hook `{ updateAvailable: false, version: "<hash>" }` zurück
- **THEN** bleibt der Update-Banner ausgeblendet

#### Scenario: Dev-Modus unterdrückt den Banner

- **WHEN** die App im Dev-Modus läuft (`import.meta.env.DEV === true`)
- **THEN** gibt der Hook `{ updateAvailable: false, version: null }` zurück
- **THEN** wird der Update-Banner unabhängig von Versionsänderungen nicht angezeigt
