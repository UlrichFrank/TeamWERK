## ADDED Requirements

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
