# uebungsgruppen-termin-filter Specification

## Purpose

Diese Capability lässt Nutzer die Terminlisten auf `/kalender` und `/termine` gezielt
auf eine oder mehrere Übungsgruppen einschränken, statt nur nach Mannschaften filtern
zu können.

## Requirements

### Requirement: Sichtbare Übungsgruppen für den eingeloggten Nutzer

`GET /api/practice-groups/my` SHALL die Übungsgruppen (`kader.kind='practice'`) der
aktiven Saison liefern, die für den anfragenden Nutzer sichtbar sind — als JSON-Array
`[{id, name}]`, sortiert nach Name. Die Route SHALL für jeden authentifizierten Nutzer
erreichbar sein (Authenticated-Tier), unabhängig von System-Rolle oder Vereinsfunktion.

Sichtbarkeit:
- admin, vorstand und sportliche_leitung SHALL alle Übungsgruppen der aktiven Saison sehen.
- Ein Trainer SHALL die Übungsgruppen sehen, in deren `kader_trainers` er eingetragen ist.
- Ein Spieler SHALL die Übungsgruppen sehen, in deren Stammkader (`kader_members`) oder
  erweitertem Kader (`kader_extended_members`) er steht.
- Ein Elternteil SHALL zusätzlich die Übungsgruppen seiner Kinder sehen (`family_links`
  auf `kader_members`/`kader_extended_members`).
- Gibt es keine aktive Saison, SHALL die Antwort ein leeres Array sein (kein Fehler) —
  Übungsgruppen sind ohne aktive Saison ohnehin nicht adressierbar.
- Ohne gültiges Auth-Token SHALL die Route mit 401 antworten.

#### Scenario: Trainer sieht seine eigene Übungsgruppe
- **WHEN** ein Trainer, der in `kader_trainers` einer Übungsgruppe „Torwarttraining"
  eingetragen ist, `GET /api/practice-groups/my` aufruft
- **THEN** enthält die Antwort „Torwarttraining"

#### Scenario: Spieler ohne Zugehörigkeit sieht eine fremde Übungsgruppe nicht
- **WHEN** ein Spieler, der weder Mitglied noch Trainer der Übungsgruppe „Torwarttraining"
  ist, `GET /api/practice-groups/my` aufruft
- **THEN** enthält die Antwort „Torwarttraining" nicht

#### Scenario: Vorstand sieht alle Übungsgruppen
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `GET /api/practice-groups/my` aufruft
- **THEN** enthält die Antwort alle Übungsgruppen der aktiven Saison, unabhängig von
  eigener Mitgliedschaft

#### Scenario: Ohne Auth-Token
- **WHEN** `GET /api/practice-groups/my` ohne gültiges Auth-Token aufgerufen wird
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Übungsgruppen als Filteroption auf /kalender und /termine

Der bestehende Mannschafts-Filter (Dropdown mit Checkboxen) auf `/kalender` und
`/termine` SHALL zusätzlich zu den Mannschaften auch die für den Nutzer sichtbaren
Übungsgruppen (`GET /api/practice-groups/my`) als wählbare Optionen anzeigen — in
derselben Liste, ohne separate Sektion oder zweites Dropdown.

Da Übungsgruppen-IDs (`kader.id`) und Mannschafts-IDs (`teams.id`) unabhängige,
potenziell überlappende ID-Räume sind, SHALL eine Übungsgruppe im Filter-Zustand
(Set der ausgewählten IDs, geteilt zwischen Mannschaften und Übungsgruppen) als
**negative** Zahl (`-kader_id`) geführt werden. Diese Kodierung ist ein reines
Implementierungsdetail des Frontend-Zustands und SHALL nicht in der URL sichtbar
anders behandelt werden als Mannschafts-IDs — der bestehende `team`-Query-Parameter
(kommaseparierte ID-Liste) nimmt negative Werte genauso auf wie positive.

Ein Trainingstermin, der einer Übungsgruppe gehört (Server-Projektion `team_id=0`),
SHALL beim Filtern über seine `kader_id` und die daraus abgeleitete negative Filter-ID
gematcht werden — unabhängig davon, wie viele Übungsgruppen es gibt, matcht dabei stets
nur die eine tatsächlich zugehörige Gruppe, nicht alle Übungsgruppen gemeinsam.

Spiele SHALL von einer Übungsgruppen-Auswahl unberührt bleiben in dem Sinn, dass ihre
bestehende Team-Zuordnung (`team_ids`) unverändert geprüft wird — da Spiele strukturell
nie einer Übungsgruppe gehören, blendet eine reine Übungsgruppen-Auswahl alle Spiele aus.

#### Scenario: Übungsgruppe erscheint im Filter-Dropdown
- **WHEN** ein Nutzer, der Mitglied der Übungsgruppe „Torwarttraining" ist, `/kalender`
  oder `/termine` öffnet
- **THEN** enthält das Mannschafts-Filter-Dropdown einen Eintrag „Torwarttraining"
  zusätzlich zu den Mannschaften

#### Scenario: Filtern auf eine einzelne Übungsgruppe zeigt nur deren Termine
- **WHEN** ein Nutzer im Filter-Dropdown ausschließlich „Torwarttraining" anhakt
- **THEN** zeigt die Liste nur Trainingstermine der Übungsgruppe „Torwarttraining"
- **THEN** verschwinden alle Spiele und alle Trainingstermine anderer Mannschaften/Gruppen
  aus der Liste

#### Scenario: Zwei Übungsgruppen mit unterschiedlichen Terminen
- **WHEN** ein admin-Nutzer sowohl „Torwarttraining" als auch „Athletiktraining" anhakt
  (beides Übungsgruppen)
- **THEN** zeigt die Liste die Trainingstermine beider Gruppen
- **THEN** zeigt die Liste keine Trainingstermine anderer, nicht angehakter Übungsgruppen

#### Scenario: Kombination aus Mannschaft und Übungsgruppe
- **WHEN** ein Nutzer sowohl eine Mannschaft als auch eine Übungsgruppe anhakt
- **THEN** zeigt die Liste die Termine der gewählten Mannschaft UND die
  Trainingstermine der gewählten Übungsgruppe
