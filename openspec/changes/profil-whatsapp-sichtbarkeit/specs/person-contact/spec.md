## MODIFIED Requirements

### Requirement: Kontaktdaten-Endpoint
Das System SHALL einen Endpoint `GET /api/users/:id/contact` bereitstellen, der für einen authentifizierten
Nutzer die öffentlich freigegebenen Kontaktdaten einer Person zurückgibt, inklusive des Flags `whatsapp_visible`.

#### Scenario: Nutzer mit freigegebenen Daten inkl. WhatsApp
- **GIVEN** ein authentifizierter Nutzer
- **WHEN** `GET /api/users/42/contact` aufgerufen wird und Nutzer 42 hat `phones_visible=true`,
  `address_visible=true` und `whatsapp_visible=true` gesetzt
- **THEN** gibt der Endpoint `{ name, photo_url, phones: [...], address: "...", whatsapp_visible: true }` zurück

#### Scenario: Nutzer mit Telefon aber ohne WhatsApp-Freigabe
- **GIVEN** ein authentifizierter Nutzer
- **WHEN** `GET /api/users/42/contact` aufgerufen wird und Nutzer 42 hat `phones_visible=true`
  aber `whatsapp_visible=false`
- **THEN** gibt der Endpoint `{ name, phones: [...], whatsapp_visible: false }` zurück

#### Scenario: Nutzer ohne Freigaben
- **GIVEN** ein authentifizierter Nutzer
- **WHEN** `GET /api/users/42/contact` aufgerufen wird und Nutzer 42 hat keine Sichtbarkeiten freigegeben
- **THEN** gibt der Endpoint `{ name, whatsapp_visible: false }` zurück (nur Name)

#### Scenario: Nutzer nicht gefunden
- **WHEN** `GET /api/users/99999/contact` aufgerufen wird und user_id 99999 existiert nicht
- **THEN** antwortet der Endpoint mit HTTP 404

#### Scenario: Nicht authentifiziert
- **WHEN** `GET /api/users/42/contact` ohne gültigen JWT aufgerufen wird
- **THEN** antwortet der Endpoint mit HTTP 401

### Requirement: PersonChip zeigt WhatsApp-Link nur bei Freigabe
Das System SHALL den WhatsApp-Link im PersonChip-Tooltip nur anzeigen, wenn `whatsapp_visible=true`
im Contact-Response enthalten ist.

#### Scenario: WhatsApp-Link sichtbar
- **WHEN** ein PersonChip-Tooltip Kontaktdaten mit `whatsapp_visible=true` anzeigt und Telefonnummern vorhanden sind
- **THEN** erscheint neben jeder Telefonnummer ein „WhatsApp"-Link (wa.me/…)

#### Scenario: WhatsApp-Link ausgeblendet
- **WHEN** ein PersonChip-Tooltip Kontaktdaten mit `whatsapp_visible=false` anzeigt
- **THEN** werden Telefonnummern nur mit `tel:`-Link angezeigt, kein „WhatsApp"-Link
