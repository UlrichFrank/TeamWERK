## ADDED Requirements

### Requirement: Idle-Timer erkennt Inaktivität
Das System SHALL Nutzerinteraktionen (Mausbewegung, Tastendruck, Klick, Touch, Scroll) auf `window`-Ebene überwachen. Bei 25 Minuten ohne Interaktion MUSS ein Warn-Dialog erscheinen. Bei 30 Minuten ohne Interaktion MUSS der Nutzer automatisch ausgeloggt werden.

#### Scenario: Vorwarnung nach 25 Minuten Inaktivität
- **WHEN** ein eingeloggter Nutzer 25 Minuten lang keine Maus-, Tastatur- oder Touch-Interaktion ausführt
- **THEN** erscheint ein modaler Dialog mit dem Text "Sie werden in 5 Minuten automatisch abgemeldet" und einem Countdown sowie den Buttons "Angemeldet bleiben" und "Jetzt abmelden"

#### Scenario: Automatischer Logout nach 30 Minuten Inaktivität
- **WHEN** ein eingeloggter Nutzer 30 Minuten lang keine Interaktion ausführt und den Warn-Dialog nicht bestätigt
- **THEN** wird `POST /api/auth/logout` aufgerufen, der Access Token gecleart und der Nutzer zur Login-Seite weitergeleitet

#### Scenario: Timer-Reset bei Nutzeraktivität
- **WHEN** der Nutzer während des Countdowns eine Interaktion ausführt
- **THEN** verschwindet der Warn-Dialog und der 30-Minuten-Timer startet neu

### Requirement: Warn-Dialog bietet Handlungsoptionen
Der Warn-Dialog MUSS zwei Aktionen anbieten: "Angemeldet bleiben" setzt den Timer zurück; "Jetzt abmelden" loggt den Nutzer sofort aus.

#### Scenario: Nutzer wählt "Angemeldet bleiben"
- **WHEN** der Warn-Dialog angezeigt wird und der Nutzer auf "Angemeldet bleiben" klickt
- **THEN** schließt der Dialog, der Idle-Timer wird auf 0 zurückgesetzt und der Nutzer bleibt eingeloggt

#### Scenario: Nutzer wählt "Jetzt abmelden"
- **WHEN** der Warn-Dialog angezeigt wird und der Nutzer auf "Jetzt abmelden" klickt
- **THEN** wird der Nutzer sofort ausgeloggt und zur Login-Seite weitergeleitet

### Requirement: Refresh-Token-Lebensdauer ist auf 2 Tage begrenzt
Der Server SHALL Refresh Tokens mit einer maximalen Laufzeit von 2 Tagen ausstellen. Nach Ablauf MUSS der Nutzer sich neu einloggen.

#### Scenario: Refresh Token nach 2 Tagen abgelaufen
- **WHEN** ein Nutzer die App nach mehr als 2 Tagen Abwesenheit öffnet und ein automatisches Token-Refresh versucht wird
- **THEN** schlägt das Refresh fehl, der Nutzer wird zur Login-Seite weitergeleitet

#### Scenario: Refresh Token vor Ablauf noch gültig
- **WHEN** ein Nutzer die App innerhalb von 2 Tagen wieder öffnet
- **THEN** gelingt das automatische Token-Refresh und der Nutzer ist direkt eingeloggt

### Requirement: Idle-Timer ist nur für eingeloggte Nutzer aktiv
Der Timer MUSS nur aktiv sein, wenn `user !== null` im AuthContext. Auf der Login-Seite und anderen öffentlichen Seiten MUSS kein Timer laufen.

#### Scenario: Nicht eingeloggter Nutzer — kein Timer
- **WHEN** ein nicht eingeloggter Nutzer die Login-Seite besucht
- **THEN** ist kein Idle-Timer aktiv und kein Warn-Dialog erscheint
