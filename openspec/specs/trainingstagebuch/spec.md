# trainingstagebuch Specification

## Purpose

Erfassung selbst absolvierter Trainingseinheiten durch Spieler (Eigentraining, getrennt von
`internal/trainings`): Datenmodell, Pflichtfelder, Artenkatalog mit Freitext-Zweig, RPE-Skala
1–10 sowie CRUD auf den eigenen Einträgen inklusive Live-Update via SSE.

## Requirements

### Requirement: Spieler erfassen eigene Trainingseinheiten

Das System SHALL jedem eingeloggten Nutzer, dem ein Mitglieds-Datensatz zugeordnet ist, erlauben,
über `POST /api/training-diary` eine selbst absolvierte Trainingseinheit zu erfassen. Der Eintrag
wird stets dem Mitglied des Aufrufers zugeordnet; eine vom Client übergebene `member_id` SHALL
ignoriert werden.

Pflichtfelder sind `trained_on` (Datum), `kind` (Art), `duration_min` (Dauer in Minuten) und `rpe`
(Intensität). `note` ist optional.

Besitzt der aufrufende Nutzer keinen Mitglieds-Datensatz, antwortet das System mit HTTP 403.

#### Scenario: Spieler erfasst eine Einheit

- **WHEN** ein Spieler `POST /api/training-diary` mit `trained_on`, `kind='kraft'`,
  `duration_min=45` und `rpe=7` aufruft
- **THEN** antwortet das System mit HTTP 201 und dem angelegten Eintrag
- **THEN** trägt der Eintrag die `member_id` des Aufrufers

#### Scenario: Client versucht, den Eintrag einem anderen Mitglied zuzuordnen

- **WHEN** ein Spieler `POST /api/training-diary` mit einer fremden `member_id` im Body aufruft
- **THEN** wird der Eintrag dem Mitglied des Aufrufers zugeordnet, nicht dem übergebenen

#### Scenario: Nutzer ohne Mitglieds-Datensatz

- **WHEN** ein eingeloggter Nutzer ohne verknüpftes Mitglied `POST /api/training-diary` aufruft
- **THEN** antwortet das System mit HTTP 403
- **THEN** wurde kein Eintrag angelegt

---

### Requirement: Trainingsart aus fester Liste oder als Freitext

Das System SHALL `kind` auf genau diese Werte beschränken: `kraft`, `ausdauer`, `athletik`,
`technik`, `beweglichkeit`, `reha`, `sonstiges`.

Bei `kind = 'sonstiges'` MUSS `kind_custom` einen nicht-leeren Text (max. 60 Zeichen) enthalten.
Bei jedem anderen `kind` MUSS `kind_custom` leer bleiben; ein dennoch übergebener Wert SHALL
verworfen werden.

Verletzt der Request diese Bedingungen, antwortet das System mit HTTP 400 und legt keinen Eintrag
an.

#### Scenario: Freitext-Art mit Bezeichnung

- **WHEN** ein Spieler einen Eintrag mit `kind='sonstiges'` und `kind_custom='Schwimmen'` anlegt
- **THEN** antwortet das System mit HTTP 201
- **THEN** liefert die Liste diesen Eintrag mit der Bezeichnung `Schwimmen`

#### Scenario: Freitext-Art ohne Bezeichnung

- **WHEN** ein Spieler einen Eintrag mit `kind='sonstiges'` und leerem `kind_custom` anlegt
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Unbekannte Art

- **WHEN** ein Spieler einen Eintrag mit `kind='yoga'` anlegt
- **THEN** antwortet das System mit HTTP 400

---

### Requirement: Intensität als RPE-Skala von 1 bis 10

Das System SHALL `rpe` als ganze Zahl zwischen 1 und 10 (jeweils einschließlich) annehmen und
Werte außerhalb dieses Bereichs mit HTTP 400 ablehnen.

Das Frontend SHALL neben der Auswahl eine standardmäßig **eingeklappte** Erklärung der Skala
anbieten, die per Klick aufgeht und die Stufen in Alltagssprache beschreibt (1–2 sehr leicht,
3–4 leicht, 5–6 mittel, 7–8 hart, 9–10 maximal) samt dem Hinweis, dass eine Schätzung genügt.

#### Scenario: Gültige Intensität

- **WHEN** ein Spieler einen Eintrag mit `rpe=1` oder `rpe=10` anlegt
- **THEN** antwortet das System mit HTTP 201

