# event-wizard Specification

## Purpose

Diese Spezifikation beschreibt die Capability `event-wizard`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Event-Typ wählen

Beim Anlegen eines Events SHALL der User zunächst den Typ wählen: Heimspiel, Auswärtsspiel oder Sonstiges Event. Der Typ bestimmt, welche Dienstplan-Vorlagen zur Auswahl stehen und welche Teams wählbar sind.

#### Scenario: Typ-Auswahl als erster Wizard-Schritt
- **WHEN** ein berechtigter User auf „Event anlegen" klickt
- **THEN** erscheint ein Dialog mit drei Typ-Optionen: Heimspiel, Auswärtsspiel, Sonstiges Event
- **THEN** kein weiterer Schritt ist sichtbar bevor ein Typ gewählt wurde

---

### Requirement: Detailfelder je Typ

Im zweiten Wizard-Schritt SHALL das Formular je nach gewähltem Typ unterschiedliche Felder zeigen.

#### Scenario: Felder für Heimspiel oder Auswärtsspiel
- **WHEN** Typ Heimspiel oder Auswärtsspiel gewählt ist
- **THEN** sind Datum, Uhrzeit, Gegner (Freitext) und Mannschaft (Single-Select) als Pflichtfelder sichtbar

#### Scenario: Felder für Sonstiges Event
- **WHEN** Typ Sonstiges Event gewählt ist
- **THEN** sind Datum, Uhrzeit, Eventname (Freitext) und Mannschaft(en) (Multi-Select, alle Teams) als Pflichtfelder sichtbar

---

### Requirement: Mannschafts-Scoping nach Rolle und Typ

Die wählbaren Mannschaften im Wizard SHALL von Rolle und Event-Typ abhängen. Die Datenquelle
des Pickers macht die Regel prüfbar: Für Heim-/Auswärtsspiele speist sich die Auswahl aus der
nutzergefilterten Liste `GET /api/teams`, für Sonstige Events aus der vereinsweiten Liste
`GET /api/teams/names` (alle aktiven Mannschaften der aktiven Saison).

#### Scenario: Trainer bei Heimspiel oder Auswärtsspiel
- **WHEN** ein Trainer Typ Heimspiel oder Auswärtsspiel wählt
- **THEN** enthält das Mannschafts-Dropdown nur die eigenen Mannschaften des Trainers (Quelle: `GET /api/teams`)

#### Scenario: Trainer bei Sonstigem Event
- **WHEN** ein Trainer Typ Sonstiges Event wählt
- **THEN** sind alle aktiven Mannschaften im Multi-Select wählbar (Quelle: `GET /api/teams/names`, kein Rollenfilter)
- **THEN** wird jede Mannschaft mit ihrem berechneten Kurznamen aus `buildTeamShortNames` beschriftet

#### Scenario: Admin oder Vorstand
- **WHEN** ein User mit Rolle admin oder vorstand ein Event anlegt
- **THEN** sind alle aktiven Mannschaften wählbar, unabhängig vom Typ

---

### Requirement: Explizite Vorlagenauswahl

Im dritten Wizard-Schritt SHALL der User eine Dienstplan-Vorlage explizit auswählen.

#### Scenario: Vorlagen gefiltert nach Typ
- **WHEN** der User Schritt 3 erreicht
- **THEN** werden nur Vorlagen angezeigt, deren `template_type` zum gewählten Event-Typ passt (heim/auswärts/generisch)

#### Scenario: Keine passende Vorlage vorhanden
- **WHEN** es keine Vorlage für den gewählten Typ gibt
- **THEN** erscheint ein Hinweis „Keine passende Vorlage — Event wird ohne Dienste angelegt"
- **THEN** der User kann dennoch fortfahren (Event ohne Slots)

---

### Requirement: Dienste bestätigen

Im vierten Wizard-Schritt SHALL der User die vorgenerierten Dienste bestätigen oder einzelne abwählen.

