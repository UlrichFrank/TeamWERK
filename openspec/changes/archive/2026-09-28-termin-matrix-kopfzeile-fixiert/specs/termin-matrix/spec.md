## MODIFIED Requirements

### Requirement: Tabellenansicht auf /termine

Die Seite `/termine` SHALL über einen Umschalter in der Kopfzeile zwischen Liste
(Default) und Tabelle wechseln; der Zustand SHALL im URL-Parameter `view=tabelle`
stehen. In der Tabellenansicht SHALL genau eine Mannschaft gewählt sein (`team=<id>`,
Einfachauswahl über die Mannschaften des Nutzers); Übungsgruppen SHALL nicht zur
Auswahl stehen. Typ-Filter und „Vergangene" SHALL auf die Spalten wirken.

Die Tabelle SHALL je Spieler zwei Teilnahme-Spalten zeigen, jeweils als Anzahl und Anteil
an den sichtbaren, nicht abgesagten und nicht per Serie abgemeldeten Terminen:
„Bisher" über die Termine vor heute (erfasste Anwesenheit, wo vorhanden, sonst Zusage)
und „Geplant" über die Termine ab heute (Zusagen, auch per Voreinstellung). Die Namensspalte SHALL beim
horizontalen Scrollen stehen bleiben; die Titelzeile (Spaltenköpfe) SHALL beim
vertikalen Scrollen am oberen Rand der Tabelle stehen bleiben. Die Kopfzelle der
Namensspalte SHALL in beide Richtungen stehen bleiben und von keiner anderen Zelle
überdeckt werden. Die Tabelle SHALL dafür höchstens so hoch sein wie der sichtbare
Bereich, sodass Titelzeile und horizontale Scrollleiste gleichzeitig erreichbar sind.
Ein Spaltenkopf SHALL zur Termin-Detailseite führen. Die Ansicht SHALL sich bei
`trainings`/`games`-Events live aktualisieren.

#### Scenario: Umschalten auf die Tabelle
- **WHEN** ein Nutzer auf `/termine` „Tabelle" wählt
- **THEN** enthält die URL `view=tabelle` und `team=<id>`
- **AND** die Seite zeigt eine Zeile je Kaderspieler und eine Spalte je Termin

#### Scenario: Typ-Filter wirkt auf Spalten und Quote
- **WHEN** in der Tabellenansicht nur „Training" aktiv ist
- **THEN** zeigt die Tabelle nur Trainingsspalten
- **AND** die Teilnahme-Quote rechnet nur über diese Spalten

#### Scenario: Teilnahme nach Vergangenheit und Zukunft getrennt
- **WHEN** ein Spieler zwei vergangenen Trainings zugesagt und eines davon laut Erfassung verpasst hat und einem künftigen Training zugesagt hat
- **THEN** zeigt die Trainer-Sicht „Bisher" `1 (50 %)` und „Geplant" `1 (100 %)`

#### Scenario: Titelzeile bleibt beim vertikalen Scrollen stehen
- **WHEN** die Tabelle mehr Spielerzeilen hat, als in den sichtbaren Bereich passen, und der Nutzer in der Tabelle nach unten scrollt
- **THEN** bleiben Terminsymbol und Datum jeder Spalte sowie die Köpfe „Spieler", „Bisher" und „Geplant" am oberen Rand der Tabelle sichtbar
- **AND** die Zellen der Spielerzeilen laufen unter der Titelzeile hindurch, ohne durchzuscheinen

#### Scenario: Kopfzelle „Spieler" bleibt in beide Richtungen stehen
- **WHEN** der Nutzer die Tabelle nach unten und nach rechts gescrollt hat
- **THEN** steht die Kopfzelle „Spieler" weiterhin oben links
- **AND** weder Terminköpfe noch Spielernamen überdecken sie

#### Scenario: Horizontale Scrollleiste ohne Scrollen ans Tabellenende
- **WHEN** die Tabelle breiter und höher ist als der sichtbare Bereich
- **THEN** ist die horizontale Scrollleiste der Tabelle erreichbar, ohne zuvor bis zur letzten Spielerzeile zu scrollen
