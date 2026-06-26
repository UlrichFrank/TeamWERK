## ADDED Requirements

### Requirement: Serverseitige Passwort-Mindeststärke

Das System SHALL bei jedem passwortsetzenden Endpunkt — `POST /api/auth/register`, `POST /api/auth/reset-password` und `POST /api/profile/password` (ChangePassword) — das Passwort serverseitig validieren, BEVOR es gehasht wird: Es SHALL mindestens 12 Zeichen lang sein und SHALL nicht mehr als 72 Byte umfassen (bcrypt-Grenze; längere Eingaben würden stillschweigend trunkiert). Verstößt die Eingabe gegen die Regel, SHALL der Server mit HTTP 400 antworten und das Passwort NICHT setzen. Die Validierung SHALL für alle drei Endpunkte identisch sein (gemeinsame Funktion).

#### Scenario: Zu kurzes Passwort bei Registrierung
- **WHEN** `POST /api/auth/register` mit einem Passwort von weniger als 12 Zeichen aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400 und legt keinen Account an

#### Scenario: Gültiges Passwort wird akzeptiert
- **WHEN** ein passwortsetzender Endpunkt mit einem Passwort von 12 bis 72 Byte aufgerufen wird
- **THEN** wird das Passwort gehasht und gesetzt (kein 400 wegen der Längenregel)

#### Scenario: Übergroßes Passwort wird abgelehnt
- **WHEN** ein passwortsetzender Endpunkt mit einem Passwort von mehr als 72 Byte aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400 (keine stille bcrypt-Trunkierung)

#### Scenario: Zu kurzes Passwort beim Zurücksetzen
- **WHEN** `POST /api/auth/reset-password` mit einem gültigen Token, aber einem Passwort unter 12 Zeichen aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400 und das Passwort wird nicht gesetzt (ein Kind-Account wird dabei NICHT auf `can_login=1` aktiviert)

### Requirement: Sanfter Upgrade-Hinweis für Bestandspasswörter

Bestehende Passwörter SHALL NICHT zwangsweise zurückgesetzt werden; ein Login mit einem korrekten, aber zu kurzen Passwort SHALL weiterhin erfolgreich sein. Beim erfolgreichen Login SHALL der Server signalisieren, ob das verwendete Passwort kürzer als die aktuelle Mindestlänge ist (Feld `password_change_recommended` in der Login-Antwort), damit das Frontend einen nicht-blockierenden Hinweis zur Passwortänderung anzeigen kann.

#### Scenario: Login mit zu kurzem Bestandspasswort gelingt mit Hinweis
- **WHEN** ein Nutzer mit korrektem, aber kürzer-als-Mindestlänge Passwort `POST /api/auth/login` aufruft
- **THEN** antwortet der Server mit 200, liefert ein gültiges Access-Token UND `password_change_recommended: true`

#### Scenario: Login mit ausreichend langem Passwort ohne Hinweis
- **WHEN** ein Nutzer mit einem Passwort ≥ Mindestlänge einloggt
- **THEN** enthält die Login-Antwort kein `password_change_recommended: true`