#### Scenario: Slot-Preview anzeigen
- **WHEN** eine Vorlage gewählt wurde
- **THEN** zeigt Schritt 4 die aus der Vorlage berechneten Slots (Zeit, Diensttyp, Anzahl, Rolle)
- **THEN** der User kann einzelne Slots per Checkbox abwählen

#### Scenario: Erstellen ohne Dienste
- **WHEN** der User „Ohne Dienste" wählt oder alle Slots abgewählt hat
- **THEN** wird das Event ohne Duty-Slots angelegt

#### Scenario: Slots pro Team
- **WHEN** mehrere Mannschaften gewählt wurden (Sonstiges Event)
- **THEN** wird für jede Mannschaft ein identischer Satz Slots angelegt

---

### Requirement: Backend-Validierung Trainer-Scope

Das Backend SHALL die übergebenen `team_ids` für reine Trainer (Vereinsfunktion `trainer` ohne
`sportliche_leitung`/`vorstand`, System-Rolle nicht `admin`) typabhängig prüfen:

- Bei `heim`/`auswärts` MÜSSEN **alle** `team_ids` zu den eigenen Mannschaften des Trainers
  gehören (via `kader_trainers` in der aktiven Saison).
- Bei `generisch` MUSS **mindestens eine** `team_id` zu den eigenen Mannschaften gehören; weitere
  `team_ids` dürfen beliebige aktive Mannschaften sein.

Die vollständigen Regeln inklusive `PUT`/`DELETE` stehen in der Capability
`game-mutation-team-scope`.

#### Scenario: Trainer übergibt fremde Mannschaft bei Heimspiel
- **WHEN** ein Trainer `event_type='heim'` mit `team_ids` übergibt, die nicht in seinen
  `kader_trainers`-Einträgen liegen
- **THEN** antwortet der Server mit HTTP 403 Forbidden

#### Scenario: Trainer übergibt fremde Mannschaften bei Sonstigem Event
- **WHEN** ein Trainer `event_type='generisch'` mit `team_ids` übergibt, die mindestens eine
  eigene und daneben fremde Mannschaften enthalten
- **THEN** legt der Server das Event mit allen übergebenen Mannschaften an (HTTP 201)

#### Scenario: Trainer übergibt ausschließlich fremde Mannschaften bei Sonstigem Event
- **WHEN** ein Trainer `event_type='generisch'` ausschließlich mit fremden `team_ids` übergibt
- **THEN** antwortet der Server mit HTTP 403 Forbidden
- **THEN** entsteht kein Event, das der Trainer selbst nicht sehen könnte

---

### Requirement: Optionales Enddatum für generische Events im Wizard

Im Event-Wizard SHALL für Events vom Typ „Sonstiges Event" (generisch) ein optionales Enddatum-Feld angezeigt werden. Es ermöglicht das Anlegen mehrtägiger Events (z.B. Trainingslager).

#### Scenario: Enddatum-Feld bei generischem Event sichtbar
- **WHEN** der User im Wizard Typ „Sonstiges Event" gewählt hat
- **THEN** erscheint im Detailformular ein optionales Feld „Enddatum" unterhalb des Startdatums

#### Scenario: Enddatum-Feld bei Heimspiel/Auswärtsspiel nicht sichtbar
- **WHEN** der User im Wizard Typ Heimspiel oder Auswärtsspiel gewählt hat
- **THEN** ist kein Enddatum-Feld sichtbar

#### Scenario: Enddatum leer lassen erzeugt eintägiges Event
- **WHEN** der User das Enddatum-Feld leer lässt und das Formular abschickt
- **THEN** wird das Event ohne `end_date` angelegt (eintägig)

#### Scenario: Enddatum vor Startdatum wird abgelehnt
- **WHEN** der User ein Enddatum eingibt, das vor dem Startdatum liegt
- **THEN** zeigt das Formular eine Validierungsfehlermeldung und verhindert das Absenden
