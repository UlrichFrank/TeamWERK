## MODIFIED Requirements

### Requirement: Geleistet- und Vorhersage-Zählung je Kind

Das System SHALL für jedes Kind mit aktiver Kader-Mitgliedschaft in der aktiven
Saison zwei Zahlen aus den `duty_assignments` des Kindes zählen:
- `geleistet`: Anzahl Zuweisungen mit `event_date` in der Vergangenheit
- `vorhersage`: Anzahl Zuweisungen mit `event_date` in der Zukunft (heute
  eingeschlossen)

Der `status` einer Zuweisung ist für diese Zählung ohne Bedeutung — die Existenz der
Zuweisung genügt.

Da eine Zuweisung nur einen Account (`user_id`) trägt, SHALL das System sie wie
folgt Kindern zurechnen — die erste nicht-leere Stufe gewinnt und wird
gleichmäßig auf ihre Mitglieder geteilt:
1. eigene Mitglieder des Accounts (eigener Login oder Proxy-Account), deren
   Stammkader-Team zum Slot passt
2. Kinder des Accounts via `family_links`, deren Stammkader-Team zum Slot passt
3. eigene Mitglieder des Accounts, die im **erweiterten Kader** eines Slot-Teams
   stehen — **Aushilfe**
4. Kinder des Accounts, die im **erweiterten Kader** eines Slot-Teams stehen —
   **Aushilfe**
5. eigene Mitglieder des Accounts, unabhängig vom Team

Eine Aushilfe-Zurechnung (Stufe 3 oder 4) SHALL **nicht** in `geleistet`/`vorhersage`
des Mitglieds eingehen, sondern getrennt je (Mitglied, Aushilfe-Team) als
`aushilfe_geleistet`/`aushilfe_vorhersage` gezählt werden. Sie hat kein Soll und
verändert weder die Zahlen des Mitglieds in seinen Stammteams noch Gesamtsumme oder
Soll des Aushilfe-Teams. Passt ein Slot mit mehreren Teams zu einem Stammkader- und
einem erweiterten Team, gewinnt die Stammkader-Stufe.

Ein generischer Slot passt zu jedem Team. Eine Eltern-Zuweisung an einem Slot, zu
dem kein Kind passt, zählt für niemanden.

#### Scenario: Zuweisung an vergangenem Termin zählt als geleistet
- **WHEN** ein Kind eine Zuweisung zu einem `duty_slot` hat, dessen `event_date`
  vor dem heutigen Datum liegt
- **THEN** zählt diese Zuweisung zu `geleistet`, unabhängig vom `status`-Feld

#### Scenario: Zuweisung an zukünftigem Termin zählt als Vorhersage
- **WHEN** ein Kind eine Zuweisung zu einem `duty_slot` hat, dessen `event_date`
  heute oder in der Zukunft liegt
- **THEN** zählt diese Zuweisung zu `vorhersage`, nicht zu `geleistet`

#### Scenario: Eltern-Zuweisung wird zwischen passenden Geschwistern geteilt
- **WHEN** ein Elternteil mit zwei Kindern im selben Kader eine Zuweisung auf
  dem eigenen Account an einem Slot dieses Kaders hat
- **THEN** zählt diese Zuweisung für jedes der beiden Kinder zu 0,5

#### Scenario: Eltern-Zuweisung ohne passendes Kind zählt nicht
- **WHEN** ein Elternteil eine Zuweisung an einem Slot einer Mannschaft hat, in
  deren Kader keines seiner Kinder steht
- **THEN** zählt diese Zuweisung für keines seiner Kinder

#### Scenario: Eigene Zuweisung des Kindes zählt voll
- **WHEN** ein Kind über seinen eigenen (Proxy-)Account eine Zuweisung hat, auch
  an einem Slot einer fremden Mannschaft
- **THEN** zählt diese Zuweisung voll für das Kind

#### Scenario: Aushilfe zählt nicht aufs Stammteam
- **WHEN** ein Spieler im Stammkader von Team A und im erweiterten Kader von Team B einen vergangenen Dienst von Team B belegt hat
- **THEN** bleibt sein `geleistet` (und damit seine Zeile in der Rangliste von Team A) unverändert
- **AND** trägt er für Team B `aushilfe_geleistet = 1`

#### Scenario: Eltern-Zuweisung für ein Kind im erweiterten Kader
- **WHEN** ein Elternteil einen Dienst von Team B belegt und sein einziges passendes Kind nur im erweiterten Kader von Team B steht
- **THEN** wird die Zuweisung dem Kind als Aushilfe für Team B zugerechnet

