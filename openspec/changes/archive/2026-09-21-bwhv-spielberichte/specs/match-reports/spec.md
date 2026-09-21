## ADDED Requirements

### Requirement: Ergebnisfelder des redaktionellen Spielberichts

Das System SHALL die Ergebnisfelder eines redaktionellen Spielberichts — Endstand beider
Mannschaften und Halbzeitstand beider Mannschaften — im Bearbeitungsformular aus dem
offiziellen BWHV-Spielbericht vorbefüllen, sofern zu diesem Spiel ein ausgewerteter
Bericht vorliegt.

Das System SHALL die vorbefüllten Werte editierbar lassen; eine Änderung durch den Autor
SHALL Vorrang haben und NICHT durch eine spätere Vorbefüllung überschrieben werden.

Das System SHALL das Formular unverändert wie bisher anbieten, wenn kein ausgewerteter
Bericht vorliegt.

Die Vorbefüllung SHALL im Frontend erfolgen. Das Backend des redaktionellen Spielberichts
SHALL das Paket der Spielstatistik NICHT importieren, da beide Domänen-Pakete sind und ein
solcher Import die Architektur-Regel „Domänen-Pakete importieren sich nicht gegenseitig"
verletzen würde.

#### Scenario: Vorbefüllung bei vorhandenem Bericht
- **WHEN** ein Autor das Formular für ein Spiel öffnet, zu dem ein ausgewerteter BWHV-Spielbericht vorliegt
- **THEN** sind Endstand und Halbzeitstand beider Mannschaften mit den Werten aus dem Bericht vorbelegt

#### Scenario: Kein Bericht vorhanden
- **WHEN** ein Autor das Formular für ein Spiel öffnet, zu dem kein ausgewerteter Bericht vorliegt
- **THEN** bleiben die Ergebnisfelder leer wie bisher

#### Scenario: Eingabe des Autors hat Vorrang
- **WHEN** ein Autor einen vorbefüllten Wert ändert und speichert
- **THEN** bleibt der eingegebene Wert erhalten

#### Scenario: Fehlgeschlagener Bericht füllt nicht vor
- **WHEN** zu einem Spiel ein Bericht im Zustand `parse_failed` vorliegt
- **THEN** bleiben die Ergebnisfelder leer
