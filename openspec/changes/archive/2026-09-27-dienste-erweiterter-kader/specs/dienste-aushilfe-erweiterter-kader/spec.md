## Purpose

Macht Dienste, die der erweiterte Kader oder dessen Eltern für ein Team übernehmen, als
**Aushilfe** erkennbar — in der Dienstbörse, an der Belegung und im Dashboard — und hält
sie von der Dienstpflicht des Stammteams getrennt.

## ADDED Requirements

### Requirement: Board-Gruppe kennzeichnet Aushilfe aus Sicht des Betrachters

Jede Gruppe in `GET /api/duty-board` SHALL ein boolesches Feld `aushilfe` tragen. Es SHALL
genau dann `true` sein, wenn der Betrachter mit keinem Team der Gruppe über Stammkader,
Trainer-Zugehörigkeit oder ein Kind im Stammkader verbunden ist, aber mit mindestens einem
Team der Gruppe über den erweiterten Kader (selbst oder Kind). Die Team-Menge einer Gruppe
ist bei Spielen `game_teams`, sonst `duty_slots.team_id`. Für Gruppen ohne Team und für
Betrachter ohne jede Kader-Verbindung (z. B. Vorstand) SHALL der Wert `false` sein. Das Feld
ändert **keine** Sichtbarkeit.

Das Frontend SHALL Gruppen mit `aushilfe: true` in der Dienstbörse mit einem Chip
„Aushilfe“ kennzeichnen.

#### Scenario: Gruppe des erweiterten Teams ist markiert
- **WHEN** ein Spieler nur im erweiterten Kader von Team B steht und die Dienstbörse öffnet
- **THEN** tragen die Gruppen von Team B `aushilfe: true` und zeigen den Chip „Aushilfe“

#### Scenario: Gruppe des Stammteams ist nicht markiert
- **WHEN** derselbe Spieler im Stammkader von Team A steht
- **THEN** tragen die Gruppen von Team A `aushilfe: false`

#### Scenario: Gemeinsames Spiel von Stamm- und erweitertem Team
- **WHEN** ein Spiel die Teams A und B hat und der Betrachter im Stammkader von A und im erweiterten Kader von B steht
- **THEN** trägt die Gruppe `aushilfe: false`

