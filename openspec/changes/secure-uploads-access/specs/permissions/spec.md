## MODIFIED Requirements

### Requirement: Public Endpoints sind ohne Auth zugänglich

Die Routen `POST /api/auth/login`, `POST /api/auth/refresh`, `POST /api/auth/logout`, `POST /api/auth/request-membership`, `POST /api/auth/register`, `GET /api/auth/token-info`, `POST /api/auth/forgot-password`, `POST /api/auth/reset-password`, `GET /api/profile/email/confirm`, `GET /api/files/{id}/download`, `GET /api/members/{id}/sepa-mandat/download` SHALL ohne Bearer-Token erreichbar sein und dürfen NICHT mit 401 antworten, nur weil kein Token vorliegt. `GET /api/uploads/*` ist NICHT mehr Teil dieser Liste und SHALL nicht ohne Autorisierung ausgeliefert werden (siehe Anforderung „Upload-Auslieferung erfordert Berechtigung").

#### Scenario: Login ohne Token
- **WHEN** ein Aufruf an `POST /api/auth/login` mit gültigem Body und ohne `Authorization`-Header gemacht wird
- **THEN** antwortet der Server NICHT mit 401 (200/400 je nach Body-Validität ist erlaubt)

#### Scenario: Tokenloser Upload-Zugriff wird abgelehnt
- **WHEN** ein Aufruf an `GET /api/uploads/<datei>` ohne gültiges Download-Token gemacht wird
- **THEN** antwortet der Server mit 401 oder 403 und liefert die Datei NICHT aus

## ADDED Requirements

### Requirement: Upload-Auslieferung erfordert Berechtigung

Das System SHALL Dateien unter `/api/uploads/*` nur gegen ein gültiges, kurzlebiges Download-Token ausliefern (analog zum bestehenden `file-download-token`-Muster für SEPA-Mandate). Das Token SHALL nur an authentifizierte Aufrufer ausgestellt werden, die die jeweilige Datei sehen dürfen (Ownership-/Sichtbarkeitsprüfung analog `policy.MemberCan`). Die Auslieferung SHALL `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store` setzen.

#### Scenario: Berechtigter Aufrufer erhält Token und Datei
- **WHEN** ein authentifizierter Aufrufer, der ein Mitgliedsfoto sehen darf, ein Download-Token anfordert und damit `GET /api/uploads/<datei>` aufruft
- **THEN** wird das Token ausgestellt und die Datei mit 200 sowie `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store` ausgeliefert

#### Scenario: Token für nicht sichtbares Foto wird verweigert
- **WHEN** ein authentifizierter Aufrufer ein Download-Token für ein Foto anfordert, das er nicht sehen darf
- **THEN** antwortet der Server mit 403 und stellt kein Token aus

#### Scenario: Ungültiges oder abgelaufenes Token
- **WHEN** `GET /api/uploads/<datei>` mit einem ungültigen oder abgelaufenen Token aufgerufen wird
- **THEN** antwortet der Server mit 401 oder 403 und liefert die Datei NICHT aus
