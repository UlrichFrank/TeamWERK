## MODIFIED Requirements

### Requirement: Ansicht „Staffeln" mit Mannschafts-Umschalter

Das System SHALL eine Oberfläche bereitstellen, die über einen Mannschafts-Umschalter
zwischen den zugeordneten Staffeln wechselt und Tabelle, Spielplan, Kreuztabelle,
Verlauf, Mannschafts-Statistiken und Spieler-Ranglisten in getrennten Reitern zeigt.

Das System SHALL im Reiter „Verlauf" zwei Darstellungen zeigen: den Tabellenverlauf der
Staffel und darunter die Spielmatrix der dem Nutzer zuzurechnenden Mannschaft.

Das System SHALL den Tabellenverlauf grafisch als Platzierung je Spieltag darstellen,
mit der besten Platzierung oben, und dabei jede Mannschaft unterscheidbar kennzeichnen.

Das System SHALL die Spielmatrix als Tabelle mit den Spielern in den Zeilen und den
Begegnungen in den Spalten darstellen, mit einer Summenspalte je Spieler und einer
Summenzeile je Begegnung, und die Spielerspalte beim waagerechten Blättern stehen
lassen.

Das System SHALL im Kopf jeder Spalte der Spielmatrix Datum, Paarung und Endstand der
Begegnung zeigen und dort den Zugang zur Darstellung des Torverlaufs anbieten, sofern
Tordaten vorliegen.

Das System SHALL unterhalb der Spielmatrix ausweisen, wie viele Begegnungen sie umfasst
und zu wie vielen davon ein ausgewerteter Spielbericht vorliegt.

Das System SHALL auf schmalen Bildschirmen je Begegnung nur die Tore zeigen und die
übrigen Werte erst ab der Mobile-/Desktop-Grenze.

Das System SHALL einen Hinweis statt der Spielmatrix zeigen, wenn dem Nutzer in dieser
Staffel keine Mannschaft zuzurechnen ist.

Das System SHALL die Kreuztabelle als Matrix mit der Heimmannschaft in der Zeile und der
Gastmannschaft in der Spalte darstellen und die Zuordnung der Achsen beschriften.

Das System SHALL diese Ansicht über einen eigenen Navigationseintrag erreichbar machen,
der allen eingeloggten Nutzern angezeigt wird.

Das System SHALL den Spielbericht einer Begegnung im Spielplan dieser Ansicht
aufklappbar zeigen und ihn zusätzlich in der Detailansicht des eigenen Spiels anbieten.

Das System SHALL den gewählten Reiter und die gewählte Mannschaft in der Adresse der
Seite führen, sodass ein Aufruf derselben Adresse dieselbe Darstellung zeigt.

Das System SHALL in Tabellenstand und Spielplan die dem Nutzer zuzurechnende Mannschaft
hervorheben.

#### Scenario: Wechsel zwischen Mannschaften
- **WHEN** ein Nutzer im Umschalter eine andere Mannschaft wählt
- **THEN** zeigen alle Reiter die Staffel dieser Mannschaft

#### Scenario: Navigationseintrag ist für alle sichtbar
- **WHEN** ein eingeloggter Nutzer ohne besondere Vereinsfunktion die Anwendung öffnet
- **THEN** ist der Navigationseintrag „Staffeln" sichtbar

#### Scenario: Eigene Mannschaft ist in Tabelle und Spielplan hervorgehoben
- **WHEN** ein Nutzer mit Kaderzugehörigkeit Tabellenstand und Spielplan seiner Staffel öffnet
- **THEN** ist seine Mannschaft in beiden Reitern hervorgehoben

#### Scenario: Verlauf zeigt Staffel und eigene Mannschaft
- **WHEN** ein Nutzer mit Kaderzugehörigkeit den Reiter „Verlauf" öffnet
- **THEN** sieht er oben den Tabellenverlauf der Staffel und darunter die Spielmatrix seiner Mannschaft

#### Scenario: Torverlauf einer Begegnung öffnen
- **WHEN** der Nutzer im Spaltenkopf einer Begegnung mit ausgewertetem Bericht den Zugang zum Torverlauf wählt
- **THEN** öffnet sich die Darstellung des Torverlaufs dieser Begegnung

#### Scenario: Verlauf ohne zuzurechnende Mannschaft
- **WHEN** ein Nutzer ohne Kaderzugehörigkeit in dieser Staffel den Reiter „Verlauf" öffnet
- **THEN** sieht er den Tabellenverlauf und an Stelle der Spielmatrix einen Hinweis, warum sie fehlt

#### Scenario: Spielbericht im Spielplan
- **WHEN** ein Nutzer im Spielplan dieser Ansicht eine Begegnung mit ausgewertetem Bericht aufklappt
- **THEN** zeigt die Ansicht Mannschaftslisten und Spielverlauf dieser Begegnung

#### Scenario: Reiter ist verlinkbar
- **WHEN** ein Nutzer die Adresse einer Ansicht mit gewähltem Reiter und gewählter Mannschaft erneut aufruft
- **THEN** zeigt die Seite denselben Reiter derselben Mannschaft

#### Scenario: Staffel ohne abgerufene Daten
- **WHEN** einer Mannschaft eine Staffel zugeordnet ist, zu der noch nichts abgerufen wurde
- **THEN** zeigt die Ansicht in jedem Reiter den Hinweis auf den ausstehenden Abruf statt einer leeren Darstellung
