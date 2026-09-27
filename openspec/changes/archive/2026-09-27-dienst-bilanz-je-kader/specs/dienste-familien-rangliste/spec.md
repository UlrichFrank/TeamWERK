## MODIFIED Requirements

### Requirement: Geleistet- und Vorhersage-Zählung je Kind

Das System SHALL für jede **Position** — ein Kind mit aktiver Kader-Mitgliedschaft
in der aktiven Saison **in einem bestimmten Stammkader** — zwei Zahlen aus den
`duty_assignments` zählen:
- `geleistet`: Anzahl Zuweisungen mit `event_date` in der Vergangenheit
- `vorhersage`: Anzahl Zuweisungen mit `event_date` in der Zukunft (heute
  eingeschlossen)

Steht ein Kind im Stammkader mehrerer Mannschaften, hat es je Mannschaft eine eigene
Position mit eigenen Zahlen. Eine Zuweisung an einem Dienst einer Mannschaft SHALL
nur in der Position dieser Mannschaft zählen, nie in der Position einer anderen
Mannschaft desselben Kindes.

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

Den Anteil eines Kindes aus den Stufen 1, 2 und 5 SHALL das System anschließend
gleichmäßig auf die Positionen dieses Kindes verteilen, auf die die Zuweisung passt:
- bei einem team-gebundenen Slot (Stufen 1/2) auf die Stammkader-Teams des Kindes,
  die zu den Teams des Slots gehören — bei einem Spiel nur einer Mannschaft des
  Kindes also vollständig auf diese eine Position;
- bei einem generischen Slot auf alle Stammkader-Teams des Kindes;
- bei Stufe 5 (Slot keiner Mannschaft des Kindes) auf alle Stammkader-Teams des
  Kindes.

Die Summe über alle Positionen eines Kindes SHALL damit gleich dem Anteil sein, den
die Zuweisung dem Kind zurechnet; die Summe einer Familie bleibt gleich der Zahl
ihrer tatsächlichen Dienste.

Eine Aushilfe-Zurechnung (Stufe 3 oder 4) SHALL **nicht** in `geleistet`/`vorhersage`
einer Position eingehen, sondern getrennt je (Mitglied, Aushilfe-Team) als
`aushilfe_geleistet`/`aushilfe_vorhersage` gezählt werden. Sie hat kein Soll und
verändert weder die Zahlen des Mitglieds in seinen Stammteams noch Gesamtsumme oder
Soll des Aushilfe-Teams. Passt ein Slot mit mehreren Teams zu einem Stammkader- und
einem erweiterten Team, gewinnt die Stammkader-Stufe. Die Stufen 3 und 4 SHALL nicht greifen, wenn
der Slot keinem Team zugeordnet ist oder der Account Trainer eines Teams des Slots ist.

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
- **WHEN** ein Kind in genau einem Stammkader über seinen eigenen (Proxy-)Account
  eine Zuweisung hat, auch an einem Slot einer fremden Mannschaft
- **THEN** zählt diese Zuweisung voll für die Position des Kindes in seinem
  Stammkader

#### Scenario: Dienst zählt nur für die Mannschaft, deren Dienst er ist
- **WHEN** ein Kind im Stammkader von Team A und von Team B steht und ein Elternteil
  einen vergangenen Dienst an einem Spiel nur von Team A belegt hat
- **THEN** trägt die Position (Kind, Team A) `geleistet = 1`
- **AND** bleibt die Position (Kind, Team B) bei `geleistet = 0`

#### Scenario: Generischer Dienst wird auf die Kader des Kindes geteilt
- **WHEN** ein Kind im Stammkader von Team A und von Team B steht und ein Elternteil
  einen vergangenen generischen Dienst (kein Spiel, kein Team) belegt hat
- **THEN** tragen die Positionen (Kind, Team A) und (Kind, Team B) je `geleistet = 0,5`

#### Scenario: Gemeinsames Spiel beider Mannschaften wird geteilt
- **WHEN** ein Kind im Stammkader von Team A und von Team B steht und ein Elternteil
  einen Dienst an einem Spiel belegt, das über `game_teams` zu Team A und Team B gehört
- **THEN** zählt die Zuweisung je 0,5 für die Positionen (Kind, Team A) und
  (Kind, Team B)

#### Scenario: Geschwister-Teilung vor Kader-Teilung
- **WHEN** ein Elternteil zwei Kinder hat, Kind 1 im Stammkader von Team A und
  Team B, Kind 2 nur im Stammkader von Team A, und einen generischen Dienst belegt
- **THEN** erhält Kind 2 für (Kind 2, Team A) 0,5
- **AND** erhält Kind 1 je 0,25 für (Kind 1, Team A) und (Kind 1, Team B)

