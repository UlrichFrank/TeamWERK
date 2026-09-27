## ADDED Requirements

### Requirement: Einheitlicher Aufstellungsstatus

Das System SHALL für jedes Paar aus Spiel und Kader-Spieler genau einen von drei
Aufstellungszuständen führen, abgeleitet aus `game_lineup`:

| Code (API) | Bezeichnung (UI, iCal) | Bedingung |
|---|---|---|
| `in` | aufgestellt | eine Zeile für dieses Spiel **und** dieses Mitglied |
| `out` | nicht aufgestellt | mindestens eine Zeile für dieses Spiel, aber keine für dieses Mitglied |
| `open` | Aufstellung offen | **keine** Zeile für dieses Spiel |

Aus einer leeren Aufstellung SHALL das System an keiner Stelle „nicht aufgestellt"
ableiten. Die Ableitung SHALL für alle Mannschaften des Spiels gemeinsam gelten
(`game_lineup` hängt am Spiel, nicht an der Mannschaft).

Der Status SHALL nur für Spieler des Stamm- oder erweiterten Kaders gelten, nie für
Trainer und nie für Trainings. Liste, Tabelle und iCal-Feed SHALL ihn nur bei Spielen
vom Typ `heim` oder `auswärts` ausweisen; die Spieldetailseite SHALL ihn bei jedem
Termin aus `games` zeigen, also auch bei `generisch` („Sonstiges"), weil Trainer dort
ebenfalls eine Aufstellung pflegen können. Wo kein Status gilt, lässt die API das Feld weg.

Die Oberfläche SHALL die drei Zustände überall mit genau diesen Bezeichnungen und
derselben Darstellung zeigen (aufgestellt vollflächig in Markengrün mit weißer Schrift
bzw. weißen Symbolen, nicht aufgestellt vollflächig grau, Aufstellung offen nur grau
gestrichelt umrandet und innen transparent) — Liste und Tabelle auf `/termine`, Spieldetailseite und iCal-Feed. Die
Spaltenüberschrift heißt „Aufstellung" (mobil „Aufst.").

#### Scenario: Keine Aufstellung gespeichert

