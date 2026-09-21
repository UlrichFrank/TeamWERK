## MODIFIED Requirements

### Requirement: Ansicht „Staffeln" mit Mannschafts-Umschalter

Das System SHALL eine Oberfläche bereitstellen, die über einen Mannschafts-Umschalter
zwischen den zugeordneten Staffeln wechselt und Tabelle, Spielplan, Kreuztabelle,
Tabellenverlauf, Mannschafts-Statistiken und Spieler-Ranglisten in getrennten Reitern
zeigt.

Das System SHALL den Tabellenverlauf grafisch als Platzierung je Spieltag darstellen,
mit der besten Platzierung oben, und dabei jede Mannschaft unterscheidbar kennzeichnen.

Das System SHALL die Kreuztabelle als Matrix mit der Heimmannschaft in der Zeile und der
Gastmannschaft in der Spalte darstellen und die Zuordnung der Achsen beschriften.

Das System SHALL diese Ansicht über einen eigenen Navigationseintrag erreichbar machen,
der allen eingeloggten Nutzern angezeigt wird.

Das System SHALL den Spielbericht einer einzelnen Begegnung NICHT in dieser Ansicht,
sondern in der Detailansicht des Spiels zeigen.

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

#### Scenario: Reiter ist verlinkbar
- **WHEN** ein Nutzer die Adresse einer Ansicht mit gewähltem Reiter und gewählter Mannschaft erneut aufruft
- **THEN** zeigt die Seite denselben Reiter derselben Mannschaft

#### Scenario: Staffel ohne abgerufene Daten
- **WHEN** einer Mannschaft eine Staffel zugeordnet ist, zu der noch nichts abgerufen wurde
- **THEN** zeigt die Ansicht in jedem Reiter den Hinweis auf den ausstehenden Abruf statt einer leeren Darstellung
