## ADDED Requirements

### Requirement: Self-Service-Geburtsdatum auf users

Das System SHALL eine Spalte `users.date_of_birth` (nullable) bereitstellen und sie über `PUT /api/profile/me` sofort aktualisieren, sobald das Feld `date_of_birth` im Request-Body enthalten ist — analog zu `street`/`zip`/`city`. Ohne Vorstands-Freigabe.

#### Scenario: Geburtsdatum erfolgreich gesetzt
- **WHEN** ein eingeloggter Nutzer `PUT /api/profile/me` mit `{ "date_of_birth": "1990-05-12", ... }` aufruft
- **THEN** wird `users.date_of_birth` für den aufrufenden Nutzer sofort auf `1990-05-12` aktualisiert und HTTP 204 zurückgegeben

#### Scenario: Leeres Geburtsdatum löscht den Self-Service-Wert
- **WHEN** `PUT /api/profile/me` mit `{ "date_of_birth": "" }` aufgerufen wird
- **THEN** wird `users.date_of_birth` auf `NULL` gesetzt

#### Scenario: Nicht eingeloggte Anfrage wird abgelehnt
- **WHEN** `PUT /api/profile/me` ohne gültigen Bearer-Token aufgerufen wird
- **THEN** antwortet das System mit HTTP 401

### Requirement: GET /api/profile/me liefert Geburtsdatum

`GET /api/profile/me` SHALL das Feld `date_of_birth` aus `users` zurückgeben (leer, wenn nie gesetzt), zusätzlich zum bereits enthaltenen `own_member.date_of_birth` aus dem Mitglieder-Record.

#### Scenario: Self-Service-Wert vorhanden
- **WHEN** `GET /api/profile/me` aufgerufen wird und `users.date_of_birth = '1990-05-12'`
- **THEN** enthält der Response `{ "date_of_birth": "1990-05-12", "own_member": { "date_of_birth": "...", ... } }`

#### Scenario: Self-Service-Wert noch nie gesetzt
- **WHEN** `GET /api/profile/me` aufgerufen wird und `users.date_of_birth IS NULL`
- **THEN** enthält der Response `date_of_birth: ""` (oder Feld fehlt), während `own_member.date_of_birth` weiterhin das Mitglieder-Record-Datum enthält

### Requirement: Kontakt-Tab zeigt Geburtsdatum-Feld für direkt verknüpfte Accounts

Das Frontend SHALL im Kontakt-Tab (`ProfileProfilTab`) unterhalb des PLZ/Ort-Feldes ein Eingabefeld „Geburtsdatum" (`<input type="date">`) anzeigen, wenn ein direkt verknüpftes Mitgliederkonto vorliegt: im eigenen Profil, wenn `ownMember !== null`; im Kind-Profil zusätzlich nur, wenn das Kind einen eigenen User-Account hat (`userContact !== null`).

#### Scenario: Eigenes Profil mit verknüpftem Mitglied
- **WHEN** ein Nutzer mit `ownMember !== null` den Kontakt-Tab öffnet
- **THEN** erscheint das Geburtsdatum-Feld unter PLZ/Ort

#### Scenario: Eigenes Profil ohne verknüpftes Mitglied
- **WHEN** ein Nutzer ohne eigenes Mitglieder-Record (`ownMember === null`, z.B. reiner Elternteil-Account) den Kontakt-Tab öffnet
- **THEN** erscheint kein Geburtsdatum-Feld

#### Scenario: Kind-Profil mit eigenem Account
- **WHEN** ein Elternteil das Kontakt-Tab-Profil eines Kindes mit eigenem User-Account (`userContact !== null`) öffnet
- **THEN** erscheint das Geburtsdatum-Feld unter PLZ/Ort

#### Scenario: Kind-Profil ohne eigenen Account
- **WHEN** ein Elternteil das Kontakt-Tab-Profil eines Kindes ohne eigenen User-Account (`userContact === null`) öffnet
- **THEN** erscheint kein Geburtsdatum-Feld

### Requirement: Default aus dem Mitgliederbereich, solange nicht explizit gesetzt

Solange der Self-Service-Wert (`users.date_of_birth` bzw. `user_contact.date_of_birth`) leer ist, SHALL das Geburtsdatum-Feld beim Laden mit dem aktuellen Geburtsdatum aus dem Mitgliederbereich (`ownMember.date_of_birth` bzw. `member.date_of_birth`) vorbelegt werden. Sobald der Self-Service-Wert einmal explizit gespeichert wurde, SHALL dieser Wert angezeigt werden, unabhängig von späteren Änderungen des Mitglieder-Records.

#### Scenario: Noch nie explizit gesetzt — Vorbelegung aus Mitgliederbereich
- **WHEN** der Kontakt-Tab geladen wird, `users.date_of_birth` ist leer und `ownMember.date_of_birth = '2008-04-15'`
- **THEN** zeigt das Geburtsdatum-Feld initial `2008-04-15`

#### Scenario: Bereits explizit gesetzt — eigener Wert hat Vorrang
- **WHEN** der Kontakt-Tab geladen wird, `users.date_of_birth = '2008-04-16'` und `ownMember.date_of_birth = '2008-04-15'` (z.B. abweichend, weil ein Antrag noch aussteht)
- **THEN** zeigt das Geburtsdatum-Feld `2008-04-16`, nicht den Mitglieder-Record-Wert

### Requirement: Speichern legt Änderungsantrag ans Mitglieder-Record an

Das Speichern im Kontakt-Tab SHALL das Geburtsdatum zusätzlich zum Self-Service-Write als Teil des bestehenden `profil`-Änderungsantrags (`member_change_drafts`, `field_name='profil'`) einreichen — genau wie Vorname, Nachname und Adresse. Der Server SHALL `date_of_birth` in `extractFieldValue`/`applyDraftToMember` für `field_name='profil'` berücksichtigen (lesen bzw. schreiben von `members.date_of_birth`).

#### Scenario: Speichern erzeugt Draft mit Geburtsdatum
- **WHEN** ein Nutzer im Kontakt-Tab das Geburtsdatum ändert und speichert
- **THEN** wird `PUT /api/profile/me` (sofortiger Self-Service-Write) **und** `POST /api/members/:id/change-request` mit `field_name=profil` und `new_value.date_of_birth` aufgerufen

#### Scenario: Vorstand nimmt Draft an
- **WHEN** der Vorstand einen `profil`-Draft mit geändertem `date_of_birth` annimmt
- **THEN** wird `members.date_of_birth` auf den neuen Wert aktualisiert

#### Scenario: Ausstehende Anfrage zeigt Geburtsdatum-Diff
- **WHEN** ein `profil`-Draft mit abweichendem `date_of_birth` (alt/neu) existiert und der Mitgliedsdaten-Tab die „Ausstehende Anfrage" anzeigt
- **THEN** erscheint eine Zeile „Geburtsdatum: <alt> → <neu>" in der Diff-Liste

### Requirement: Keine zusätzliche Formatvalidierung

Das System SHALL das Geburtsdatum-Feld ohne serverseitige Mindestalter- oder Plausibilitätsprüfung akzeptieren — konsistent mit dem bestehenden Adress-Verhalten. Die einzige Validierung ist die native Kalenderauswahl von `<input type="date">`.

#### Scenario: Zukunftsdatum wird akzeptiert
- **WHEN** ein Nutzer ein Geburtsdatum in der Zukunft einträgt und speichert
- **THEN** wird der Wert ohne Fehlermeldung übernommen (kein HTTP 400 wegen Plausibilität)
