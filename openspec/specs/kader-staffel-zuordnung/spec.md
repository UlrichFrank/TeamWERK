# kader-staffel-zuordnung Specification

## Purpose
Zuordnung eines BWHV-Staffelcodes zu einem Kader. Da ein Kader saison-gebunden
ist, gilt die Zuordnung je Saison ohne zusätzliche Verknüpfung.

Gespeichert wird der Staffelcode, nicht die Handball4All-Klassen-ID: die
wechselt mit der Saison, der Code nicht.

## Requirements

### Requirement: Staffelcode je Kader und Saison

Das System SHALL an jedem Kader einen optionalen Staffelcode führen. Der Code wird über
die bestehende Route `PUT /api/kader/{id}` gepflegt und folgt deren Antwortvertrag
(HTTP 204 bei Erfolg); er ist dort Tri-State — fehlendes Feld bedeutet unverändert, ein
leerer Wert entfernt die Zuordnung. Da ein Kader
saison-gebunden ist, gilt die Zuordnung je Saison ohne zusätzliche Verknüpfung.

Das System SHALL den Staffelcode ausschließlich an Kadern der Art `team` zulassen und
einen Schreibversuch an einer Übungsgruppe (`kind='practice'`) mit HTTP 409 ablehnen.

#### Scenario: Staffel wird an einem Mannschafts-Kader gesetzt
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` an einem Kader der Art `team` den Staffelcode `mB-RL-BW` setzt
- **THEN** antwortet der Server mit HTTP 204 und der Code ist gespeichert

#### Scenario: Übungsgruppe nimmt keinen Staffelcode
- **WHEN** ein Staffelcode an einem Kader mit `kind='practice'` gesetzt wird
- **THEN** antwortet der Server mit HTTP 409

#### Scenario: Staffel wird entfernt
- **WHEN** der Staffelcode eines Kaders geleert wird
- **THEN** wird für diesen Kader kein Abruf mehr durchgeführt

### Requirement: Validierung des Staffelcodes gegen den Kader

Das System SHALL beim Speichern prüfen, ob Geschlecht und Altersklasse, die im Staffelcode
kodiert sind, mit `gender` und `age_class` des Kaders übereinstimmen, und einen
unpassenden Code mit HTTP 400 ablehnen.

Das System SHALL einen Code, dessen Aufbau nicht interpretierbar ist, ebenfalls mit
HTTP 400 ablehnen, statt ihn ungeprüft zu übernehmen.

#### Scenario: Unpassendes Geschlecht wird abgelehnt
- **WHEN** der Code `mB-RL-BW` an einem Kader mit `gender='f'` gesetzt wird
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Unpassende Altersklasse wird abgelehnt
- **WHEN** der Code `mB-RL-BW` an einem Kader mit `age_class='C-Jugend'` gesetzt wird
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Nicht interpretierbarer Code wird abgelehnt
- **WHEN** ein Code gesetzt wird, dessen Aufbau kein Geschlecht und keine Altersklasse erkennen lässt
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Passender Code wird angenommen
- **WHEN** der Code `mB-RL-BW` an einem Kader mit `gender='m'` und `age_class='B-Jugend'` gesetzt wird
- **THEN** antwortet der Server mit HTTP 204

### Requirement: Auswahl aus dem Live-Katalog mit Freitext-Alternative

Das System SHALL eine Route bereitstellen, die den Staffel-Katalog des Verbands und der
Bezirke für die aktive Periode liefert, beschränkt auf Nutzer mit Vereinsfunktion
`vorstand` oder Systemrolle `admin`.

Das System SHALL in der Kader-Verwaltung eine Auswahl aus diesem Katalog anbieten und
zusätzlich die direkte Eingabe eines Staffelcodes als Freitext erlauben. Beide Wege
SHALL derselben Validierung unterliegen.

#### Scenario: Vorstand ruft den Katalog ab
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` den Staffel-Katalog abruft
- **THEN** antwortet der Server mit HTTP 200 und den Staffeln des Verbands und der Bezirke

#### Scenario: Standard-Nutzer darf den Katalog nicht abrufen
- **WHEN** ein Nutzer ohne Vereinsfunktion `vorstand` und ohne Systemrolle `admin` den Katalog abruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Freitext unterliegt derselben Prüfung
- **WHEN** ein Staffelcode als Freitext eingegeben wird, der nicht zum Kader passt
- **THEN** antwortet der Server mit HTTP 400, wie bei der Auswahl aus dem Katalog

#### Scenario: Katalog nicht erreichbar blockiert die Pflege nicht
- **WHEN** der Katalog-Abruf fehlschlägt
- **THEN** bleibt die Freitext-Eingabe verfügbar
