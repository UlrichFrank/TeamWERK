## ADDED Requirements

### Requirement: Notification bei neuem Angebot
Wenn ein neues Bieter-Angebot eingestellt wird, SHALL das System allen Nutzern, die ein aktives Gesuch für dasselbe Spiel haben, eine Push-Benachrichtigung senden.

#### Scenario: Bieter legt Angebot an
- **WHEN** `POST /api/mitfahrgelegenheiten` mit `typ='biete'` erfolgreich ist
- **THEN** erhalten alle Nutzer mit einem offenen Gesuch (`typ='suche'`) für dasselbe Spiel eine Push-Notification mit dem Hinweis auf das neue Angebot

### Requirement: Notification bei neuer Suchanfrage
Wenn ein neues Gesuch eingestellt wird, SHALL das System allen Nutzern, die ein aktives Angebot für dasselbe Spiel haben, eine Push-Benachrichtigung senden.

#### Scenario: Sucher legt Gesuch an
- **WHEN** `POST /api/mitfahrgelegenheiten` mit `typ='suche'` erfolgreich ist
- **THEN** erhalten alle Nutzer mit einem offenen Angebot (`typ='biete'`) für dasselbe Spiel eine Push-Notification mit dem Hinweis auf das neue Gesuch

### Requirement: Notification bei zurückgezogenem Angebot
Wenn ein Bieter-Eintrag gelöscht oder deaktiviert wird, SHALL das System alle Sucher, die eine aktive Paarung mit diesem Angebot haben, per Push-Benachrichtigung informieren.

#### Scenario: Bieter zieht Angebot zurück
- **WHEN** `DELETE /api/mitfahrgelegenheiten/{id}` für einen Bieter-Eintrag aufgerufen wird
- **THEN** erhalten alle Sucher mit einer `pending`- oder `confirmed`-Paarung zu diesem Angebot eine Push-Notification über den Rückzug

### Requirement: Push-Versand ist nicht blockierend
Der Push-Versand MUST asynchron und fehlertolerant erfolgen, sodass ein fehlgeschlagener Push-Aufruf die eigentliche API-Antwort nicht verzögert oder verhindert.

#### Scenario: Push-Fehler blockiert nicht
- **WHEN** der Push-Dienst nicht erreichbar ist oder ein Endpoint ungültig ist
- **THEN** schlägt die ursprüngliche API-Operation nicht fehl; der Fehler wird lediglich geloggt

### Requirement: Team-Push bei neuer Suche zum nächsten Spiel
Wenn ein User eine neue Suche (`typ='suche'`, kein Update) zu einem Spiel anlegt, das für mindestens eines der assoziierten Teams das nächste anstehende Spiel ist, SHALL das System einen Push an Eltern der Kaderspieler (regulär und erweitert) sowie an die Trainer der qualifizierenden Kader-Zeile(n) senden. Der Steller selbst MUST aus dem Empfängerkreis ausgeschlossen werden.

#### Scenario: Suche zum nächsten Spiel löst Team-Push aus
- **GIVEN** ein Spiel G mit Team T, das in `game_teams` für T das früheste Spiel mit `date >= date('now')` ist
- **AND** ein Kader K mit `K.team_id = T` und `K.season_id = G.season_id`
- **WHEN** `POST /api/mitfahrgelegenheiten` mit `typ='suche'` und `gameId=G.id` einen neuen Eintrag erzeugt
- **THEN** erhalten alle `parent_user_id` aus `family_links` über `kader_members ∪ kader_extended_members` von K sowie die `user_id` der `kader_trainers` von K einen Push der Kategorie `"carpooling"` mit Titel "Mitfahrgelegenheit" und Body "{Name} sucht eine Mitfahrgelegenheit zu {opponent}, {Datum}"
- **AND** die `user_id` des Stellers ist nicht in der Empfängerliste enthalten

#### Scenario: Suche zu späterem Spiel löst keinen Team-Push aus
- **GIVEN** ein Spiel G mit Team T, für das mindestens ein weiteres Spiel G' mit `G'.date < G.date` und `G'.date >= date('now')` existiert
- **WHEN** `POST /api/mitfahrgelegenheiten` mit `typ='suche'` und `gameId=G.id` einen neuen Eintrag erzeugt
- **THEN** wird kein Team-Push versendet; nur der bestehende `notifyOpposite`-Push an `biete`-User des Spiels läuft

#### Scenario: Update einer Suche löst keinen Team-Push aus
- **GIVEN** eine bereits existierende Suche desselben Users zum selben Spiel
- **WHEN** `POST /api/mitfahrgelegenheiten` denselben Eintrag aktualisiert (Notiz, Treffpunkt, Plätze)
- **THEN** wird kein Team-Push versendet

#### Scenario: Fehlender Kader führt zu stillem Skip
- **GIVEN** ein Spiel G mit Team T, für das **keine** `kader`-Zeile mit `(T, G.season_id)` existiert
- **WHEN** `POST /api/mitfahrgelegenheiten` mit `typ='suche'` und `gameId=G.id` einen neuen Eintrag erzeugt
- **THEN** wird kein Team-Push versendet, und die HTTP-Antwort bleibt `204 No Content`

#### Scenario: Multi-Team-Spiel — nur qualifizierende Teams tragen bei
- **GIVEN** ein Spiel G mit `game_teams` `{A, B}`, das für A das nächste anstehende Spiel ist, für B aber nicht
- **WHEN** `POST /api/mitfahrgelegenheiten` mit `typ='suche'` und `gameId=G.id` einen neuen Eintrag erzeugt
- **THEN** umfasst der Empfängerkreis ausschließlich Eltern/Trainer des Kaders von Team A; B's Kader trägt nicht bei

#### Scenario: Push-Präferenz wird respektiert
- **GIVEN** ein Empfänger mit `notification_preferences` für Kategorie `"carpooling"` und `push_enabled=0`
- **WHEN** ein Team-Push ausgelöst würde
- **THEN** erhält dieser User keinen Push (Standard-Filterpfad über `notify.Send`)
