## MODIFIED Requirements

### Requirement: Teamübergreifender Kontakt im Zugriffskreis

Das System SHALL zwei Mitgliedern des **Zugriffskreises** erlauben, sich gegenseitig zu kontaktieren — sowohl per Direktnachricht (`POST /api/chat/conversations` mit `type=direct`) als auch als Teilnehmer beim Gruppenaufbau (`type=group`) — auch ohne gemeinsames Team. Der Zugriffskreis ist definiert als: User, die (a) Trainer eines Kaders der aktiven Saison sind (`kader_trainers`), ODER Vereinsfunktion (b) `vorstand`, (c) `sportliche_leitung` ODER (d) `vorstand_beisitzer` haben; `admin` stets berechtigt.

Die Kontaktprüfung (`canContactUser`) SHALL in dieser Reihenfolge auswerten: (1) Caller hat **vereinsweite Reichweite** — Rolle `admin` ODER Vereinsfunktion `vorstand` ODER `sportliche_leitung` → erlaubt; (2) Caller UND Ziel sind beide im Zugriffskreis → erlaubt; (3) Caller und Ziel teilen ein Team (`user_accessible_teams`) → erlaubt; (4) sonst HTTP 403. Die Regeln (2) und (3) bleiben unverändert.

Die Menge aus Schritt (1) ist dieselbe, die in `chat-team-groups` die Standardgruppen **aller** Teams der aktiven Saison sieht. Beides MUSS zusammenfallen: wer eine fremde Kader-Gruppe auflösen darf, muss die aufgelösten Mitglieder auch anschreiben dürfen — sonst scheitert der Gruppenaufbau an Schritt (4) für jedes teamfremde Mitglied, obwohl die Gruppe im Modal angeboten wurde.

#### Scenario: Trainer schreibt teamfremden Trainer 1:1 an

- **WHEN** ein Kader-Trainer von T1 `POST /api/chat/conversations` mit `{ type: "direct", userId: <Trainer von T2> }` aufruft und kein gemeinsames Team besteht
- **THEN** wird die Direktkonversation erstellt (HTTP 201/200)

#### Scenario: Sportliche Leitung schreibt teamfremden Trainer an

- **WHEN** ein User mit `sportliche_leitung` einen Trainer eines Teams, in dem er nicht eingetragen ist, per Direktnachricht kontaktiert
- **THEN** wird die Konversation erstellt

#### Scenario: Sportliche Leitung legt Gruppe aus fremder Kader-Gruppe an

- **WHEN** ein User mit `sportliche_leitung` ohne eigene Teamzugehörigkeit `GET /api/chat/team-groups/{fremdesTeam}/spieler/members` auflöst und die zurückgegebenen IDs als `memberIds` an `POST /api/chat/conversations` mit `type=group` schickt
- **THEN** passieren alle Mitglieder die `canContactUser`-Prüfung und die Gruppe wird erstellt (HTTP 201)

#### Scenario: „Alle Trainer"-Gruppe anlegen ist erlaubt

- **WHEN** ein Zugriffskreis-Mitglied `POST /api/chat/conversations` mit `type=group` und den aus „Alle Trainer" aufgelösten Mitgliedern aufruft
- **THEN** passieren alle Mitglieder die `canContactUser`-Prüfung und die Gruppe wird erstellt

#### Scenario: Spieler kann teamfremden Trainer nicht kontaktieren

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-Zugehörigkeit einen Trainer eines fremden Teams per Direktnachricht kontaktieren will
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Spieler kann teamfremden Spieler nicht in eine Gruppe nehmen

- **WHEN** ein Spieler von T1 `POST /api/chat/conversations` mit `type=group` und `memberIds=[<Spieler von T2>]` aufruft
- **THEN** antwortet der Server mit HTTP 403 und es entsteht keine Konversation

### Requirement: Nutzersuche findet Zugriffskreis teamübergreifend

Das System SHALL in `GET /api/chat/users` einem Caller, der im Zugriffskreis ist, zusätzlich zu Usern mit gemeinsamem Team **alle anderen Zugriffskreis-Mitglieder** als Suchtreffer liefern (Dedup nach `user_id`, Namens-/E-Mail-Filter `q` und `LIMIT 50` bleiben bestehen). Für Caller mit **vereinsweiter Reichweite** (`admin`, `vorstand`, `sportliche_leitung`) sucht der Endpoint über alle User; für Caller außerhalb des Zugriffskreises bleibt die Suche auf gemeinsame Teams beschränkt.

Die Reichweite der Suche MUSS mit Schritt (1) der Kontaktprüfung übereinstimmen — sonst erscheint ein teamfremdes Mitglied als Gruppen-Chip, ist über die Suche daneben aber nicht auffindbar.

#### Scenario: Trainer findet teamfremden Trainer

- **WHEN** ein Kader-Trainer von T1 `GET /api/chat/users?q=<Name eines Trainers von T2>` aufruft
- **THEN** enthält das Ergebnis den Trainer von T2, obwohl kein gemeinsames Team besteht

#### Scenario: Sportliche Leitung findet teamfremden Spieler

- **WHEN** ein User mit `sportliche_leitung` ohne eigene Teamzugehörigkeit `GET /api/chat/users` aufruft
- **THEN** enthält das Ergebnis auch Spieler von Teams, in denen er nicht eingetragen ist

#### Scenario: Spieler findet teamfremden Trainer nicht

- **WHEN** ein Spieler ohne Trainer-/Vorstand-/sL-Zugehörigkeit nach einem Trainer eines fremden Teams sucht
- **THEN** ist dieser nicht im Ergebnis enthalten
