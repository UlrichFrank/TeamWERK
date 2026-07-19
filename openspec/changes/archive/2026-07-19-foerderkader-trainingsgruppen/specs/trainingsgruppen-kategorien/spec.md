## ADDED Requirements

### Requirement: Getrennte Referenzliste der Trainingsgruppen-Kategorien
Das System SHALL nicht-spielgebundene Kader-Kategorien (z. B. „Förderkader",
„Perspektivkader") in einer eigenen Tabelle `training_group_categories` führen,
**getrennt** von `age_class_game_rules`. Diese Kategorien SHALL **keine**
Spielregeln (Halbzeit-/Pausendauer) tragen und den CHECK-Constraint der
Spiel-Altersklassen NICHT berühren. Die Liste dient als Eingabe-Unterstützung für
die Kader-Anlage; `kader.age_class`/`teams.age_class` bleiben Freitext ohne
Fremdschlüssel auf diese Tabelle.

#### Scenario: Kategorienliste lesen
- **WHEN** ein authentifizierter Nutzer `GET /api/training-group-categories`
  aufruft
- **THEN** liefert die API die gepflegten Kategorien (Seed enthält mindestens
  „Förderkader" und „Perspektivkader") mit HTTP 200

#### Scenario: Spiel-Altersklassen bleiben unberührt
- **WHEN** eine Trainingsgruppen-Kategorie angelegt wird
- **THEN** enthält `age_class_game_rules` keinen zusätzlichen Eintrag und dessen
  CHECK-Constraint bleibt auf `A/B/C/D-Jugend` beschränkt

### Requirement: Verwaltung der Trainingsgruppen-Kategorien durch den Vorstand
Das System SHALL es dem Vorstand (System-Rolle `admin` immer) erlauben, eine
Trainingsgruppen-Kategorie anzulegen (`POST /api/training-group-categories`) und zu
löschen (`DELETE /api/training-group-categories/{name}`). Jede Mutation SHALL einen
Broadcast auslösen (`training-group-categories-changed`), das Frontend abonniert ihn
via `useLiveUpdates`. Das Löschen einer noch von Kadern verwendeten Kategorie SHALL
zulässig sein; betroffene Kader behalten ihren `age_class`-Text (kein Kaskaden- oder
Sperrverhalten), die Kategorie verschwindet nur aus dem Anlage-Dropdown.

#### Scenario: Vorstand legt eine Kategorie an
- **WHEN** der Vorstand `POST /api/training-group-categories` mit einem neuen Namen
  sendet
- **THEN** antwortet das System mit HTTP 200/201, der Eintrag erscheint in der Liste
  und ein `training-group-categories-changed`-Broadcast wird gesendet

#### Scenario: Nicht-Vorstand darf nicht verwalten
- **WHEN** ein Nutzer ohne Vorstand-Funktion (und ohne `admin`)
  `POST`/`DELETE /api/training-group-categories` aufruft
- **THEN** antwortet das System mit HTTP 403 und die Liste bleibt unverändert

#### Scenario: Verwendete Kategorie löschen
- **WHEN** eine Kategorie gelöscht wird, die ein bestehender Kader als `age_class`
  nutzt
- **THEN** wird die Kategorie entfernt (HTTP 200), der Kader bleibt mit unverändertem
  `age_class` erhalten und weiterhin nutzbar

### Requirement: Kader-Anlage unioniert beide Kategorienquellen mit freier Jahrgangswahl
Das System SHALL in der Kader-Anlage-Maske sowohl die Spiel-Altersklassen (aus
`age_class_game_rules`) als auch die Trainingsgruppen-Kategorien (aus
`training_group_categories`) zur Auswahl anbieten. Für eine gewählte
Spiel-Altersklasse SHALL die Jahrgangsauswahl auf die berechneten Alter-Brackets
(`ComputeAgeBrackets`) beschränkt bleiben; für eine gewählte
Trainingsgruppen-Kategorie SHALL der Jahrgang **frei** wählbar sein (unabhängig von
Spiel-Brackets). Der Backend-Anlagepfad (`POST /api/kader`) bleibt unverändert und
akzeptiert `age_class` als Freitext sowie einen beliebigen `dedicated_birth_year`.
Bei mehreren Kadern derselben Trainingsgruppen-Kategorie und `gender` in einer
Saison SHALL die `team_number` nach **aufsteigendem `dedicated_birth_year`**
vergeben werden, damit die laufende Nummer im Kurznamen deterministisch dem
Jahrgang folgt.

#### Scenario: Förderkader mit freiem Jahrgang anlegen
- **WHEN** der Vorstand in der Anlage-Maske die Kategorie „Förderkader", ein
  Geschlecht und den Jahrgang 2016 wählt und speichert
- **THEN** wird ein Kader mit `age_class = 'Förderkader'` und
  `dedicated_birth_year = 2016` angelegt (HTTP 200/201)

#### Scenario: Spiel-Altersklasse behält Bracket-gebundene Jahrgänge
- **WHEN** in der Anlage-Maske eine Spiel-Altersklasse (z. B. „D-Jugend") gewählt
  wird
- **THEN** bietet die Jahrgangsauswahl ausschließlich die für diese Altersklasse und
  Saison berechneten Brackets an (unverändertes Verhalten)

#### Scenario: Kurzname folgt der bestehenden Formel ohne A–D zu verändern
- **WHEN** die Kader „Perspektivkader" (gemischt, Jahrgang 2015), „Förderkader"
  (gemischt, 2016) und „Förderkader" (gemischt, 2017) in einer Saison bestehen
- **THEN** liefert der kanonische `display_short` `gP`, `gF1` und `gF2` (Nummer nach
  aufsteigendem Jahrgang), während die Kurznamen der A–D-Jugend-Teams unverändert
  bleiben (die geteilte Kurzname-Formel wird nicht geändert)

#### Scenario: Jahrgang bleibt über den Badge sichtbar
- **WHEN** ein Trainingsgruppen-Kader mit `dedicated_birth_year` angezeigt wird
- **THEN** erscheint der Jahrgang als Badge in der Kader-Ansicht (nicht im
  Kurznamen), sodass „gF1 = 2016" für die aktuelle Saison eindeutig erkennbar ist

#### Scenario: Trainingsgruppen-Kader hat standardmäßig keine Spiele
- **WHEN** ein Kader für eine Trainingsgruppen-Kategorie angelegt wird
- **THEN** hat er `games_per_season = 0`, es werden keine Spiele oder Dienst-Slots
  automatisch erzeugt, und Trainings/RSVP funktionieren über die abgeleitete
  `team_id` unverändert

### Requirement: Kanonische Sortierreihenfolge der Kader-/Team-Listen
Das System SHALL Kader- und Team-Listen nach einer festen Kategorie-Ordnung
sortieren: zuerst alle Nicht-Trainingsgruppen (`*-Jugend`) alphabetisch, danach die
Trainingsgruppen-Kategorien nach ihrem `training_group_categories.sort_order`;
sekundär nach `gender` und `team_number`. Die Reihenfolge der Trainingsgruppen SHALL
ausschließlich über `sort_order` bestimmt werden (Single Source of Truth), nicht
über den alphabetischen Namen. Die Sortierung der A–D-Jugend-Klassen SHALL dadurch
unverändert bleiben.

#### Scenario: Trainingsgruppen erscheinen nach A–D in definierter Ordnung
- **WHEN** Kader der A–D-Jugend sowie „Perspektivkader" (`sort_order=1`) und
  „Förderkader" (`sort_order=2`) in einer Saison bestehen
- **THEN** liefert die Kaderliste die Reihenfolge A-Jugend, B-Jugend, C-Jugend,
  D-Jugend, Perspektivkader, Förderkader (nicht alphabetisch, das „Förderkader" vor
  „Perspektivkader" stellen würde)

#### Scenario: Sekundärsortierung innerhalb einer Kategorie
- **WHEN** zwei Förderkader-Kader derselben `gender` mit `team_number` 1 (Jahrgang
  2016) und 2 (Jahrgang 2017) bestehen
- **THEN** erscheint der Kader mit `team_number=1` (2016) vor dem mit
  `team_number=2` (2017)

#### Scenario: A–D-Sortierung bleibt unverändert
- **WHEN** ausschließlich A–D-Jugend-Kader vorhanden sind
- **THEN** ist ihre Reihenfolge identisch zum bisherigen Verhalten (alphabetisch nach
  `age_class`, dann `gender`, `team_number`)