- **WHEN** für ein Heimspiel keine Zeile in `game_lineup` existiert
- **THEN** ist der Status jedes Kader-Spielers `open` („Aufstellung offen")
- **AND** zeigt keine Oberfläche für dieses Spiel „nicht aufgestellt"

#### Scenario: Aufstellung gespeichert

- **WHEN** die Aufstellung eines Spiels die Mitglieder A und B enthält und C im Kader steht
- **THEN** ist der Status von A und B `in` („aufgestellt") und der von C `out` („nicht aufgestellt")

#### Scenario: Aufstellung einer anderen Mannschaft desselben Spiels

- **WHEN** ein Spiel zwei Mannschaften zugeordnet ist und die Aufstellung nur Mitglieder der ersten Mannschaft enthält
- **THEN** ist der Status der Spieler der zweiten Mannschaft `out`

#### Scenario: Generisches Event

- **WHEN** ein Termin vom Typ `generisch` in Liste, Tabelle oder iCal-Feed erscheint
- **THEN** trägt er dort keinen Aufstellungsstatus
- **AND** zeigt seine Detailseite die Spalte „Aufstellung" mit denselben drei Zuständen wie ein Spiel

## MODIFIED Requirements

### Requirement: Aufstellung ist per Participants-Endpoint abrufbar

Das System SHALL `GET /api/games/{id}/participants` bereitstellen, der alle regulären und erweiterten Kader-Mitglieder des Teams zurückgibt, jeweils mit RSVP-Status (`rsvp_status`, nullable), Lineup-Status (`in_lineup: bool`) und dem dreiwertigen Aufstellungsstatus (`lineup`: `in` | `out` | `open`). Der Aufstellungsstatus SHALL serverseitig über die gesamte Aufstellung des Spiels abgeleitet werden — auch über Zeilen, die dem Aufrufer nicht ausgeliefert werden. Trainerzeilen SHALL kein `lineup`-Feld tragen; Termine vom Typ `generisch` tragen es wie Spiele. Die Antwort SHALL zusätzlich `lineup_count` liefern — die Zahl der Mitglieder in der gespeicherten Aufstellung des Spiels, ebenfalls unabhängig davon, welche Zeilen ausgeliefert werden.

#### Scenario: Participant-Liste enthält reguläre und erweiterte Mitglieder

- **WHEN** ein Trainer `GET /api/games/{id}/participants` abruft
- **THEN** enthält die Antwort sowohl `kader_members` (mit RSVP-Status) als auch `kader_extended_members` (mit `rsvp_status: null`) des Teams

#### Scenario: Lineup-Status ist korrekt gesetzt

- **WHEN** ein Mitglied in `game_lineup` für dieses Spiel eingetragen ist
- **THEN** enthält sein Eintrag in der Participants-Antwort `in_lineup: true` und `lineup: "in"`

#### Scenario: Aufstellung besteht nur aus für den Aufrufer unsichtbaren Zeilen

- **WHEN** ein Spieler die Participants eines Spiels abruft, dessen Aufstellung ausschließlich Mitglieder enthält, die ihm nicht ausgeliefert werden
- **THEN** trägt seine eigene Zeile `lineup: "out"`, nicht `"open"`
- **AND** entspricht `lineup_count` der Zahl aller Zeilen in `game_lineup` für dieses Spiel

#### Scenario: Keine Aufstellung

- **WHEN** für das Spiel keine Zeile in `game_lineup` existiert
- **THEN** tragen alle Spielerzeilen `lineup: "open"` und Trainerzeilen kein `lineup`-Feld

### Requirement: Spieldetail zeigt Aufstellungs-Spalte

Das System SHALL auf `/termine/spiel/{id}` in der Teilnahme-Tabelle eine Spalte „Aufstellung" anzeigen. Jede Spielerzeile SHALL eine Checkbox tragen, die die drei Zustände in der Darstellung von Liste und Tabelle zeigt: grün mit weißem Haken (aufgestellt), grau gefüllt (nicht aufgestellt), grau gestrichelt umrandet und innen transparent (Aufstellung offen); die zugängliche Bezeichnung nennt den Zustand. Trainer können die Checkbox bedienen, für Spieler und Eltern ist sie nur lesend. Der Kopf der Teilnahme-Karte SHALL den Gesamtzustand als Text-Kennzeichen ohne Symbol im Stil der Liste nennen: „Aufstellung offen" (gestrichelt), solange keine Aufstellung gespeichert ist, sonst „N aufgestellt" (grün) mit N = `lineup_count`.

#### Scenario: Trainer kann Aufstellung über Checkbox setzen

- **WHEN** ein Trainer die Checkbox eines Mitglieds in der Aufstellungs-Spalte aktiviert
- **THEN** wird dieses Mitglied in die Aufstellung aufgenommen (optimistic update + API-Call)
- **AND** wechselt der Kartenkopf von „Aufstellung offen" auf „1 aufgestellt"

#### Scenario: Spieler sieht Aufstellung read-only

- **WHEN** ein Spieler die Spieldetail-Seite eines Spiels mit gespeicherter Aufstellung öffnet
- **THEN** sieht er in der Aufstellungs-Spalte je Zeile eine nicht bedienbare Checkbox, grün angehakt für „aufgestellt" oder grau gefüllt für „nicht aufgestellt"

#### Scenario: Spieler sieht offene Aufstellung

- **WHEN** ein Spieler die Spieldetail-Seite eines Spiels ohne gespeicherte Aufstellung öffnet
- **THEN** zeigt die Aufstellungs-Spalte für alle Spielerzeilen eine grau gestrichelt umrandete, nicht bedienbare Checkbox
- **AND** erscheint nirgends die Bezeichnung „nicht aufgestellt"

#### Scenario: Erweitertes Kader-Mitglied erscheint in Aufstellungs-Tabelle

- **WHEN** ein erweitertes Kader-Mitglied für das Team eingetragen ist
- **THEN** erscheint es in der Teilnahme-Tabelle ohne RSVP-Status, aber mit Aufstellungs-Checkbox (für Trainer bedienbar, sonst nur lesend)
