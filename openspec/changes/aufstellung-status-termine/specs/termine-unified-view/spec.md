## ADDED Requirements

### Requirement: Aufstellungsstatus auf Spielkarten der Liste

Die Listenansicht von `/termine` SHALL auf jeder Karte eines Spiels vom Typ `heim` oder
`auswärts` den Aufstellungsstatus (Capability `spiel-aufstellung`) anzeigen — für die
eigene Zeile, wenn der Nutzer als Spieler im Stamm- oder erweiterten Kader steht, und
für jede Kind-Zeile eines Elternteils. Die Anzeige SHALL ein Kennzeichen sein, das nur die
Bezeichnung als Text trägt (aufgestellt / nicht aufgestellt / Aufstellung offen), ohne
Symbol, mit fester, für alle drei Zustände gleicher Breite, rechts in der Zeile der
Zu-/Absage-Knöpfe. Darstellung wie in der Tabelle: aufgestellt vollflächig in Markengrün mit
weißer Schrift, nicht aufgestellt vollflächig grau, Aufstellung offen nur grau
gestrichelt umrandet und innen transparent.

Das Grün der Zusage auf `/termine` (aktiver „Zusagen“-Knopf, Zusage-Zähler) SHALL
dasselbe Markengrün sein wie die Fläche „aufgestellt“.

`GET /api/games/my` SHALL dazu je Spiel `my_lineup` (`in` | `out` | `open`) und je
Eintrag in `children_rsvp` ein Feld `lineup` liefern. Für Trainer ohne Spielerzugehörigkeit,
Nicht-Teilnehmer und generische Events SHALL das Feld fehlen, und die Karte SHALL dann
kein Kennzeichen zeigen. Trainings SHALL kein Kennzeichen tragen.

Das Kennzeichen SHALL live aktualisiert werden: speichert ein Trainer die Aufstellung,
zeigen offene Listen den neuen Status ohne manuelles Neuladen.

#### Scenario: Spieler sieht eigene Aufstellung in der Liste

- **WHEN** ein Stammspieler `/termine` öffnet und für ein kommendes Heimspiel aufgestellt ist
- **THEN** zeigt die Karte neben seinen Zu-/Absage-Knöpfen das Kennzeichen „aufgestellt"

#### Scenario: Aufstellung noch offen

- **WHEN** für ein Spiel keine Aufstellung gespeichert ist
- **THEN** zeigt die Karte das Kennzeichen „Aufstellung offen"

#### Scenario: Elternteil sieht den Status je Kind

- **WHEN** ein Elternteil mit zwei Kindern im Kader ein Spiel sieht, in dessen Aufstellung nur das erste Kind steht
- **THEN** trägt die Zeile des ersten Kindes „aufgestellt" und die des zweiten „nicht aufgestellt"

#### Scenario: Trainer ohne Spielerrolle

- **WHEN** ein Trainer, der nur über `kader_trainers` am Spiel hängt, die Liste öffnet
- **THEN** zeigt die Karte kein Aufstellungskennzeichen

#### Scenario: Live-Aktualisierung

- **WHEN** ein Trainer die Aufstellung eines Spiels speichert, während ein Spieler die Liste geöffnet hat
- **THEN** wechselt das Kennzeichen des Spielers ohne Neuladen der Seite
