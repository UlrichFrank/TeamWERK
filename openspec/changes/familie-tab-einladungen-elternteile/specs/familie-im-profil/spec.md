## MODIFIED Requirements

### Requirement: Familie Tab zeigt registrierte und ausstehende Erziehungsberechtigte
Der Familie Tab auf `/mitglieder/{id}` SHALL registrierte Erziehungsberechtigte (`family_links`) und ausstehende Einladungen (`invitation_tokens` mit `parent_member_id = memberId`) in einer gemeinsamen Liste anzeigen. Die Gesamtzahl (registriert + pending) darf 2 nicht überschreiten.

#### Scenario: Nur registrierte Elternteile vorhanden
- **WHEN** der Familie Tab geöffnet wird und nur registrierte `family_links` existieren
- **THEN** werden diese als normale Einträge angezeigt (Name + E-Mail + „Entfernen"-Button)

#### Scenario: Ausstehende Einladung als Elternteil vorgemerkt
- **WHEN** der Familie Tab geöffnet wird und eine Einladung hat `parent_member_id = memberId`
- **THEN** wird die Einladung in der Liste mit E-Mail und Badge „Einladung ausstehend" angezeigt

#### Scenario: Gemischte Liste (registriert + pending)
- **WHEN** ein registrierter Elternteil und eine ausstehende Einladung für dasselbe Mitglied existieren
- **THEN** werden beide in einer gemeinsamen Liste angezeigt (max. 2 insgesamt)

#### Scenario: Maximum von 2 erreicht
- **WHEN** bereits 2 Einträge (registriert oder pending) vorhanden sind
- **THEN** wird der Dropdown zum Hinzufügen weiterer Elternteile ausgeblendet

### Requirement: Familie Tab erlaubt Verknüpfung ausstehender Einladungen
Der Familie Tab SHALL einen Dropdown mit ausstehenden Einladungen (gefiltert: noch nicht als `parent_member_id` dieses Mitglieds verknüpft) anzeigen, wenn noch Platz für weitere Elternteile vorhanden ist.

#### Scenario: Einladung als Elternteil verknüpfen
- **WHEN** ein Admin eine Einladung aus dem Dropdown auswählt und „Hinzufügen" klickt
- **THEN** wird `PUT /api/admin/invitations/{id}/parent-member` mit `{ member_id }` aufgerufen und die Liste aktualisiert

#### Scenario: Pending-Verknüpfung entfernen
- **WHEN** ein Admin bei einer ausstehenden Einladung auf „Entfernen" klickt
- **THEN** wird `PUT /api/admin/invitations/{id}/parent-member` mit `{ member_id: null }` aufgerufen und der Eintrag verschwindet aus der Liste