#### Scenario: Vorstand ohne Kader-Verbindung
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` ohne eigene Kader-Zugehörigkeit die Dienstbörse öffnet
- **THEN** tragen alle Gruppen `aushilfe: false`

### Requirement: Eingetragene tragen ein Aushilfe-Kennzeichen

Jeder Eintrag der Belegungsliste eines Slots in `GET /api/duty-board` SHALL ein boolesches
Feld `aushilfe` tragen. Es SHALL `true` sein, wenn die Zuweisung nach der Zurechnungsregel
der Dienst-Bilanz auf Stufe 3 oder 4 (Aushilfe) fällt, also der eingetragene Account weder
selbst noch über ein Kind im Stammkader oder als Trainer eines Slot-Teams verbunden ist,
wohl aber über den erweiterten Kader. Das Kennzeichen SHALL für **alle** Betrachter der
Gruppe gleich sein und keine weiteren personenbezogenen Daten preisgeben.

Das Frontend SHALL einen Eingetragenen mit `aushilfe: true` mit dem Kennzeichen „Aushilfe“
neben dem Namen zeigen.

#### Scenario: Trainer sieht, dass eine Aushilfe eingetragen ist
- **WHEN** ein Spieler des erweiterten Kaders von Team B einen Slot von Team B belegt hat und der Trainer von Team B die Dienstbörse öffnet
- **THEN** trägt dieser Eintrag `aushilfe: true` und zeigt das Kennzeichen „Aushilfe“

#### Scenario: Stammkader-Eintrag ist nicht markiert
- **WHEN** ein Elternteil eines Stammkader-Kindes von Team B einen Slot von Team B belegt hat
- **THEN** trägt der Eintrag `aushilfe: false`

#### Scenario: Kennzeichen folgt dem aktuellen Kader
- **WHEN** eine Aushilfe später in den Stammkader von Team B aufgenommen wird
- **THEN** trägt ihr bestehender Eintrag ab dann `aushilfe: false`

### Requirement: Dashboard „Meine Dienste“ zeigt Aushilfe getrennt

Die Kachel „Meine Dienste“ in `GET /api/dashboard` SHALL ihren bestehenden Block
(nächstes Spiel mit Diensten, eigene Zusagen, offene Slots) weiterhin ausschließlich aus
Stammkader-, Kinder-im-Stammkader- und Trainer-Teams bilden. Zusätzlich SHALL sie ein Feld
`aushilfe` liefern mit:

- `mySlots`: die eigenen kommenden Zusagen (des Nutzers oder für seine Kinder), die nach
  der Zurechnungsregel Aushilfe sind, höchstens fünf, chronologisch;
- `nextGame` und `openSlotsCount`: das nächste kommende Spiel eines Teams, mit dem der
  Nutzer nur über den erweiterten Kader verbunden ist und das offene, zur Zielgruppe des
  Nutzers passende Slots hat, samt deren Anzahl.

Ist beides leer, SHALL `aushilfe` `null` sein, und das Frontend SHALL den Block nicht
zeigen. Andernfalls SHALL das Frontend die Aushilfe-Zeilen unter den Stamm-Zeilen zeigen,
ohne eigene Abschnitts-Überschrift: jede Aushilfe-Zeile trägt hinter dem Titel dasselbe
Kennzeichen „Aushilfe“ wie die Dienstbörse.

Jeder Eintrag in `mySlots` (Stamm und Aushilfe) SHALL `slotId` und `teamLabel` tragen —
`teamLabel` nennt die Mannschaft(en) des Nutzers, zu denen der Slot gehört. Das Frontend
SHALL die Mannschaft in beiden Blöcken gleich, durch „·“ abgetrennt, in der Unterzeile
zeigen. Ein Klick auf eine Dienst-Zeile SHALL auf `/dienste?focus=slot-<slotId>` springen,
ein Klick auf eine „offene Dienste“-Zeile auf `/dienste?focus=game-<gameId>`.

Die Dienst-Bilanz-Kachel (`meineDienste.dutyAccount`) SHALL Aushilfe-Positionen je
(Mitglied, Aushilfe-Team) in einem eigenen Feld `dutyAccountAushilfe` liefern — nur
`geleistet` und `vorhersage`, kein `soll`. Das Frontend SHALL sie in einem eigenen
Abschnitt „Aushilfe“ unter den Stammteam-Positionen zeigen, ohne Soll-Balken.

#### Scenario: Stamm-Block bleibt unverändert
- **WHEN** ein Spieler im Stammkader von Team A und im erweiterten Kader von Team B steht und Team B das frühere Spiel mit Diensten hat
- **THEN** zeigt `meineDienste.nextGame` weiterhin das Spiel von Team A

#### Scenario: Aushilfe-Block zeigt das nächste Spiel des erweiterten Teams
- **WHEN** derselbe Spieler die Kachel lädt und Team B morgen ein Spiel mit drei offenen passenden Slots hat
- **THEN** enthält `meineDienste.aushilfe.nextGame` dieses Spiel und `openSlotsCount = 3`

#### Scenario: Dienst-Zeile springt auf den Slot
- **WHEN** der Spieler im Dashboard auf eine eigene Dienst-Zusage (Stamm oder Aushilfe) klickt
- **THEN** öffnet sich `/dienste?focus=slot-<slotId>` mit diesem Slot im Fokus
- **AND** zeigt die Zeile die Mannschaft des Slots, bei Aushilfe zusätzlich das Kennzeichen „Aushilfe“

#### Scenario: Eigene Aushilfe-Zusage erscheint im Aushilfe-Block
- **WHEN** der Spieler einen kommenden Dienst von Team B belegt hat
- **THEN** enthält `meineDienste.aushilfe.mySlots` diese Zusage
- **AND** erscheint sie nicht in `meineDienste.mySlots`

#### Scenario: Kein erweiterter Kader, kein Block
- **WHEN** ein Nutzer weder selbst noch über ein Kind im erweiterten Kader eines Teams steht
- **THEN** ist `meineDienste.aushilfe` `null` und das Dashboard zeigt keinen Aushilfe-Abschnitt

#### Scenario: Bilanz trennt Aushilfe
- **WHEN** ein Kind im Stammkader von Team A 4 Dienste und als Aushilfe in Team B 2 Dienste geleistet hat
- **THEN** zeigt `dutyAccount` für Team A `geleistet = 4` mit Soll
- **AND** zeigt `dutyAccountAushilfe` für Team B `geleistet = 2` ohne Soll
