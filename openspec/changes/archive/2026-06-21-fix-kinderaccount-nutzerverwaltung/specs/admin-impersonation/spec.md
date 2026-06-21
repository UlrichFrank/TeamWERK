## MODIFIED Requirements

### Requirement: Admin kann User impersonieren
Ein eingeloggter Admin SHALL in der Nutzerverwaltung einen beliebigen Standard-User auswählen und dessen Session-Sicht übernehmen können. Der Impersonation-Endpoint gibt ein kurzlebiges JWT mit den Claims des Ziel-Users zurück. Das Impersonieren eines anderen Admins ist nicht erlaubt. Die Identität des Ziel-Users MUSS NULL-sicher aufgelöst werden — für Konten ohne E-Mail (`email IS NULL`, z. B. Kinder-Accounts) dient der `login_name` als Identitäts-Claim (`COALESCE(NULLIF(email,''), login_name, '')`, konsistent mit Login/Refresh). Ein fehlender E-Mail-Wert DARF NICHT zu einem Scan-Fehler oder HTTP 404 führen.

#### Scenario: Impersonation starten
- **WHEN** Admin klickt "Testen als" bei einem Standard-User in der Nutzerverwaltung
- **THEN** sendet das Frontend `POST /api/impersonate/{userId}`
- **THEN** gibt das Backend ein gültiges JWT mit role, club_functions und is_parent des Ziel-Users zurück
- **THEN** aktualisiert das Frontend den AuthContext mit dem neuen Token und dem impersonating-State

#### Scenario: Impersonation eines Kinder-Kontos ohne E-Mail
- **WHEN** Admin sendet `POST /api/impersonate/{userId}` für ein aktiviertes Kinder-Konto (`email IS NULL`, `login_name='Lena.Schmidt'`, `can_login=1`, role=standard)
- **THEN** antwortet das Backend mit HTTP 200 und einem gültigen JWT
- **THEN** ist der Identitäts-Claim des JWT der `login_name` (`"Lena.Schmidt"`), nicht leer

#### Scenario: Impersonation eines Admins wird abgelehnt
- **WHEN** Admin sendet `POST /api/impersonate/{userId}` für einen User mit role=admin
- **THEN** antwortet das Backend mit HTTP 400

#### Scenario: Selbst-Impersonation wird abgelehnt
- **WHEN** Admin sendet `POST /api/impersonate/{userId}` mit der eigenen userId
- **THEN** antwortet das Backend mit HTTP 400
