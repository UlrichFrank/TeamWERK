## REMOVED Requirements

### Requirement: Aufstellungsstatus im Spiel-Event des erweiterten Kaders

**Reason**: Der Aufstellungsstatus gilt künftig für alle Spieler des Kaders, nicht nur
für den erweiterten Kader; das Szenario „Regulärer Kader bekommt keinen Status" kehrt
sich um.
**Migration**: Ersetzt durch das Requirement „Aufstellungsstatus im Spiel-Event"
(gleiche drei Zustände, gleiche Kennwörter und Sätze, zusätzlich für `kader_members`).
Abonnierte Kalender übernehmen die neuen Titel beim nächsten Abruf; kein Handgriff nötig.

## ADDED Requirements

### Requirement: Aufstellungsstatus im Spiel-Event

Für ein Spiel-Event vom Typ `heim` oder `auswärts`, an dem der Feed-Nutzer als Spieler
hängt — über den regulären Kader (`kader_members`) **oder** den erweiterten Kader
(`kader_extended_members`) —, SHALL der Feed den Aufstellungsstatus dieses Nutzers
ausweisen: in der Mannschafts-Klammer des `SUMMARY` als kurzes Kennwort und im
`DESCRIPTION` als vollständiger Satz.

Der Status SHALL die drei Zustände der Capability `spiel-aufstellung` kennen, mit
denselben Bezeichnungen wie in der Oberfläche:

| Zustand | Bedingung | Kennwort im `SUMMARY` | Satz im `DESCRIPTION` |
|---|---|---|---|
| aufgestellt | eine Zeile für dieses Spiel **und** dieses Mitglied | `aufgestellt` | „Du bist für das Spiel aufgestellt." |
| nicht aufgestellt | mindestens eine Zeile für dieses Spiel, aber keine für dieses Mitglied | `nicht aufgestellt` | „Du bist für das Spiel NICHT aufgestellt. Bitte mit der Trainerin/dem Trainer absprechen, ob eine Anwesenheit trotzdem erwünscht ist." |
| offen | **keine** Zeile für dieses Spiel | `Aufstellung offen` | „Die Aufstellung für dieses Spiel steht noch nicht fest." |

Der Zustand „offen" SHALL eigenständig bleiben: aus einer leeren Aufstellung SHALL das
System **nicht** „nicht aufgestellt" ableiten. Eine nicht gepflegte Aufstellung ist keine
Nichtberücksichtigung, und der Feed SHALL dem Empfänger keine Absage melden, die niemand
ausgesprochen hat.

Der Satz SHALL an eine vorhandene Notiz des Termins angehängt werden, getrennt durch eine
Leerzeile; die Notiz SHALL dabei erhalten bleiben. Hat der Termin keine Notiz, SHALL das
`DESCRIPTION` allein den Satz tragen.

Der Zusatz `erw. Kader` in der Mannschafts-Klammer SHALL weiterhin nur bei Zugehörigkeit
ausschließlich über den erweiterten Kader erscheinen; das Status-Kennwort steht in
beiden Fällen als letztes Element der Klammer. Nutzer, die nur als Trainer
(`kader_trainers`) am Spiel hängen, SHALL der Feed **ohne** Status ausweisen. Events vom
Typ `generisch` SHALL keinen Status tragen; sie haben keine Aufstellung.

#### Scenario: Aufgestellter Spieler des erweiterten Kaders

- **WHEN** ein Nutzer über `kader_extended_members` am Kader von `mB1` hängt, für das Heimspiel gegen `SG Weinstadt` eine Aufstellung gespeichert ist und sein Mitglied darin steht
- **THEN** lautet das `SUMMARY` `Heim: Team (mB1 · erw. Kader · aufgestellt) – SG Weinstadt`
- **AND** enthält das `DESCRIPTION` den Satz `Du bist für das Spiel aufgestellt.`

#### Scenario: Nicht aufgestellter Spieler des erweiterten Kaders

- **WHEN** für dasselbe Spiel eine Aufstellung gespeichert ist, das Mitglied des Nutzers aber nicht darin steht
- **THEN** trägt das `SUMMARY` in der Mannschafts-Klammer den Zusatz `· nicht aufgestellt`
- **AND** enthält das `DESCRIPTION` den Satz `Du bist für das Spiel NICHT aufgestellt. Bitte mit der Trainerin/dem Trainer absprechen, ob eine Anwesenheit trotzdem erwünscht ist.`

#### Scenario: Aufstellung noch nicht gespeichert

- **WHEN** für das Spiel **keine** Zeile in `game_lineup` existiert
- **THEN** trägt das `SUMMARY` in der Mannschafts-Klammer den Zusatz `· Aufstellung offen`
- **AND** enthält das `DESCRIPTION` den Satz `Die Aufstellung für dieses Spiel steht noch nicht fest.`
- **AND** enthält der Feed an keiner Stelle die Aussage `NICHT aufgestellt`

#### Scenario: Notiz und Aufstellungssatz stehen beide im DESCRIPTION

- **WHEN** das Spiel eine Notiz trägt und der Nutzer als Spieler daran hängt
- **THEN** enthält das `DESCRIPTION` zuerst die Notiz, dann eine Leerzeile, dann den Aufstellungssatz

#### Scenario: Stammspieler bekommt den Status

- **WHEN** ein Nutzer über `kader_members` am Kader von `mB1` hängt und in der gespeicherten Aufstellung des Heimspiels gegen `SG Weinstadt` steht
- **THEN** lautet das `SUMMARY` `Heim: Team (mB1 · aufgestellt) – SG Weinstadt` ohne Kader-Zusatz
- **AND** enthält das `DESCRIPTION` den Satz `Du bist für das Spiel aufgestellt.`

#### Scenario: Stammspieler bei offener Aufstellung

- **WHEN** ein Nutzer über `kader_members` am Kader hängt und für das Spiel keine Aufstellung gespeichert ist
- **THEN** lautet das `SUMMARY` `Heim: Team (mB1 · Aufstellung offen) – SG Weinstadt`

#### Scenario: Doppelte Zugehörigkeit — regulär schlägt erweitert

- **WHEN** ein Nutzer an demselben Spiel sowohl über `kader_members` als auch über `kader_extended_members` hängt
- **THEN** enthält der Feed genau ein VEVENT für dieses Spiel
- **AND** trägt es keinen Kader-Zusatz, wohl aber das Status-Kennwort

#### Scenario: Trainer bekommt keinen Status

- **WHEN** ein Nutzer nur über `kader_trainers` am Kader des Teams hängt
- **THEN** trägt das `SUMMARY` kein Status-Kennwort und das `DESCRIPTION` keinen Aufstellungssatz

#### Scenario: Generisches Event trägt keinen Status

- **WHEN** ein Event vom Typ `generisch` im Feed erscheint und der Nutzer als Spieler daran hängt
- **THEN** bleibt das `SUMMARY` der Terminname ohne Status-Zusatz
- **AND** enthält das `DESCRIPTION` keinen Aufstellungssatz
