## ADDED Requirements

### Requirement: Spielmatrix der eigenen Mannschaft

Das System SHALL je Staffel die Einzelwerte jedes Spielers der dem Nutzer
zuzurechnenden Mannschaft für jede ihrer Begegnungen bereitstellen: Tore,
Siebenmeter-Versuche, Siebenmeter-Tore, Zeitstrafen, Verwarnungen und
Disqualifikationen.

Das System SHALL zusätzlich je Spieler eine Summe über alle Begegnungen samt der Zahl
der Spiele und je Begegnung eine Summe über alle Spieler der Mannschaft ausweisen.

Das System SHALL die Mannschaft aus der Verknüpfung zwischen BWHV-Begegnung und eigenem
Spieltermin ableiten und sie NICHT über einen Vergleich von Mannschaftsnamen bestimmen.

Das System SHALL mehrere dem Nutzer zuzurechnende Mannschaften derselben Staffel
getrennt ausweisen, jede mit eigener Spiel- und Spielerliste.

Das System SHALL eine leere Menge liefern, wenn dem Nutzer in dieser Staffel keine
Mannschaft zuzurechnen ist.

Das System SHALL je Begegnung mit erfasstem Ergebnis eine Position der Matrix bilden,
auch wenn zu ihr kein ausgewerteter Spielbericht vorliegt, und solche Begegnungen als
„ohne Bericht" kennzeichnen.

Das System SHALL eine Begegnung ohne erfasstes Ergebnis NICHT in die Matrix aufnehmen.

Das System SHALL zwischen „der Spieler stand in der Mannschaftsliste dieser Begegnung
nicht" und „er stand darin und hat nicht getroffen" unterscheidbar antworten.

Das System SHALL die dritte Zeitstrafe eines Spielers als eine Zeitstrafe weniger und
eine Disqualifikation mehr werten, wie in allen anderen Auswertungen dieser Staffel.

Das System SHALL die Mannschaft einer Spielerzeile über die Seite der Begegnung
bestimmen und damit die Schreibweise des Spielplans sprechen, nicht die des
Spielberichts.

Das System SHALL Spielerzeilen aus Berichten, deren Auswertung gescheitert ist, NICHT
einbeziehen.

Das System SHALL je Mannschaft die für ihre Altersklasse konfigurierte Halbzeitdauer
mitliefern und das Feld leer lassen, wenn dafür keine Regel gepflegt ist.

Das System SHALL die Matrix allen eingeloggten Nutzern zugänglich machen.

#### Scenario: Werte je Spieler und Begegnung
- **WHEN** ein Nutzer mit Kaderzugehörigkeit die Spielmatrix seiner Staffel abruft
- **THEN** antwortet der Server mit HTTP 200, und zu jedem Spieler seiner Mannschaft stehen für jede Begegnung mit ausgewertetem Bericht dessen Einzelwerte

#### Scenario: Ohne zuzurechnende Mannschaft
- **WHEN** ein Nutzer ohne Kaderzugehörigkeit in dieser Staffel die Spielmatrix abruft
- **THEN** antwortet der Server mit HTTP 200 und einer leeren Mannschaftsmenge

#### Scenario: Begegnung ohne Spielbericht
- **WHEN** eine Begegnung der Mannschaft ein Ergebnis trägt, zu ihr aber kein ausgewerteter Bericht vorliegt
- **THEN** erscheint sie als Position der Matrix mit Endstand, aber ohne Spielerwerte, und ist als „ohne Bericht" gekennzeichnet

#### Scenario: Nicht im Kader ist nicht null Tore
- **WHEN** ein Spieler in der Mannschaftsliste einer Begegnung nicht steht
- **THEN** trägt die Matrix für ihn an dieser Position keinen Wert, und nicht den Wert null

#### Scenario: Dritte Zeitstrafe zählt als Disqualifikation
- **WHEN** ein Spieler in einer Begegnung drei Zeitstrafen erhalten hat
- **THEN** weist die Matrix für ihn zwei Zeitstrafen und eine Disqualifikation aus

#### Scenario: Zwei eigene Mannschaften in einer Staffel
- **WHEN** dem Nutzer in derselben Staffel zwei Mannschaften zuzurechnen sind
- **THEN** enthält die Antwort zwei getrennte Einträge mit je eigener Spiel- und Spielerliste

#### Scenario: Unbekannte Staffel
- **WHEN** eine unbekannte Staffel-ID abgerufen wird
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Ohne Anmeldung
- **WHEN** die Spielmatrix ohne gültiges Token abgerufen wird
- **THEN** antwortet der Server mit HTTP 401

### Requirement: Tor-Momentum einer Begegnung

