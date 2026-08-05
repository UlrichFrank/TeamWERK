## ADDED Requirements

### Requirement: Ein optionaler Nachweis je Eintrag, auch nachträglich

Das System SHALL je Trainingstagebuch-Eintrag **höchstens eine** Nachweisdatei speichern. Der
Nachweis SHALL über `POST /api/training-diary/{id}/proof` (multipart, Feld `proof`) jederzeit
angefügt werden können — beim Anlegen des Eintrags ebenso wie beliebig lange danach.

Ein erneuter Upload auf denselben Eintrag SHALL den bisherigen Nachweis ersetzen und dessen Datei
von der Platte entfernen. `DELETE /api/training-diary/{id}/proof` SHALL den Nachweis entfernen und
den Eintrag im Übrigen unverändert lassen.

Ausschließlich der **Eigentümer** des Eintrags darf einen Nachweis hochladen, ersetzen oder
löschen; alle anderen erhalten HTTP 403, auch wenn sie den Eintrag lesen dürfen.

Ein fehlender Nachweis SHALL **kein Fehlerzustand** sein: Einträge ohne Nachweis sind gültig,
vollwertig und gehen unverändert in alle Auswertungen ein.

#### Scenario: Nachweis nachträglich anfügen

- **WHEN** der Eigentümer `POST /api/training-diary/{id}/proof` auf einen zwei Wochen alten
  Eintrag ohne Nachweis aufruft
- **THEN** antwortet das System mit HTTP 201
- **THEN** liegt die Datei im Speicherverzeichnis und der Eintrag verweist darauf

#### Scenario: Nachweis ersetzen

- **WHEN** der Eigentümer einen zweiten Nachweis auf denselben Eintrag hochlädt
- **THEN** antwortet das System mit HTTP 201
- **THEN** ist die zuvor gespeicherte Datei von der Platte entfernt

#### Scenario: Nachweis entfernen

- **WHEN** der Eigentümer `DELETE /api/training-diary/{id}/proof` aufruft
- **THEN** antwortet das System mit HTTP 204
- **THEN** existiert der Eintrag weiterhin mit unveränderten Werten für Art, Dauer und RPE

#### Scenario: Fremder Upload

- **WHEN** ein Trainer des Kaders einen Nachweis auf den Eintrag eines Spielers hochlädt
- **THEN** antwortet das System mit HTTP 403
- **THEN** wurde keine Datei geschrieben

#### Scenario: Eintrag ohne Nachweis ist vollwertig

- **WHEN** ein Eintrag ohne Nachweis angelegt wird
- **THEN** erscheint er in der eigenen Liste und in der Mannschaftsübersicht mit allen Kennzahlen

---

### Requirement: Bilder werden vor dem Upload stark verkleinert

Das Frontend SHALL Bilddateien vor dem Upload über `compressImage` verkleinern, mit einer Zielgröße
von **150 KB** und einer längsten Kante von **1280 px** — deutlich enger als die Vorgaben für Chat
und Mitteilungen (1 MB / 1920 px). Die Reihenfolge der Ausgabeformate (WebP vor JPEG) bleibt
unverändert.

Nicht-Bilddateien SHALL unverändert übertragen werden.

#### Scenario: Großes Foto wird verkleinert

- **WHEN** ein Spieler ein 4 MB großes Foto als Nachweis auswählt
- **THEN** wird `compressImage` mit `targetBytes` 150 KB und `maxEdge` 1280 aufgerufen
- **THEN** überträgt der Client das verkleinerte Ergebnis

#### Scenario: PDF bleibt unangetastet

- **WHEN** ein Spieler ein PDF als Nachweis auswählt
- **THEN** wird die Datei unverändert übertragen

---

### Requirement: Der Server begrenzt Typ und Größe unabhängig vom Client

Das System SHALL den Inhaltstyp jeder hochgeladenen Datei per Content-Sniffing bestimmen und nur
`image/jpeg`, `image/png`, `image/webp` sowie `application/pdf` annehmen. Jeder andere erkannte Typ
— ausdrücklich auch `image/heic` und `image/heif` — SHALL mit HTTP 400 abgelehnt werden, ohne dass
eine Datei geschrieben wird.

Unabhängig vom Typ SHALL eine Datei über **1 MB** mit HTTP 413 abgelehnt werden.

Die Prüfung SHALL sich nicht auf Dateiendung oder den vom Client gemeldeten `Content-Type`
verlassen.

#### Scenario: Nicht unterstütztes Bildformat

- **WHEN** ein Client eine HEIC-Datei hochlädt, weil die Browser-Engine sie nicht wandeln konnte
- **THEN** antwortet das System mit HTTP 400 und einer Meldung, dass das Format nicht unterstützt
  wird
- **THEN** wurde keine Datei geschrieben

#### Scenario: Datei über dem Limit

- **WHEN** ein Client eine 2 MB große Datei hochlädt
- **THEN** antwortet das System mit HTTP 413

#### Scenario: Falsch deklarierter Inhaltstyp

- **WHEN** ein Client eine ausführbare Datei mit dem `Content-Type` `image/jpeg` und der Endung
  `.jpg` hochlädt
- **THEN** erkennt das System den tatsächlichen Typ und antwortet mit HTTP 400

---

### Requirement: Nachweise liegen in einem eigenen, zugriffsgeschützten Speicher

Das System SHALL Nachweisdateien unter dem konfigurierbaren Verzeichnis `TRAINING_DIARY_DIR`
(Default `./storage/training-diary`) unter einem zufälligen Namen (`<uuid>.<ext>`) ablegen. Es
SHALL dafür **weder die `media`-Tabelle noch das Medien-Verzeichnis** verwenden.

Die Auslieferung erfolgt ausschließlich über `GET /api/training-diary/{id}/proof` und SHALL
dieselbe Zugriffsprüfung anwenden wie der Eintrag selbst (siehe `trainingstagebuch-sichtbarkeit`).
Die Antwort SHALL den gespeicherten Inhaltstyp und den Header `X-Content-Type-Options: nosniff`
tragen.

Der Dateiname auf der Platte SHALL keinen Rückschluss auf Mitglied, Eintrag oder Datum erlauben.

#### Scenario: Eigentümer ruft seinen Nachweis ab

- **WHEN** der Eigentümer `GET /api/training-diary/{id}/proof` aufruft
- **THEN** antwortet das System mit HTTP 200, den Bytes und dem gespeicherten Inhaltstyp

#### Scenario: Fremder Spieler ruft einen Nachweis ab

- **WHEN** ein anderer Spieler denselben Endpoint aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Kein Nachweis vorhanden

- **WHEN** der Endpoint für einen Eintrag ohne Nachweis aufgerufen wird
- **THEN** antwortet das System mit HTTP 404

#### Scenario: Nachweise sind nicht über den Medien-Endpoint erreichbar

- **WHEN** ein Nutzer versucht, einen Trainingsnachweis über `GET /api/media/{id}` abzurufen
- **THEN** existiert dort kein entsprechender Datensatz
