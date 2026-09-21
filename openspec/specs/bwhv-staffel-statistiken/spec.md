# bwhv-staffel-statistiken Specification

## Purpose

Auswertungen, die aus den bereits gespeicherten Ergebnissen und Spielberichten einer
Staffel gerechnet werden: Kreuztabelle, Platzierungsverlauf über die Spieltage sowie
Mannschafts- und Schiedsrichter-Ranglisten. Alles ist Aggregat über vorhandene Daten —
kein zusätzlicher Abruf beim Verband und kein eigener Datenbestand.

## Requirements

### Requirement: Kreuztabelle einer Staffel

Das System SHALL je Staffel eine Matrix aller Mannschaften bereitstellen, in der zu jeder
Paarung aus Heim- und Gastmannschaft der Endstand der Begegnung steht.

Das System SHALL zu einer noch nicht gespielten Begegnung das angesetzte Datum statt eines
Ergebnisses ausweisen und beide Fälle unterscheidbar kennzeichnen.

Das System SHALL eine Begegnung genau dann als gespielt behandeln, wenn ihr Ergebnis
erfasst ist. Ein Spielstand von `0:0` SHALL als gespieltes Spiel gelten.

Das System SHALL die Diagonale (eine Mannschaft gegen sich selbst) als leer ausweisen.

Das System SHALL die Kreuztabelle allein aus den Ergebnissen der Begegnungen bilden und
KEINEN ausgewerteten Spielbericht voraussetzen.

Das System SHALL die Kreuztabelle allen eingeloggten Nutzern zugänglich machen.

#### Scenario: Gespielte Begegnung trägt den Endstand
- **WHEN** ein eingeloggter Nutzer die Kreuztabelle einer Staffel abruft, in der eine Begegnung mit Ergebnis vorliegt
- **THEN** antwortet der Server mit HTTP 200 und die Zelle dieser Paarung trägt den Endstand

#### Scenario: Torloses Spiel verschwindet nicht
- **WHEN** eine Begegnung mit dem Ergebnis `0:0` erfasst ist
- **THEN** gilt sie als gespielt und ihre Zelle trägt `0:0`, nicht das Datum

#### Scenario: Künftige Begegnung trägt das Datum
- **WHEN** eine Begegnung ohne erfasstes Ergebnis vorliegt
- **THEN** trägt ihre Zelle das angesetzte Datum und ist als noch nicht gespielt gekennzeichnet

#### Scenario: Ohne Spielbericht vollständig
- **WHEN** zu keiner Begegnung der Staffel ein ausgewerteter Spielbericht vorliegt
- **THEN** ist die Kreuztabelle dennoch vollständig mit allen Ergebnissen gefüllt

#### Scenario: Nicht eingeloggter Zugriff
- **WHEN** die Kreuztabelle ohne gültiges Zugangstoken abgerufen wird
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Platzierungsverlauf über die Spieltage

Das System SHALL je Staffel für jeden Spieltag, an dem mindestens eine Begegnung gespielt
wurde, die Platzierung jeder Mannschaft zu diesem Zeitpunkt bereitstellen.

Das System SHALL einen Spieltag über das Spieldatum bilden: alle an einem Datum gespielten
Begegnungen gehören zu demselben Spieltag.

Das System SHALL die Platzierung aus den bis zu diesem Spieltag kumulierten Ergebnissen
bilden und dabei in dieser Rangfolge sortieren: Punkte, dann Tordifferenz, dann
geworfene Tore.

Das System SHALL für die Punkte die im Spielbetrieb des Verbands übliche Wertung ansetzen:
Sieg zwei Punkte, Unentschieden ein Punkt, Niederlage kein Punkt.

Das System SHALL Begegnungen ohne erfasstes Ergebnis nicht in den Verlauf einbeziehen.

Das System SHALL den Verlauf allein aus den Ergebnissen bilden und KEINEN ausgewerteten
Spielbericht voraussetzen.

Das System SHALL den Verlauf allen eingeloggten Nutzern zugänglich machen.

#### Scenario: Verlauf über mehrere Spieltage
- **WHEN** ein eingeloggter Nutzer den Platzierungsverlauf einer Staffel mit Ergebnissen an drei verschiedenen Spieldaten abruft
- **THEN** antwortet der Server mit HTTP 200 und einer Folge von drei Spieltagen, jeder mit der Platzierung aller bis dahin beteiligten Mannschaften

#### Scenario: Punkte bestimmen die Platzierung
- **WHEN** zwei Mannschaften nach demselben Spieltag unterschiedlich viele Punkte haben
- **THEN** steht die Mannschaft mit mehr Punkten auf der besseren Platzierung

