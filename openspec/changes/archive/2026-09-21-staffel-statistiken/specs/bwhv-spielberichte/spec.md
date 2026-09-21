## ADDED Requirements

### Requirement: Schiedsrichter eines Berichts als einzelne Personen

Das System SHALL die im Kopf eines Spielberichts genannten Schiedsrichter als einzelne
Personen erfassen, nicht als eine gemeinsame Textzeile.

Das System SHALL die Trennung der Namen aus den Spaltenpositionen der Textebene des
Dokuments ableiten — derselbe Weg, auf dem die Mannschaftsliste ihre Spaltengrenzen
gewinnt — und NICHT aus einer Vermutung über den Aufbau der Namen.

Das System SHALL nur dann auf eine Namensregel zurückfallen, wenn das Dokument die Namen
in einer einzigen Spalte liefert, und in diesem Fall die Trennung als unsicher am Bericht
vermerken.

Das System SHALL die ungetrennte Namenszeile des Dokuments zusätzlich erhalten, damit eine
später verbesserte Trennung ohne erneuten Fremdabruf möglich bleibt.

Das System SHALL Platzhalter-Einträge ohne echten Namen nicht als Schiedsrichter erfassen.

Das System SHALL einen Bericht wegen einer nicht trennbaren Schiedsrichter-Zeile NICHT
als fehlgeschlagen behandeln — die Namen sind für die Auswertung des Spiels entbehrlich.

#### Scenario: Zwei Spalten ergeben zwei Personen
- **WHEN** ein Bericht die beiden Schiedsrichter in getrennten Spalten der Textebene führt
- **THEN** werden zwei Schiedsrichter erfasst und die Trennung ist nicht als unsicher vermerkt

#### Scenario: Eine Spalte wird als unsicher gekennzeichnet
- **WHEN** ein Bericht beide Namen in einer einzigen Spalte führt
- **THEN** wird nach der Namensregel getrennt und die Trennung als unsicher am Bericht vermerkt

#### Scenario: Platzhalter erzeugt keinen Schiedsrichter
- **WHEN** der Bericht an der Stelle der Schiedsrichter einen Platzhalter ohne Person trägt
- **THEN** wird kein Schiedsrichter erfasst

#### Scenario: Nicht trennbare Zeile kippt den Bericht nicht
- **WHEN** die Schiedsrichter-Zeile eines Berichts gar nicht interpretierbar ist
- **THEN** wird der Bericht dennoch ausgewertet und erhält NICHT den Zustand `parse_failed`
