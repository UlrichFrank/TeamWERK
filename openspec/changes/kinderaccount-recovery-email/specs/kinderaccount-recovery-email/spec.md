## ADDED Requirements

### Requirement: Persistente Wiederherstellungs-E-Mail auf dem Konto
Das System SHALL eine Spalte `users.recovery_email TEXT` (nullable) führen, die die Eltern-/Wiederherstellungsadresse eines Kontos trägt. Diese Spalte SHALL **kein** Unique-Index erhalten und SHALL **niemals** als Lookup-Key für Login oder Passwort-Zurücksetzung verwendet werden — sie ist ausschließlich Ziel-Adresse für Korrespondenz.

#### Scenario: Geschwister teilen dieselbe Adresse
- **WHEN** zwei Kinderkonten dieselbe `recovery_email` erhalten
- **THEN** entsteht kein Unique-Konflikt (kein Index auf der Spalte)

#### Scenario: Eltern-Adresse ist kein Login-Key
- **WHEN** ein Kind eine `recovery_email` hat, die gleich der `email` eines Eltern-Accounts ist
- **THEN** matcht ein Login mit dieser Adresse ausschließlich den Eltern-Account, nie das Kind

### Requirement: Approval persistiert die Eltern-E-Mail
Das System SHALL beim Genehmigen eines Kind-Beitrittsantrags die `parent_email` des Antrags als `recovery_email` auf dem neu angelegten Konto speichern.

#### Scenario: Kind-Approval setzt recovery_email
- **WHEN** ein Vorstand `POST /api/auth/approve-membership-request/{id}` für einen Antrag mit `is_child=1` und gesetzter `parent_email` aufruft
- **THEN** hat das neu angelegte `users`-Konto `recovery_email = parent_email`
- **AND** der Passwort-Setup-Link wird wie bisher an diese Adresse versendet

### Requirement: Passwort-Vergessen über Nutzername mit Versand an die Wiederherstellungs-E-Mail
Das System SHALL `POST /api/auth/forgot-password` so erweitern, dass der Identifier sowohl `email` als auch `login_name` sein kann (`WHERE (LOWER(email)=LOWER(?) OR LOWER(login_name)=LOWER(?)) AND can_login=1`). Die Reset-Mail SHALL an `COALESCE(NULLIF(email,''), recovery_email)` gesendet werden. Die Antwort SHALL immer HTTP 204 sein (keine Enumeration).

#### Scenario: Kind setzt Passwort über login_name zurück
- **WHEN** für ein Kind (`email IS NULL`, `login_name` gesetzt, `recovery_email` gesetzt, `can_login=1`) `POST /api/auth/forgot-password` mit dem `login_name` aufgerufen wird
- **THEN** wird ein `password_reset_tokens`-Eintrag angelegt
- **AND** die Reset-Mail geht an die `recovery_email`
- **AND** der Server antwortet mit HTTP 204

#### Scenario: Erwachsener unverändert
- **WHEN** ein Erwachsener mit gesetzter `email` `POST /api/auth/forgot-password` mit seiner E-Mail aufruft
- **THEN** geht die Reset-Mail wie bisher an `email`

#### Scenario: Eltern-Adresse trifft nicht das Kind
- **WHEN** jemand `forgot-password` mit der Eltern-E-Mail aufruft, die als `recovery_email` eines Kindes hinterlegt ist
- **THEN** wird höchstens der Eltern-Account (über dessen `email`) getroffen, nie das Kind (kein Token fürs Kind)

#### Scenario: Unbekannter Identifier
- **WHEN** `forgot-password` mit einer unbekannten E-Mail/Nutzername aufgerufen wird
- **THEN** antwortet der Server mit HTTP 204 und legt keinen Token an

### Requirement: Frontend zeigt „E-Mail oder Nutzername"
Das Frontend SHALL auf `/passwort-vergessen` das Eingabefeld mit Label und Placeholder `„E-Mail oder Nutzername"` anbieten und SHALL keine reine `type=email`-Validierung erzwingen (ein `login_name` ist keine E-Mail).