#### Scenario: Stammkader-Kind schlägt Aushilfe-Kind
- **WHEN** ein Elternteil einen Dienst von Team B belegt, ein Kind im Stammkader von Team B und ein anderes im erweiterten Kader von Team B steht
- **THEN** zählt die Zuweisung voll für das Stammkader-Kind und nicht als Aushilfe

#### Scenario: Aushilfe ändert das Soll des fremden Teams nicht
- **WHEN** ein Aushilfe-Dienst in Team B belegt wird
- **THEN** bleiben Gesamtsumme und Soll von Team B unverändert

### Requirement: Rangliste — Sichtbarkeit für Standard-Nutzer

Ein Nutzer ohne Rolle `admin` und ohne Vereinsfunktion `vorstand` SHALL im
Team-Filter der Rangliste ausschließlich Teams zur Auswahl angeboten bekommen, zu
denen er selbst eine Verbindung hat: ein eigenes Kind ist über `family_links` im
Kader oder im erweiterten Kader dieses Teams, oder er ist selbst als Spieler im Kader
oder im erweiterten Kader dieses Teams.

Innerhalb einer für ihn sichtbaren Rangliste SHALL die Zeile des eigenen Kindes
(bzw. die eigene Zeile, falls er selbst Spieler ist) den echten Namen zeigen. Alle
anderen Zeilen SHALL ausschließlich ihre Platzierung zeigen (in der Rang-Spalte);
anstelle des Namens steht nur ein Strich „-" — kein Name, kein erfundenes
Pseudonym.

#### Scenario: Elternteil sieht nur Teams der eigenen Kinder
- **WHEN** ein Elternteil hat ein Kind im Kader der wCJ, aber kein Kind oder
  eigene Mitgliedschaft in einem anderen Kader
- **THEN** bietet der Team-Filter der Rangliste ausschließlich die wCJ zur Auswahl

#### Scenario: Eigene Zeile ist benannt, fremde nicht
- **WHEN** ein Elternteil öffnet die Rangliste des Kaders seines Kindes
- **THEN** zeigt die Zeile des eigenen Kindes dessen echten Namen
- **THEN** zeigen alle anderen Zeilen ausschließlich eine Platzierung, keinen Namen

#### Scenario: Zugriff auf fremdes Team wird verweigert
- **WHEN** ein Standard-Nutzer versucht, die Rangliste eines Teams abzurufen, zu
  dem er keine Verbindung hat
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Team des erweiterten Kaders wird angeboten
- **WHEN** ein Elternteil ein Kind im Stammkader von Team A und im erweiterten Kader von Team B hat
- **THEN** bietet der Team-Filter Team A und Team B an

## ADDED Requirements

### Requirement: Rangliste — Aushilfen getrennt ausweisen

Jeder Ranglisten-Block SHALL unterhalb der gerankten Zeilen einen eigenen Abschnitt
„Aushilfen“ zeigen, sobald mindestens ein Mitglied für dieses Team einen Aushilfe-Wert
größer 0 hat. Der Abschnitt SHALL je Mitglied `aushilfe_geleistet` und
`aushilfe_vorhersage` zeigen, **ohne** Platzierung und **ohne** Soll, und SHALL nicht in
die Sortierung der gerankten Zeilen eingehen. Für die Benennung gilt dieselbe Regel wie
in der Rangliste: Standard-Nutzer sehen nur eigene bzw. Kinder-Zeilen benannt, alle
anderen als „-"; `admin`/`vorstand` sehen alle Namen.

#### Scenario: Aushilfen stehen getrennt unter der Rangliste
- **WHEN** Team B zwei Aushilfen mit Werten > 0 hat
- **THEN** zeigt der Block von Team B unterhalb der Rangliste einen Abschnitt „Aushilfen“ mit zwei Zeilen ohne Platz

#### Scenario: Fremde Aushilfe ist anonymisiert
- **WHEN** ein Standard-Nutzer die Rangliste von Team B öffnet und eine Aushilfe nicht zu seiner Familie gehört
- **THEN** trägt diese Zeile keinen Namen und keine `memberId`

#### Scenario: Kein Abschnitt ohne Aushilfe
- **WHEN** in Team B niemand einen Aushilfe-Wert > 0 hat
- **THEN** zeigt der Block keinen Abschnitt „Aushilfen“
