## ADDED Requirements

### Requirement: Aufstellungsstatus in der Tabellenansicht

`GET /api/teams/{id}/rsvp-matrix` SHALL in jeder Zelle einer Spalte vom Typ `heim` oder
`auswärts` den Aufstellungsstatus des Spielers als `lineup` (`in` | `out` | `open`,
Capability `spiel-aufstellung`) liefern. Zellen von Trainings und generischen Terminen
SHALL kein `lineup` tragen. Der Status SHALL für alle Aufrufer sichtbar sein, die die
Matrix sehen dürfen — er ist, wie auf der Spieldetailseite, keine vertrauliche Angabe.

Die Tabellenansicht auf `/termine?view=tabelle` SHALL den Status als Fläche hinter dem
Rückmeldesymbol jeder Spielzelle zeigen, in derselben Darstellung wie Liste und
Detailseite:

- `in` („aufgestellt"): vollflächig in Markengrün, das Rückmeldesymbol darauf weiß;
- `out` („nicht aufgestellt"): vollflächig grau;
- `open` („Aufstellung offen"): nur ein grau gestrichelter Rahmen, innen vollständig
  transparent (keine Füllung).

Alle Zellen SHALL dieselbe Geometrie haben. Die Farbe SHALL nicht der einzige Träger der
Information sein: Titel bzw. zugängliche Bezeichnung der Zelle SHALL den Status benennen,
und die Legende SHALL alle drei Zustände mit ihren Bezeichnungen zeigen.

#### Scenario: Spalte mit gespeicherter Aufstellung

- **WHEN** die Aufstellung eines Spiels in der Matrix Spieler A enthält, Spieler B nicht
- **THEN** ist die Zelle von A grün und die von B grau gefüllt
- **AND** benennt der Titel der Zelle von B „nicht aufgestellt" zusammen mit der Rückmeldung

#### Scenario: Spalte ohne Aufstellung

- **WHEN** für ein Spiel keine Aufstellung gespeichert ist
- **THEN** sind seine Zellen grau gestrichelt umrandet und innen transparent

#### Scenario: Trainingsspalte

- **WHEN** die Matrix eine Trainingsspalte enthält
- **THEN** tragen deren Zellen weder `lineup` noch eine Aufstellungsfläche oder -umrandung

#### Scenario: Live-Aktualisierung

- **WHEN** ein Trainer die Aufstellung eines Spiels speichert, während die Tabellenansicht dieser Mannschaft geöffnet ist
- **THEN** zeigt die Tabelle den neuen Status ohne manuelles Neuladen