#### Scenario: Tordifferenz bricht den Punktgleichstand
- **WHEN** zwei Mannschaften nach einem Spieltag gleich viele Punkte, aber unterschiedliche Tordifferenz haben
- **THEN** steht die Mannschaft mit der besseren Tordifferenz auf der besseren Platzierung

#### Scenario: Staffel ohne Ergebnis
- **WHEN** in einer Staffel noch keine Begegnung gespielt wurde
- **THEN** antwortet der Server mit HTTP 200 und einem leeren Verlauf

### Requirement: Mannschafts-Ranglisten einer Staffel

Das System SHALL je Staffel Mannschafts-Ranglisten bereitstellen: Torverhältnis
(geworfene Tore, erhaltene Tore, Differenz), Angriff (Tore insgesamt und je Spiel),
Verteidigung (erhaltene Tore insgesamt und je Spiel), Fair-Play sowie die Verteilung der
Tore auf die Spieler.

Das System SHALL Torverhältnis, Angriff und Verteidigung allein aus den Ergebnissen der
Begegnungen bilden und KEINEN ausgewerteten Spielbericht voraussetzen.

Das System SHALL die Fair-Play-Wertung aus Zeitstrafen und Karten der ausgewerteten
Spielberichte bilden und die zugrunde liegende Gewichtung der einzelnen Strafarten
ausweisen, damit die Zahl nachvollziehbar bleibt.

Das System SHALL die Torverteilung einer Mannschaft als Durchschnitt und Median der Tore
je Spieler sowie als Maß der Ungleichverteilung ausweisen und dieses Maß so bilden, dass
ein Wert nahe Null eine gleichmäßige Verteilung und ein hoher Wert die Abhängigkeit von
wenigen Werfern bedeutet.

Das System SHALL zu jeder Mannschaft die Zahl der Spiele ausweisen, auf der ihre Werte
beruhen.

Das System SHALL alle Mannschaften der Staffel aufnehmen, auch die anderer Vereine.

Das System SHALL Berichte im Zustand `parse_failed` NICHT in die aus Spielberichten
gebildeten Werte einbeziehen.

Das System SHALL die Mannschafts-Ranglisten allen eingeloggten Nutzern zugänglich machen.

#### Scenario: Torverhältnis ohne Spielbericht
- **WHEN** ein eingeloggter Nutzer die Mannschafts-Ranglisten einer Staffel abruft, zu deren Begegnungen kein Spielbericht ausgewertet ist
- **THEN** antwortet der Server mit HTTP 200 und Torverhältnis, Angriff und Verteidigung sind gefüllt

#### Scenario: Fair-Play braucht Berichte
- **WHEN** zu einer Mannschaft kein ausgewerteter Spielbericht vorliegt
- **THEN** ist ihre Fair-Play-Wertung leer und wird nicht als straffreie Mannschaft ausgewiesen

#### Scenario: Fremde Mannschaften sind enthalten
- **WHEN** die Mannschafts-Ranglisten gebildet werden
- **THEN** enthalten sie auch die Mannschaften anderer Vereine

#### Scenario: Fehlgeschlagener Bericht zählt nicht
- **WHEN** ein Bericht im Zustand `parse_failed` vorliegt
- **THEN** fließen seine Werte in keine Mannschafts-Rangliste ein

#### Scenario: Nicht eingeloggter Zugriff
- **WHEN** die Mannschafts-Ranglisten ohne gültiges Zugangstoken abgerufen werden
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Hervorhebung der eigenen Zugehörigkeit

Das System SHALL zu einer Staffel bereitstellen, welche Mannschaften und welche Spieler
dem aufrufenden Nutzer zuzurechnen sind.

Das System SHALL als eigene Mannschaft jede Mannschaft ausweisen, zu der der Nutzer über
die Kaderzugehörigkeit der aktiven Saison einen Zugang hat — als Spieler des Stammkaders
oder des erweiterten Kaders, als Trainer oder als Elternteil eines solchen Spielers.

Das System SHALL als eigenen Spieler jeden Spieler ausweisen, der einem Mitglied des
Nutzers oder einem seiner Kinder zugeordnet ist.

Das System SHALL die Zuordnung einer Mannschaft zur Mannschaftsbezeichnung des Verbands
aus der Verknüpfung zwischen Begegnung und eigenem Spieltermin ableiten und NICHT über
einen Vergleich der Namen erraten.

Das System SHALL keine Zugehörigkeit ausweisen, wenn sie nicht belegbar ist, statt eine
zu vermuten.

