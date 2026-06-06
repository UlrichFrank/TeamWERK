## ADDED Requirements

### Requirement: Einladung kann vor Registrierung mit einem Mitglied verknüpft werden
Das System SHALL es einem Admin ermöglichen, eine offene Einladung (`invitation_tokens`) mit einem bestehenden `members`-Eintrag zu verknüpfen, bevor sich der eingeladene Nutzer registriert hat. Die Verknüpfung wird in `invitation_tokens.member_id` gespeichert.

#### Scenario: Verknüpfung setzen
- **WHEN** Admin wählt „Mit Mitglied verknüpfen" im ActionMenu einer Einladungs-Zeile
- **THEN** öffnet sich ein Modal mit einer durchsuchbaren Mitgliederliste
- **WHEN** Admin wählt ein Mitglied aus und bestätigt
- **THEN** setzt das System `invitation_tokens.member_id` auf die gewählte `members.id` (204 No Content)

#### Scenario: Mitglied bereits mit einem User verknüpft
- **WHEN** das gewählte Mitglied bereits ein `user_id` gesetzt hat (`members.user_id IS NOT NULL`)
- **THEN** gibt das System 409 Conflict zurück und das Frontend zeigt eine Fehlermeldung an

#### Scenario: Verknüpfung entfernen
- **WHEN** Admin wählt „Verknüpfung aufheben" im ActionMenu einer bereits verknüpften Einladung
- **THEN** setzt das System `invitation_tokens.member_id = NULL` (204 No Content)

### Requirement: Verknüpfung wird beim Registrieren automatisch übertragen
Das System SHALL beim Registrieren eines Nutzers über einen Token prüfen, ob `invitation_tokens.member_id` gesetzt ist, und in diesem Fall automatisch `members.user_id` auf den neu erstellten `users.id` setzen.

#### Scenario: Registrierung mit Mitglied-Verknüpfung
- **WHEN** ein Nutzer sich über einen Token registriert, der `member_id IS NOT NULL` hat
- **THEN** legt das System den neuen `users`-Eintrag an
- **THEN** setzt das System `members.user_id = new_user_id` für den verknüpften `members`-Eintrag

#### Scenario: Mitglied wurde zwischenzeitlich anderweitig verknüpft
- **WHEN** ein Nutzer sich über einen Token mit `member_id` registriert, aber das Mitglied bereits `user_id IS NOT NULL` hat
- **THEN** legt das System den `users`-Eintrag trotzdem an (Registrierung gelingt)
- **THEN** wird `members.user_id` nicht überschrieben (kein Fehler, stille Nicht-Übertragung)
