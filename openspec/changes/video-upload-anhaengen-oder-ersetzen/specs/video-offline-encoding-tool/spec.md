## ADDED Requirements

### Requirement: Hinzufügen-oder-Ersetzen-Auswahl bei bereits vorhandenem Video

Sobald in der Spiel-Auswahl ein Spiel gewählt ist, zu dem laut `GET /api/videos`
bereits (ein) Video(s) existieren, SHALL das Tool eine Auswahl zwischen **„Hinzufügen"**
(Voreinstellung — es entsteht ein weiteres, eigenständiges Video) und **„Ersetzen"**
anzeigen. Ist kein Spiel gewählt oder existiert zum gewählten Spiel noch kein Video, SHALL
diese Auswahl verborgen bleiben und das Verhalten ist unverändert „Hinzufügen".

#### Scenario: Spiel ohne vorhandenes Video

- **WHEN** ein Nutzer ein Spiel auswählt, zu dem noch kein Video existiert
- **THEN** bleibt die Hinzufügen/Ersetzen-Auswahl verborgen und ein Start legt ein neues
  Video an wie bisher

#### Scenario: Spiel mit vorhandenem Video

- **WHEN** ein Nutzer ein Spiel auswählt, zu dem bereits ein oder mehrere Videos
  existieren
- **THEN** erscheint die Auswahl „Hinzufügen"/„Ersetzen" mit „Hinzufügen" als
  Voreinstellung

#### Scenario: Wechsel zurück auf „Freier Titel"

- **WHEN** der Nutzer nach Auswahl eines Spiels mit vorhandenem Video wieder „Freier
  Titel" wählt
- **THEN** verschwindet die Auswahl wieder; ein Start legt wie bisher immer ein neues
  Video ohne Spiel-Zuordnung an

### Requirement: Bestätigung vor dem Ersetzen

Ist „Ersetzen" gewählt, SHALL das Tool vor dem Start des Encode-/Upload-Vorgangs eine
explizite Bestätigung einholen, die auf die unwiderrufliche Löschung der zuvor
vorhandenen Video(s) hinweist. Bricht der Nutzer die Bestätigung ab, SHALL kein Encode-,
Upload- oder Löschvorgang beginnen.

#### Scenario: Nutzer bestätigt

- **WHEN** der Nutzer bei gewähltem „Ersetzen" auf „Encodieren und hochladen" klickt und
  die Bestätigung annimmt
- **THEN** startet der reguläre Encode-/Upload-Vorgang

#### Scenario: Nutzer bricht die Bestätigung ab

- **WHEN** der Nutzer die Bestätigung ablehnt
- **THEN** bleibt der Zustand unverändert — kein Encode, kein Upload, keine Löschung, und
  der Nutzer kann die Auswahl erneut ändern

### Requirement: Löschung erst nach erfolgreichem neuen Upload

Bei gewähltem „Ersetzen" SHALL das Tool das/die zuvor vorhandene(n) Video(s) des
gewählten Spiels erst löschen, nachdem der neue Upload (Video-Anlage per
`POST /api/videos` UND vollständiger tus-Datei-Transfer) erfolgreich abgeschlossen ist.
Das Tool SHALL niemals vor oder während eines laufenden Uploads löschen. Das gerade neu
hochgeladene Video SHALL dabei unter keinen Umständen mitgelöscht werden.

#### Scenario: Upload schlägt fehl

- **WHEN** bei gewähltem „Ersetzen" der Encode- oder Upload-Vorgang mit einem Fehler
  abbricht
- **THEN** bleiben alle zuvor vorhandenen Videos des Spiels unangetastet

#### Scenario: Upload erfolgreich, danach Löschung

- **WHEN** bei gewähltem „Ersetzen" der neue Upload erfolgreich abgeschlossen ist
- **THEN** ruft das Tool anschließend `DELETE /api/videos/{id}` für jedes zuvor
  vorhandene Video dieses Spiels auf

#### Scenario: Mehrere vorhandene Videos

- **WHEN** zum gewählten Spiel mehrere Videos existieren (z. B. zwei Halbzeiten) und der
  neue Upload erfolgreich ist
- **THEN** werden alle zuvor vorhandenen Videos dieses Spiels gelöscht, nicht nur eines

### Requirement: Fehlgeschlagene Löschung ist ein Hinweis, kein Fehlschlag

Schlägt nach einem erfolgreichen neuen Upload das Löschen eines zuvor vorhandenen Videos
fehl (z. B. fehlende Berechtigung oder Netzwerkfehler), SHALL das Tool den Gesamtlauf
dennoch als Erfolg melden und zusätzlich einen Hinweis anzeigen, das betroffene Video
manuell in TeamWERK zu löschen.

#### Scenario: Löschung scheitert an fehlender Berechtigung

- **WHEN** ein Nutzer mit der Vereinsfunktion `sportliche_leitung` (darf hochladen, aber
  laut `video-management` keine Videos löschen) „Ersetzen" wählt und der neue Upload
  erfolgreich ist
- **THEN** meldet das Tool den Upload als erfolgreich abgeschlossen und weist zusätzlich
  darauf hin, dass das alte Video manuell gelöscht werden muss

#### Scenario: Löschung scheitert an einem Netzwerkfehler

- **WHEN** die Verbindung beim Löschversuch abbricht
- **THEN** meldet das Tool den Upload weiterhin als erfolgreich, mit demselben Hinweis auf
  eine erforderliche manuelle Löschung
