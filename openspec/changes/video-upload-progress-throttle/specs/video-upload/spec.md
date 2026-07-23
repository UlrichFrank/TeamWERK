## ADDED Requirements

### Requirement: Client-seitige Progress-Throttle

Die Video-Upload-Seite (`VideoUploadPage`) SHALL den vom tus-Client gelieferten `onProgress`-Callback throttlen, sodass React-`setState`-Aufrufe für Fortschritt und Restzeit **maximal 1× pro Sekunde** erfolgen. Der Prozent-Wert MUST zusätzlich gegen den letzten publizierten Wert verglichen werden — bei unverändertem gerundeten Prozentwert MUST kein State-Update erfolgen. Die Restzeit-Schätzung MUST auf einem Sliding-Window der letzten ~10 Sekunden Upload-Rate basieren, nicht auf dem Gesamt-Durchschnitt seit Upload-Start.

#### Scenario: Häufige Progress-Events lösen nur ein State-Update pro Sekunde aus
- **WHEN** der tus-Client `onProgress` 100× innerhalb von 100 ms aufruft
- **THEN** wird `setProgress` genau einmal aufgerufen und `setRemaining` genau einmal aufgerufen

#### Scenario: Gleicher Prozentwert löst kein zusätzliches Update aus
- **WHEN** zwei `onProgress`-Aufrufe im Abstand von mehr als 1 Sekunde denselben gerundeten Prozentwert liefern
- **THEN** wird `setProgress` nur beim ersten Aufruf aufgerufen

#### Scenario: Restzeit reagiert auf Bandbreiten-Änderung
- **WHEN** der Upload-Durchsatz nach 5 Minuten stabilem Verlauf für 15 Sekunden auf ein Drittel einbricht
- **THEN** reflektiert die angezeigte Restzeit innerhalb dieser 15 Sekunden den geringeren Durchsatz (Fenster-basierte Rate statt Gesamt-Durchschnitt)

#### Scenario: Refs werden zwischen zwei Uploads zurückgesetzt
- **WHEN** ein Nutzer nach einem erfolgreichen Upload ohne Seiten-Reload ein zweites Video hochlädt
- **THEN** startet der zweite Upload mit frischen Throttle-Refs (Timestamp = 0, Sample-Ring leer, letzter Prozentwert = -1) und die Progress-Anzeige beginnt sauber bei 0 %
