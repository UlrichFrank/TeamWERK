## ADDED Requirements

### Requirement: Vereinsidentität aus Konfiguration
Vereinsname, Kurzname, öffentliche URL und Absender-E-Mail MUST aus der Konfiguration (ENV) stammen. Ohne gesetzte Werte MUST ein neutraler Default greifen; an keiner Stelle des Defaults darf „Team Stuttgart" erscheinen.

#### Scenario: Neutraler Default ohne Konfiguration
- **WHEN** TeamWERK ohne branding-spezifische ENV-Variablen startet
- **THEN** verwendet die Anwendung einen neutralen Vereinsnamen (z. B. „Beispielverein")
- **AND** kein ausgeliefertes Default-Artefakt enthält „Team Stuttgart"

#### Scenario: Konfiguration überschreibt Default
- **WHEN** `CLUB_NAME` und `PUBLIC_URL` per ENV gesetzt sind
- **THEN** verwenden UI, E-Mails und CORS diese Werte

### Requirement: CORS aus konfigurierter Domain
Die CORS-Origin MUST aus der konfigurierten öffentlichen URL abgeleitet werden, nicht hartcodiert sein.

#### Scenario: Konfigurierte Origin wird erlaubt
- **WHEN** eine Anfrage mit `Origin` = konfigurierte `PUBLIC_URL` eintrifft
- **THEN** antwortet der Server mit passendem `Access-Control-Allow-Origin`-Header

#### Scenario: Fremde Origin wird nicht erlaubt
- **WHEN** eine Anfrage mit einer nicht konfigurierten `Origin` eintrifft
- **THEN** setzt der Server keinen `Access-Control-Allow-Origin` für diese Origin

### Requirement: Austauschbare Texte mit neutralem Default
Begrüßungs-E-Mail-Text und Login-/Beitrittsseiten-Texte MUST instanz-spezifisch sein und den konfigurierten Vereinsnamen verwenden. Ohne Konfiguration MUST ein neutraler Default greifen.

#### Scenario: Begrüßungsmail nutzt konfigurierten Vereinsnamen
- **WHEN** eine Begrüßungs-E-Mail versendet wird
- **THEN** enthält der Text den konfigurierten Vereinsnamen und keinen hartcodierten Vereinsbezug

### Requirement: Konfigurierbares Theming
Markenfarben und Logo MUST je Instanz konfigurierbar sein, mit neutralem Default. Im Code dürfen keine rohen Marken-Hex-Werte außerhalb der zentralen Theme-Konfiguration stehen.

#### Scenario: Eigene Markenfarben übernehmen
- **WHEN** eine Instanz eigene `brand-*`-Farbwerte konfiguriert und baut
- **THEN** verwendet das ausgelieferte Frontend diese Farben durchgängig über die `brand-*`-Tokens
