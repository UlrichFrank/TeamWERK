# spieler-saisonstatistik Specification

## Purpose
Identitätsauflösung der Spieler aus den ausgewerteten Spielberichten und die
daraus gebildete Saisonbilanz: Kennzahlen im Spieler-Profil sowie Mannschafts-
und Staffel-Ranglisten.

Die Identität folgt dem Namen (mit Levenshtein-Toleranz); die Trikotnummer
bestätigt und bricht Gleichstand, ist aber nicht der Schlüssel.

## Requirements

### Requirement: Identität eines Spielers über die Saison

Das System SHALL einen Spieler innerhalb einer Staffel und Mannschaft über seinen Namen
identifizieren und dabei Schreibvarianten mit einer Levenshtein-Toleranz zusammenführen.

Das System SHALL die Trikotnummer als Bestätigung verwenden und bei mehreren Spielern
gleichen Namens innerhalb derselben Mannschaft als unterscheidendes Merkmal heranziehen.

Das System SHALL die Trikotnummer NICHT als führendes Identitätsmerkmal verwenden, damit
eine abweichende Nummer (etwa bei einem geliehenen Trikot) keine zweite Person erzeugt.

Das System SHALL einen Widerspruch zwischen Name und Nummer am Spieler festhalten und
sichtbar machen, statt ihn stillschweigend aufzulösen.

Das System SHALL Platzhalter-Einträge ohne echten Namen nicht als Spieler führen.

#### Scenario: Schreibvariante ist dieselbe Person
- **WHEN** derselbe Spieler in zwei Berichten mit Namen erfasst wird, die sich um höchstens zwei Zeichen unterscheiden
- **THEN** existiert genau ein Spieler-Datensatz für ihn

#### Scenario: Trikotwechsel zerreißt die Person nicht
- **WHEN** ein Spieler in zwei Berichten denselben Namen, aber verschiedene Trikotnummern trägt
- **THEN** existiert genau ein Spieler-Datensatz und der Widerspruch ist am Spieler vermerkt

#### Scenario: Gleiche Namen werden über die Nummer getrennt
- **WHEN** zwei Spieler derselben Mannschaft denselben Namen, aber verschiedene Trikotnummern tragen
- **THEN** existieren zwei Spieler-Datensätze

#### Scenario: Platzhalter erzeugt keinen Spieler
- **WHEN** eine Berichtszeile einen Platzhalter ohne Person als Namen trägt
- **THEN** entsteht kein Spieler-Datensatz

### Requirement: Zuordnung eigener Spieler zu Mitgliedern

Das System SHALL Spieler der eigenen Mannschaften zusätzlich auf einen Mitglieds-Datensatz
auflösen und dabei Name, Geburtsjahrgang, Trikotnummer und Kaderzugehörigkeit der Saison
heranziehen.

Das System SHALL einen nicht eindeutig auflösbaren Spieler als nicht zugeordnet
kennzeichnen und eine manuelle Zuordnung ermöglichen, statt eine Zuordnung zu raten.

Das System SHALL eine einmal vorgenommene manuelle Zuordnung für die gesamte Saison
beibehalten.

#### Scenario: Eindeutiger Spieler wird zugeordnet
- **WHEN** ein Spieler der eigenen Mannschaft in Name, Jahrgang und Kaderzugehörigkeit genau einem Mitglied entspricht
- **THEN** wird er diesem Mitglied zugeordnet

#### Scenario: Mehrdeutiger Spieler bleibt offen
- **WHEN** ein Spieler der eigenen Mannschaft auf mehrere Mitglieder passt
- **THEN** bleibt er als nicht zugeordnet gekennzeichnet und wird zur manuellen Zuordnung angeboten

#### Scenario: Manuelle Zuordnung hält
- **WHEN** ein zuvor offener Spieler manuell einem Mitglied zugeordnet wurde
- **THEN** wird er in weiteren Berichten derselben Saison diesem Mitglied zugerechnet

### Requirement: Saisonbilanz im Spieler-Profil

Das System SHALL zu einem Mitglied die Saisonbilanz aus den ausgewerteten Spielberichten
bereitstellen: Zahl der Spiele, Tore, Siebenmeter-Versuche und -Treffer, Zeitstrafen und
Karten sowie den Verlauf über die Saison.

Das System SHALL Berichte im Zustand `parse_failed` NICHT in die Bilanz einbeziehen.

#### Scenario: Bilanz eines Mitglieds
- **WHEN** die Saisonbilanz eines Mitglieds abgerufen wird, dem Spieler aus mehreren Berichten zugeordnet sind
- **THEN** antwortet der Server mit HTTP 200 und den summierten Werten der Saison

#### Scenario: Unbekanntes Mitglied
- **WHEN** die Saisonbilanz zu einer nicht existierenden Mitglieds-Kennung abgerufen wird
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Mitglied ohne ausgewertete Spiele
- **WHEN** einem Mitglied kein Spieler aus einem ausgewerteten Bericht zugeordnet ist
- **THEN** antwortet der Server mit HTTP 200 und einer leeren Bilanz

#### Scenario: Fehlgeschlagener Bericht zählt nicht
- **WHEN** ein Bericht im Zustand `parse_failed` vorliegt
- **THEN** fließt er in keine Saisonbilanz ein

### Requirement: Ranglisten je Staffel

Das System SHALL je Staffel Ranglisten bereitstellen — mindestens Torschützen,
Siebenmeter-Quote und eine Fair-Play-Wertung aus Zeitstrafen und Karten — und diese über
alle ausgewerteten Berichte der Saison bilden.

Das System SHALL eigene und fremde Spieler gleichermaßen in die Ranglisten aufnehmen.

Das System SHALL die Ranglisten allen eingeloggten Nutzern zugänglich machen.

#### Scenario: Torschützenliste
- **WHEN** ein eingeloggter Nutzer die Ranglisten einer Staffel abruft
- **THEN** antwortet der Server mit HTTP 200 und einer nach Toren absteigend sortierten Liste

#### Scenario: Fremde Spieler sind enthalten
- **WHEN** eine Rangliste gebildet wird
- **THEN** enthält sie auch Spieler von Mannschaften anderer Vereine

#### Scenario: Nicht eingeloggter Zugriff
- **WHEN** Ranglisten ohne gültiges Zugangstoken abgerufen werden
- **THEN** antwortet der Server mit HTTP 401
