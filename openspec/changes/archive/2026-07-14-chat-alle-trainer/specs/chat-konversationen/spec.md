## ADDED Requirements

### Requirement: Teamübergreifender Kontakt im Zugriffskreis

Das System SHALL zwei Mitgliedern des **Zugriffskreises** erlauben, sich gegenseitig zu kontaktieren — sowohl per Direktnachricht (`POST /api/chat/conversations` mit `type=direct`) als auch als Teilnehmer beim Gruppenaufbau (`type=group`) — auch ohne gemeinsames Team. Der Zugriffskreis ist definiert als: User, die (a) Trainer eines Kaders der aktiven Saison sind (`kader_trainers`), ODER Vereinsfunktion (b) `vorstand`, (c) `sportliche_leitung` ODER (d) `vorstand_beisitzer` haben; `admin` stets berechtigt.

Die Kontaktprüfung (`canContactUser`) SHALL in dieser Reihenfolge auswerten: (1) Caller ist `admin` oder `vorstand` → erlaubt; (2) Caller UND Ziel sind beide im Zugriffskreis → erlaubt; (3) Caller und Ziel teilen ein Team (`user_accessible_teams`) → erlaubt; (4) sonst HTTP 403. Die bestehenden Regeln (1) und (3) bleiben unverändert.

#### Scenario: Trainer schreibt teamfremden Trainer 1:1 an

- **WHEN** ein Kader-Trainer von T1 `POST /api/chat/conversations` mit `{ type: "direct", userId: <Trainer von T2> }` aufruft und kein gemeinsames Team besteht
- **THEN** wird die Direktkonversation erstellt (HTTP 201/200)

#### Scenario: Sportliche Leitung schreibt teamfremden Trainer an

- **WHEN** ein User mit `sportliche_leitung` einen Trainer eines Teams, in dem er nicht eingetragen ist, per Direktnachricht kontaktiert
- **THEN** wird die Konversation erstellt

#### Scenario: „Alle Trainer"-Gruppe anlegen ist erlaubt

- **WHEN** ein Zugriffskreis-Mitglied `POST /api/chat/conversations` mit `type=group` und den aus „Alle Trainer" aufgelösten Mitgliedern aufruft
- **THEN** passieren alle Mitglieder die `canContactUser`-Prüfung und die Gruppe wird erstellt

#### Scenario: Spieler kann teamfremden Trainer nicht kontaktieren

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-Zugehörigkeit einen Trainer eines fremden Teams per Direktnachricht kontaktieren will
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Nutzersuche findet Zugriffskreis teamübergreifend

Das System SHALL in `GET /api/chat/users` einem Caller, der im Zugriffskreis ist, zusätzlich zu Usern mit gemeinsamem Team **alle anderen Zugriffskreis-Mitglieder** als Suchtreffer liefern (Dedup nach `user_id`, Namens-/E-Mail-Filter `q` und `LIMIT 50` bleiben bestehen). Für `admin`/`vorstand` bleibt die Suche über alle User unverändert; für Caller außerhalb des Zugriffskreises bleibt die Suche auf gemeinsame Teams beschränkt.

#### Scenario: Trainer findet teamfremden Trainer

- **WHEN** ein Kader-Trainer von T1 `GET /api/chat/users?q=<Name eines Trainers von T2>` aufruft
- **THEN** enthält das Ergebnis den Trainer von T2, obwohl kein gemeinsames Team besteht

#### Scenario: Spieler findet teamfremden Trainer nicht

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-Zugehörigkeit nach einem Trainer eines fremden Teams sucht
- **THEN** ist dieser nicht im Ergebnis enthalten