#### Scenario: Ungültige Intensität

- **WHEN** ein Spieler einen Eintrag mit `rpe=0` oder `rpe=11` anlegt
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Erklärung der Skala ist erreichbar

- **WHEN** das Erfassungsformular geöffnet wird
- **THEN** ist die Erklärung der RPE-Skala eingeklappt sichtbar
- **THEN** zeigt ein Klick darauf die Beschreibung aller Stufen

---

### Requirement: Dauer und Datum werden plausibilisiert

Das System SHALL `duration_min` als ganze Zahl größer 0 und höchstens 600 annehmen und
`trained_on` nur mit einem Datum akzeptieren, das **nicht in der Zukunft** liegt. Verletzungen
werden mit HTTP 400 beantwortet.

Mehrere Einträge am selben Tag SHALL zulässig sein.

#### Scenario: Training in der Zukunft

- **WHEN** ein Spieler einen Eintrag mit `trained_on` = morgen anlegt
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Unplausible Dauer

- **WHEN** ein Spieler einen Eintrag mit `duration_min=0` oder `duration_min=1000` anlegt
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Zwei Einheiten an einem Tag

- **WHEN** ein Spieler zwei Einträge mit demselben `trained_on` anlegt
- **THEN** antwortet das System beide Male mit HTTP 201
- **THEN** enthält die Liste beide Einträge

---

### Requirement: Spieler verwalten ihre eigenen Einträge

Das System SHALL über `GET /api/training-diary` die eigenen Einträge des Aufrufers liefern,
absteigend nach `trained_on` sortiert, optional gefiltert über `?season=<id>`. Über
`PUT /api/training-diary/{id}` und `DELETE /api/training-diary/{id}` SHALL der Eigentümer seine
Einträge ändern und löschen können.

Ändert oder löscht ein anderer Nutzer als der Eigentümer, antwortet das System mit HTTP 403 —
**auch dann, wenn dieser Nutzer den Eintrag lesen darf** (Trainer, Elternteil, sportliche Leitung,
admin). Schreibrechte hat ausschließlich der Eigentümer.

Beim Löschen eines Eintrags SHALL eine zugehörige Nachweisdatei mit entfernt werden.

#### Scenario: Eigene Liste enthält nur eigene Einträge

- **WHEN** ein Spieler `GET /api/training-diary` aufruft
- **THEN** antwortet das System mit HTTP 200
- **THEN** enthält die Antwort ausschließlich Einträge seines eigenen Mitglieds

#### Scenario: Eigenen Eintrag ändern

- **WHEN** der Eigentümer `PUT /api/training-diary/{id}` mit geänderter Dauer aufruft
- **THEN** antwortet das System mit HTTP 200 und der Eintrag trägt den neuen Wert

#### Scenario: Trainer versucht, einen Eintrag zu ändern

- **WHEN** ein Trainer des Kaders `PUT /api/training-diary/{id}` auf den Eintrag eines seiner
  Spieler aufruft
- **THEN** antwortet das System mit HTTP 403
- **THEN** bleibt der Eintrag unverändert

#### Scenario: Eintrag mit Nachweis löschen

- **WHEN** der Eigentümer `DELETE /api/training-diary/{id}` auf einen Eintrag mit Nachweis aufruft
- **THEN** antwortet das System mit HTTP 204
- **THEN** ist die Nachweisdatei aus dem Speicherverzeichnis entfernt

---

### Requirement: Mutationen lösen ein Live-Update aus

Jede Mutations-Route des Trainingstagebuchs (`POST`/`PUT`/`DELETE` auf Einträgen und Nachweisen)
SHALL `h.hub.Broadcast("training-diary-changed")` aufrufen. Das Ereignis SHALL **keine Nutzlast**
tragen; die Clients laden nach Empfang neu und erhalten dabei ausschließlich die für sie
freigegebenen Daten.

Die Tagebuch-Seiten SHALL dieses Ereignis über `useLiveUpdates` abonnieren.

#### Scenario: Anlegen aktualisiert offene Sichten

- **WHEN** ein Spieler einen Eintrag anlegt
- **THEN** wird `training-diary-changed` über den Event-Hub gesendet
- **THEN** lädt eine offene Trainer-Übersicht ihre Daten neu

#### Scenario: Ereignis transportiert keine Daten

- **WHEN** ein Client das Ereignis empfängt
- **THEN** besteht es ausschließlich aus der Zeichenkette `training-diary-changed`