Das System SHALL zu einer Begegnung mit ausgewertetem Spielbericht eine Darstellung des
Torverlaufs anbieten, in der jedes Tor als Punkt auf einer Zeitachse der werfenden
Mannschaft erscheint.

Das System SHALL zu jedem Tor kenntlich machen, das wievielte Tor in Folge ohne
Gegentreffer es war, und in welcher Spielsituation — in Führung, unentschieden oder im
Rückstand — sich die werfende Mannschaft unmittelbar nach diesem Tor befand.

Das System SHALL die Spielsituation aus dem mitgezählten Spielstand der Tor-Ereignisse
bestimmen.

Das System SHALL die Darstellung nach Halbzeiten trennen und die Halbzeitgrenze aus dem
Halbzeitstand des Berichtskopfes bestimmen, NICHT aus einer angenommenen oder
konfigurierten Spieldauer.

Das System SHALL bei fehlendem Halbzeitstand eine durchgehende Zeitachse zeigen statt
einer geratenen Trennung.

Das System SHALL die Länge der Zeitachse aus der für die Altersklasse der Mannschaft
konfigurierten Halbzeitdauer bestimmen.

Das System SHALL eine konfigurierte Halbzeitdauer verwerfen, die dem Verlauf der
Begegnung widerspricht, und in diesem Fall sowie bei fehlender Konfiguration die Länge
aus dem Verlauf ableiten.

Das System SHALL sicherstellen, dass kein Tor außerhalb der dargestellten Zeitachse
liegt.

Das System SHALL ausschließlich Tore darstellen; verworfene Siebenmeter, Strafen,
Verwarnungen und Auszeiten SHALL es NICHT als Tor zeigen.

Das System SHALL Minute, Schütze und Spielstand eines Tores in Textform zugänglich
machen, sodass die Aussage nicht allein über Farbe getragen wird.

Das System SHALL die Darstellung nur zu Begegnungen anbieten, zu denen Tordaten
vorliegen.

#### Scenario: Lauf wird gezählt
- **WHEN** eine Mannschaft drei Tore hintereinander ohne Gegentreffer wirft
- **THEN** tragen die drei Tore die Laufwerte eins, zwei und drei, und das nächste Tor der Gegenseite beginnt wieder bei eins

#### Scenario: Spielsituation aus Sicht des Schützen
- **WHEN** ein Tor den Ausgleich herstellt
- **THEN** ist es als unentschieden gekennzeichnet, unabhängig davon, welche Mannschaft es geworfen hat

#### Scenario: Halbzeitgrenze aus dem Halbzeitstand
- **WHEN** der Bericht einen Halbzeitstand ausweist
- **THEN** endet die erste Halbzeit mit dem Tor, das diesen Stand herstellt, und alle späteren Tore stehen in der zweiten

#### Scenario: Ohne Halbzeitstand
- **WHEN** der Bericht keinen Halbzeitstand ausweist
- **THEN** zeigt die Darstellung eine durchgehende Zeitachse

#### Scenario: Konfigurierte Spieldauer wird genutzt
- **WHEN** für die Altersklasse der Mannschaft eine Halbzeitdauer gepflegt ist, die zum Verlauf der Begegnung passt
- **THEN** richtet sich die Zeitachse nach dieser Dauer

#### Scenario: Konfigurierte Spieldauer widerspricht dem Bericht
- **WHEN** die gepflegte Halbzeitdauer länger ist als die tatsächlich gespielte und das erste Tor der zweiten Halbzeit damit vor deren Beginn läge
- **THEN** wird die gepflegte Dauer verworfen und die Länge aus dem Verlauf abgeleitet

#### Scenario: Ohne gepflegte Altersklassen-Regel
- **WHEN** für die Altersklasse der Mannschaft keine Halbzeitdauer gepflegt ist
- **THEN** wird die Länge der Zeitachse aus dem Verlauf abgeleitet

#### Scenario: Halbzeitzuordnung hängt nicht an der Spieldauer
- **WHEN** dieselbe Begegnung mit einer anderen konfigurierten Halbzeitdauer dargestellt wird
- **THEN** stehen dieselben Tore in denselben Halbzeiten

#### Scenario: Kein Tor fällt aus der Achse
- **WHEN** das letzte Tor einer Halbzeit spät fällt
- **THEN** reicht die dargestellte Zeitachse mindestens bis zu diesem Tor

#### Scenario: Verworfener Siebenmeter ist kein Tor
- **WHEN** der Verlauf einen verworfenen Siebenmeter enthält
- **THEN** erscheint dafür kein Punkt auf der Zeitachse

#### Scenario: Keine Tordaten
- **WHEN** zu einer Begegnung kein ausgewerteter Bericht vorliegt
- **THEN** wird die Darstellung nicht angeboten
