# maps-provider-preference Specification

## Purpose

Diese Spezifikation beschreibt die Capability `maps-provider-preference`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Kartendienst-Präferenz speichern
Das System SHALL pro Nutzer eine Kartendienst-Präferenz (`auto` | `google` | `apple`) in der Datenbank speichern. Default ist `auto`.

#### Scenario: Neuer Nutzer hat Default-Präferenz
- **WHEN** ein neuer User-Account erstellt wird
- **THEN** ist `maps_provider = 'auto'`

#### Scenario: Ungültiger Wert wird abgelehnt
- **WHEN** `PUT /api/profile/me` mit `maps_provider = 'osm'` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

---

### Requirement: Präferenz im Profil lesen und setzen
`GET /api/profile/me` SHALL `maps_provider` zurückgeben. `PUT /api/profile/me` SHALL `maps_provider` akzeptieren und persistieren.

#### Scenario: Präferenz lesen
- **WHEN** ein authentifizierter Nutzer `GET /api/profile/me` aufruft
- **THEN** enthält die Antwort das Feld `maps_provider` mit dem aktuellen Wert

#### Scenario: Präferenz setzen
- **WHEN** ein authentifizierter Nutzer `PUT /api/profile/me` mit `{"maps_provider": "apple"}` aufruft
- **THEN** antwortet der Server mit HTTP 200 und nachfolgende GET-Aufrufe geben `maps_provider: 'apple'` zurück

---

### Requirement: Präferenz nach Login in AuthContext laden
Nach erfolgreichem Login oder Token-Refresh SHALL die Anwendung `GET /api/profile/me` aufrufen und `mapsProvider` im AuthContext setzen, damit `MapsLink` überall darauf zugreifen kann.

#### Scenario: Präferenz nach Login verfügbar
- **WHEN** ein Nutzer sich einloggt
- **THEN** ist `mapsProvider` im AuthContext gesetzt bevor die erste Seite gerendert wird

#### Scenario: Profil-Fetch schlägt fehl
- **WHEN** `GET /api/profile/me` nach Login einen Fehler zurückgibt
- **THEN** bleibt `mapsProvider` auf dem Default `'auto'` und die App funktioniert weiter

---

### Requirement: Kartendienst-Auswahl im Profil (Sonstiges)
`ProfileMiscTab.tsx` SHALL ein Auswahlfeld mit drei Optionen anzeigen: „Automatisch", „Google Maps", „Apple Maps". Die Auswahl MUSS sofort via `PUT /api/profile/me` gespeichert werden.

#### Scenario: Nutzer wählt Apple Maps
- **WHEN** Nutzer in „Sonstiges" → „Kartendienst" auf „Apple Maps" klickt und speichert
- **THEN** werden alle Maps-Links im Tab/Fenster künftig zu `maps.apple.com` aufgelöst (nach Reload)

---

### Requirement: URL-Auflösung in MapsLink
`MapsLink.tsx` SHALL die Maps-URL je nach `mapsProvider`-Wert aus dem AuthContext bauen.

#### Scenario: Präferenz 'google'
- **WHEN** `mapsProvider = 'google'`
- **THEN** wird `https://maps.google.com/?q=<encoded-address>` gebaut

#### Scenario: Präferenz 'apple'
- **WHEN** `mapsProvider = 'apple'`
- **THEN** wird `https://maps.apple.com/?q=<encoded-address>` gebaut

#### Scenario: Präferenz 'auto' auf iOS/macOS
- **WHEN** `mapsProvider = 'auto'` UND `navigator.userAgent` enthält `iPhone`, `iPad`, `iPod` oder `Macintosh`
- **THEN** wird `https://maps.apple.com/?q=<encoded-address>` gebaut

#### Scenario: Präferenz 'auto' auf anderen Plattformen
- **WHEN** `mapsProvider = 'auto'` UND User-Agent ist kein Apple-Gerät
- **THEN** wird `https://maps.google.com/?q=<encoded-address>` gebaut