#### Scenario: Eigener Dienst bei fremder Mannschaft wird auf die Kader geteilt
- **WHEN** ein Spieler im Stammkader von Team A und Team B über seinen eigenen
  Account einen Dienst von Team C belegt, in dem er weder im Stammkader noch im
  erweiterten Kader steht
- **THEN** zählt die Zuweisung je 0,5 für (Spieler, Team A) und (Spieler, Team B)

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

#### Scenario: Trainer-Zuweisung ist keine Aushilfe
- **WHEN** ein Trainer von Team B einen Dienst von Team B belegt und sein Kind nur im erweiterten Kader von Team B steht
- **THEN** zählt die Zuweisung nicht als Aushilfe des Kindes

#### Scenario: Aushilfe ändert das Soll des fremden Teams nicht
- **WHEN** ein Aushilfe-Dienst in Team B belegt wird
- **THEN** bleiben Gesamtsumme und Soll von Team B unverändert

### Requirement: Fair-Anteil je Kind

Das System SHALL für jede Position (Kind, Kader) den Fair-Anteil dieses Kaders
ausweisen:

```
Fair-Anteil(Kind, Kader) = Gesamtsumme(Kader) / Anzahl Spieler im Kader
```

Der Fair-Anteil einer Position SHALL nur mit `geleistet`/`vorhersage` derselben
Position verglichen werden. Ein Kind im Stammkader mehrerer Mannschaften zählt in
jedem dieser Kader als voller Spieler und trägt in jedem den vollen Fair-Anteil
dieses Kaders.

Geschwister im selben Kader werden NICHT dedupliziert — jedes Kind zieht seinen
eigenen vollen Anteil, auch wenn dieselben Eltern dahinterstehen. Ein Spieler ohne
verknüpften Elternteil (`family_links`) gilt für diese Berechnung als eigene
Familie.

#### Scenario: Zwei Geschwister im selben Kader
- **WHEN** Kader K hat 20 Spieler inkl. zweier Geschwister mit denselben Eltern,
  `Gesamtsumme(K) = 60`
- **THEN** beträgt der Fair-Anteil für JEDES der beiden Geschwister-Kinder 3
  (nicht gemeinsam 3, sondern je 3)

#### Scenario: Erwachsener Spieler ohne Elternverknüpfung
- **WHEN** ein Kader-Mitglied hat keinen Eintrag in `family_links`
- **THEN** wird für dieses Mitglied dennoch ein Fair-Anteil berechnet, und das
  Mitglied selbst (statt eines Elternteils) sieht die eigene Zeile

#### Scenario: Kind in zwei Stammkadern
- **WHEN** ein Kind im Stammkader von Team A (Fair-Anteil 3) und von Team B
  (Fair-Anteil 5) steht, und für Team A 2 Dienste, für Team B 1 Dienst geleistet
  wurden
- **THEN** zeigt die Position (Kind, Team A) 2 von 3
- **AND** zeigt die Position (Kind, Team B) 1 von 5

### Requirement: Rangliste — Sortierung und Team-Filter

Die Rangliste-Seite SHALL pro ausgewähltem Team einen eigenen Block mit je einer
Zeile pro Kind dieses Kaders zeigen, absteigend sortiert nach
`geleistet + vorhersage` der Position des Kindes **in diesem Kader**. Ist mehr als
ein Team ausgewählt (Mehrfachauswahl über den bestehenden Team-Filter), SHALL die
Seite für jedes ausgewählte Team einen separaten Block zeigen — Kinder
unterschiedlicher Kader werden NICHT in einer gemeinsamen Liste vermischt, da ihr
Fair-Anteil unterschiedlich ist. Ein Kind in mehreren Stammkadern erscheint in jedem
dieser Blöcke mit den Zahlen der jeweiligen Position.

#### Scenario: Absteigende Sortierung innerhalb eines Kaders
- **WHEN** Kader K hat drei Kinder mit `geleistet+vorhersage` = 6, 1, 3
- **THEN** erscheinen sie in der Reihenfolge 6, 3, 1

#### Scenario: Mehrfachauswahl zeigt getrennte Blöcke
- **WHEN** ein Nutzer im Team-Filter zwei Teams auswählt
- **THEN** zeigt die Seite zwei separate, jeweils für sich absteigend sortierte
  Ranglisten-Blöcke, einen pro Team

#### Scenario: Kind in zwei Kadern mit unterschiedlichen Zahlen
- **WHEN** ein Kind im Stammkader von Team A und Team B steht, für Team A 3 Dienste
  und für Team B keinen Dienst belegt hat
- **THEN** zeigt der Block von Team A für dieses Kind `geleistet+vorhersage = 3`
- **AND** zeigt der Block von Team B für dieses Kind `0` und platziert es danach
