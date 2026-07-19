## ADDED Requirements

### Requirement: Förderkind als gültiger Member-Status
Das System SHALL `foerderkind` als validen Wert für `members.status` akzeptieren.
Ein Förderkind ist ein jahrgangsbezogenes Talent-/Trainingskind (häufig Gastkind
aus dem Stammverein), das an Trainings einer Trainingsgruppe teilnimmt, aber
**kein beitragspflichtiges Vereinsmitglied** ist und keinen eigenen System-Account
benötigt. Ein Kind, das **zusätzlich** reguläres Mitglied ist, behält
`status='aktiv'` und wird nicht als Förderkind geführt.

#### Scenario: Vorstand legt ein Förderkind an
- **WHEN** der Vorstand ein neues Mitglied mit `status = 'foerderkind'` anlegt (nur
  Vorname, Nachname und Geburtsdatum angegeben, ohne `join_date`)
- **THEN** wird der Datensatz mit `status = 'foerderkind'` gespeichert (HTTP
  200/201) und ist in der Mitgliederliste sichtbar

#### Scenario: Kein join_date-Zwang für Förderkinder
- **WHEN** ein Mitglied mit `status = 'foerderkind'` **ohne** `join_date` angelegt
  oder gespeichert wird
- **THEN** akzeptiert die API den Datensatz (kein HTTP 400 wegen fehlendem
  Eintrittsdatum), analog zum Verhalten bei `anwaerter`

#### Scenario: Ungültiger Status wird abgelehnt
- **WHEN** ein Request an die Mitglieder-Anlage/-Bearbeitung einen unbekannten
  Status-Wert sendet
- **THEN** antwortet die API mit HTTP 400

### Requirement: Förderkind über Kader und erweiterten Kader einbindbar
Das System SHALL es erlauben, ein Mitglied mit `status = 'foerderkind'` wie jedes
andere Mitglied einem Kader (`kader_members`) und/oder erweiterten Kader
(`kader_extended_members`) zuzuordnen. Die Zuordnung folgt denselben Regeln und
Endpunkten wie bei aktiven Mitgliedern; es gibt keinen Sonderpfad.

#### Scenario: Förderkind in einen Förderkader aufnehmen
- **WHEN** der Vorstand ein Förderkind über die Kader-Bearbeitung dem Kader
  „Förderkader 2016" hinzufügt
- **THEN** erscheint es in den `kader_members` dieses Kaders und ist über die vom
  Kader abgeleitete `team_id` für Trainings-RSVP dieses Teams berechtigt

#### Scenario: Förderkind gleichzeitig in mehreren Kadern
- **WHEN** ein Mitglied bereits in einem Kader eingetragen ist und zusätzlich einem
  Förderkader hinzugefügt wird
- **THEN** bleiben beide Zuordnungen bestehen (kein Konflikt, `kader_members`
  erlaubt Mitgliedschaft in mehreren Kadern)

### Requirement: Visueller Förderkind-Hinweis und Filter in der Mitgliederliste
Das System SHALL Mitglieder mit `status = 'foerderkind'` in der Mitgliederliste mit
einem erkennbaren Label/Badge kennzeichnen und als Filterkriterium anbieten, analog
zur Kennzeichnung von `anwaerter`.

#### Scenario: Badge in der Mitgliederliste
- **WHEN** ein Mitglied mit `status = 'foerderkind'` in der Liste angezeigt wird
- **THEN** zeigt die Zeile/Karte ein „Förderkind"-Label neben dem Namen

#### Scenario: Nach Förderkindern filtern
- **WHEN** in der Mitgliederliste der Status-Filter auf „Förderkind" gesetzt wird
- **THEN** enthält das Ergebnis ausschließlich Mitglieder mit
  `status = 'foerderkind'`
