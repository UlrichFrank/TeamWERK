# Spec Delta

## MODIFIED Requirements

### Requirement: Ein Dienst-Slot trägt seine eigene Dauer

Jeder `duty_slots`-Eintrag SHALL eine Spalte `hours_value` (`REAL`, Stunden, `NOT NULL`,
`> 0`) tragen. Dieser Wert SHALL zugleich die angezeigte Dauer **und** die Gutschrift auf
das Dienstkonto sein — es SHALL keine zweite Zahl für dieselbe Größe geben.

Der Wert SHALL beim Erzeugen des Slots **materialisiert** werden, wie `event_time`,
`slots_total` und `audiences` es bereits werden. Ein Slot SHALL zur Laufzeit weder die
Vorlage noch den Diensttyp nach seiner Dauer befragen.

Jede Auswertung geleisteter Dienststunden (Dienst-Bilanz, Rangliste) SHALL die Dauer aus
`duty_slots.hours_value` lesen, nicht aus `duty_types.hours_value`.

#### Scenario: Dauer eines Slots weicht vom Diensttyp ab

- **WHEN** ein Vorstand die Dauer eines Slots von 1,0 auf 2,0 Stunden ändert
- **THEN** zeigt die Dienstbörse für diesen Slot die längere Spanne
- **AND** zählt eine Auswertung geleisteter Stunden diesen Slot mit 2,0 Stunden
- **AND** bleibt die Dauer des zugrundeliegenden Diensttyps unverändert

#### Scenario: Spätere Änderung am Diensttyp erreicht bestehende Slots nicht

- **WHEN** die Dauer eines Diensttyps nach dem Erzeugen von Slots geändert wird
- **THEN** behalten die bestehenden Slots ihre materialisierte Dauer
- **AND** tragen erst neu erzeugte Slots den geänderten Wert