Das System SHALL die eigene Mannschaft in **allen** Tabellen und Statistiken der
Staffel-Ansicht hervorheben: Tabellenstand, Spielplan, Kreuztabelle, Tabellenverlauf und
die Mannschafts-Ranglisten.

Das System SHALL den eigenen Spieler in den Spieler-Ranglisten hervorheben.

Das System SHALL die Hervorhebung so gestalten, dass sie von einer aus anderen Gründen
fett gesetzten einzelnen Zelle unterscheidbar bleibt.

Das System SHALL die Hervorhebung nicht ausschließlich visuell tragen, sondern sie auch
für Hilfsmittel der Bedienung erkennbar machen.

#### Scenario: Eigene Mannschaft im Tabellenstand
- **WHEN** ein Spieler den Tabellenstand der Staffel seiner Mannschaft abruft
- **THEN** ist die Zeile seiner Mannschaft hervorgehoben und die übrigen Zeilen sind es nicht

#### Scenario: Eigene Mannschaft in Kreuztabelle und Verlauf
- **WHEN** ein Spieler Kreuztabelle und Tabellenverlauf seiner Staffel abruft
- **THEN** sind Zeile und Spalte seiner Mannschaft in der Kreuztabelle und ihre Linie im Verlauf hervorgehoben

#### Scenario: Eigener Spieler in der Torschützenliste
- **WHEN** ein Spieler die Torschützenliste seiner Staffel abruft und dort mit eigenen Toren geführt wird
- **THEN** ist seine Zeile hervorgehoben

#### Scenario: Elternteil sieht die Mannschaft und das Kind
- **WHEN** ein Elternteil die Staffel-Ansicht der Mannschaft seines Kindes abruft
- **THEN** sind die Mannschaft des Kindes und die Zeilen des Kindes hervorgehoben

#### Scenario: Nutzer ohne Zugehörigkeit
- **WHEN** ein Nutzer ohne Kaderzugehörigkeit zu einer Staffel deren Ansichten abruft
- **THEN** ist keine Zeile hervorgehoben und die Darstellung bleibt im Übrigen unverändert

#### Scenario: Keine Zuordnung ohne Verknüpfung
- **WHEN** zu einer Staffel keine Begegnung mit einem eigenen Spieltermin verknüpft ist
- **THEN** wird keine Mannschaft als eigene ausgewiesen, auch wenn ein Mannschaftsname dem eigenen Verein ähnelt

#### Scenario: Nicht eingeloggter Zugriff
- **WHEN** die Zugehörigkeit ohne gültiges Zugangstoken abgerufen wird
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Schiedsrichter-Rangliste einer Staffel

Das System SHALL je Staffel zu jedem in den ausgewerteten Spielberichten genannten
Schiedsrichter die Zahl seiner Spiele und die in diesen Spielen verhängten Zeitstrafen
und Karten bereitstellen.

Das System SHALL die Strafen beider Mannschaften eines Spiels zählen und kenntlich
machen, dass es sich um die Strafen des Spiels und nicht um eine Bewertung der Person
handelt.

Das System SHALL einen Bericht ohne benannte Schiedsrichter nicht in die Rangliste
aufnehmen.

Das System SHALL einen Schiedsrichter, dessen Name nicht zuverlässig von dem seines
Gespannpartners zu trennen war, in der Rangliste als unsicher kennzeichnen.

Das System SHALL die Schiedsrichter-Rangliste allen eingeloggten Nutzern zugänglich
machen.

#### Scenario: Schiedsrichter mit mehreren Spielen
- **WHEN** ein eingeloggter Nutzer die Schiedsrichter-Rangliste einer Staffel abruft, in der derselbe Schiedsrichter zwei ausgewertete Spiele geleitet hat
- **THEN** antwortet der Server mit HTTP 200 und er erscheint mit zwei Spielen und der Summe der Strafen beider Spiele

#### Scenario: Strafen beider Mannschaften zählen
- **WHEN** in einem Spiel beide Mannschaften Zeitstrafen erhalten haben
- **THEN** zählen die Strafen beider Mannschaften zum Spiel dieses Schiedsrichters

#### Scenario: Bericht ohne Schiedsrichter
- **WHEN** ein ausgewerteter Bericht keine Schiedsrichter-Namen trägt
- **THEN** erzeugt er keine Zeile in der Rangliste

#### Scenario: Unsichere Trennung ist gekennzeichnet
- **WHEN** die Namen eines Gespanns nicht zuverlässig getrennt werden konnten
- **THEN** sind die betroffenen Zeilen als unsicher gekennzeichnet