#### Scenario: Nutzername eingebbar
- **WHEN** ein Nutzer auf `/passwort-vergessen` einen `login_name` (z.B. „Lena.Schmidt") eingibt und absendet
- **THEN** wird der Wert ohne E-Mail-Format-Fehler an `POST /api/auth/forgot-password` gesendet

### Requirement: Änderung der Wiederherstellungs-E-Mail durch Eltern mit doppelter Bestätigung
Das System SHALL verknüpften Eltern erlauben, die `recovery_email` eines Kindkontos zu ändern (`POST /api/profile/kind/{memberId}/recovery-email` mit `{ "new_email": "..." }`). Die Änderung SHALL erst nach **zwei** Bestätigungen wirksam werden: zuerst an der **aktuellen (alten)** Adresse (Autorisierung), danach an der **neuen** Adresse (Erreichbarkeit). Die ausstehende Änderung SHALL in `email_change_tokens` mit `field='recovery_email'` und `stage` gespeichert werden.

#### Scenario: Eltern stoßen Änderung an — Mail an alte Adresse
- **WHEN** ein verknüpftes Elternteil `POST /api/profile/kind/{memberId}/recovery-email` mit `new_email` aufruft
- **THEN** wird ein Token mit `field='recovery_email'`, `stage='auth'`, `new_email=...` gespeichert
- **AND** eine Bestätigungsmail geht an die **aktuelle** `recovery_email` des Kindes
- **AND** an die neue Adresse wird (noch) nichts gesendet

#### Scenario: Kein verknüpftes Elternteil
- **WHEN** ein Nutzer ohne `family_links`-Beziehung zum Kind den Endpoint aufruft
- **THEN** antwortet der Server mit HTTP 403 und legt keinen Token an

#### Scenario: Stufe 1 (alte Adresse) bestätigt → Stufe 2 ausgelöst
- **WHEN** der Link aus der Mail an die alte Adresse aufgerufen wird (`GET /api/profile/recovery-email/confirm?token=...`, `stage='auth'`, gültig)
- **THEN** wird der Token auf `stage='verify'` überführt (Token rotiert, alter Token verbraucht)
- **AND** eine Bestätigungsmail geht an die **neue** Adresse
- **AND** `users.recovery_email` ist noch **unverändert**

#### Scenario: Stufe 2 (neue Adresse) bestätigt → Änderung wirksam
- **WHEN** der Link aus der Mail an die neue Adresse aufgerufen wird (`stage='verify'`, gültig)
- **THEN** wird `users.recovery_email` auf die neue Adresse gesetzt
- **AND** der Token wird mit `used_at` markiert
- **AND** der Server leitet zu einer Bestätigungsseite / `/login` weiter

#### Scenario: Abgelaufener oder unbekannter Token
- **WHEN** `GET /api/profile/recovery-email/confirm?token=...` mit abgelaufenem/unbekanntem Token aufgerufen wird
- **THEN** wird nichts geschrieben und zu `…?error=invalid_token` weitergeleitet

### Requirement: Admin/Vorstand setzen die Wiederherstellungs-E-Mail direkt
Das System SHALL Caller mit Funktion `admin` oder `vorstand` erlauben, `users.recovery_email` direkt zu setzen (`PUT /api/users/{id}/recovery-email` mit `{ "recovery_email": "..." }`) — **ohne** Bestätigungs-Workflow. Dies ist der Escape-Hatch, wenn die alte Adresse nicht mehr existiert und der Bestätigungs-Loop deshalb nicht abschließbar ist.

#### Scenario: Vorstand setzt Adresse direkt
- **WHEN** ein Caller mit `vorstand`- oder `admin`-Funktion `PUT /api/users/{id}/recovery-email` aufruft
- **THEN** wird `users.recovery_email` sofort gesetzt
- **AND** es wird kein Token erzeugt und keine Bestätigungsmail versendet
- **AND** der Server antwortet mit HTTP 204

#### Scenario: Ohne Funktion verweigert
- **WHEN** ein Caller ohne `admin`/`vorstand`-Funktion den Endpoint aufruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Kind kann die Wiederherstellungs-E-Mail nicht ändern, aber lesen
Das System SHALL die `recovery_email` im Self-Edit des Kindes (`PUT /api/profile/account`) **nicht** als beschreibbares Feld exponieren; ein im Body mitgeschicktes `recovery_email` SHALL ignoriert werden. Die `recovery_email` SHALL im eigenen Profil des Kindes und im eingeblendeten Kindprofil der Eltern **lesbar** sein.

#### Scenario: Kind versucht Selbst-Änderung
- **WHEN** ein eingeloggtes Kind `PUT /api/profile/account` mit einem `recovery_email`-Feld aufruft
- **THEN** bleibt `users.recovery_email` unverändert

#### Scenario: Lesbar im eingeblendeten Kindprofil
- **WHEN** ein verknüpftes Elternteil `GET /api/profile/kind/{memberId}` aufruft
- **THEN** enthält die Antwort die `recovery_email` des Kindkontos

#### Scenario: Lesbar im eigenen Profil des Kindes
- **WHEN** ein Kind sein eigenes Profil lädt
- **THEN** wird die `recovery_email` read-only angezeigt
