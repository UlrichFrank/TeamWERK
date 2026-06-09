## Context

`user_visibility` hat aktuell 4 Boolean-Spalten: `phones_visible`, `address_visible`, `photo_visible`,
`email_visible`. Der PersonChip-Tooltip zeigt immer dann einen WhatsApp-Link (wa.me/...), wenn
`phones_visible=1` — ohne eigene Freigabe. Die Sichtbarkeitskontrollen im Profil sind als Checkboxen
implementiert, während Push-Benachrichtigungen denselben Toggle-Schalter aus `ProfileMiscTab.tsx` verwenden.

## Goals / Non-Goals

**Goals:**
- Neues `whatsapp_visible`-Feld in DB und API
- WhatsApp-Link im PersonChip-Tooltip nur bei expliziter Freigabe
- Alle 5 Sichtbarkeitskontrollen einheitlich als Toggle-Schalter

**Non-Goals:**
- Separate WhatsApp-Nummer (immer dieselben Nummern aus `user_phones`)
- Sichtbarkeitskontrollen im Kind-Profil überarbeiten (gleiche Logik, aber außerhalb dieses Changes)
- Rückwirkende Aktivierung für bestehende Nutzer (Default 0 = aus)

## Decisions

**1. `whatsapp_visible` als neue Spalte in `user_visibility`**
Alternativ wäre eine separate Tabelle oder ein JSON-Blob denkbar. Da alle anderen Sichtbarkeitsfelder
ebenfalls als INTEGER-Spalten in `user_visibility` liegen, ist eine weitere Spalte konsistent und
einfach zu migrieren. Default `0` bedeutet: bestehende Nutzer sehen zunächst keinen WhatsApp-Link.

**2. `whatsapp_visible` im Contact-Response**
Der `GET /api/users/:id/contact`-Endpoint gibt bereits alle sichtbaren Kontaktdaten zurück. Das Flag
wird als Boolean-Feld `whatsapp_visible` im Response ergänzt — so entscheidet der Client, ob er den
WhatsApp-Link rendert. Alternative: Backend liefert Telefonnummern nur ohne WhatsApp-Kontext. Nachteil:
der Client müsste gar keine wa.me-Links generieren können, was künftige Erweiterungen erschwert.

**3. Toggle-Schalter statt Checkboxen**
Die `Toggle`-Komponente existiert bereits in `ProfileMiscTab.tsx` als lokale Inline-Komponente.
Sie wird in eine gemeinsame Datei `web/src/components/Toggle.tsx` extrahiert und von beiden Tabs importiert.

**4. Reihenfolge der Toggles**
Empfohlene Reihenfolge: Telefonnummern → WhatsApp → Adresse → Profilbild → E-Mail-Adresse.
WhatsApp direkt unter Telefonnummern, damit der Zusammenhang klar ist.

## Risks / Trade-offs

- **WhatsApp-Link bricht nicht**: Wenn `phones_visible=true` aber `whatsapp_visible=false`, sehen
  Nutzer weiterhin die Nummer per `tel:`, nur der wa.me-Link entfällt. Kein Datenverlust.
- **Default 0**: Bestehende Nutzer, die bisher implizit den WhatsApp-Link geteilt haben, müssen
  `whatsapp_visible` manuell aktivieren. Akzeptierter Trade-off: explizite Freigabe ist besser.
- **Toggle-Extraktion**: Minimales Refactoring — `ProfileMiscTab.tsx` importiert dann aus
  `components/Toggle.tsx` statt die Komponente inline zu definieren.

## Migration Plan

1. Migration `ALTER TABLE user_visibility ADD COLUMN whatsapp_visible INTEGER NOT NULL DEFAULT 0`
2. Backend deployen (abwärtskompatibel, Default 0)
3. Frontend deployen
4. Rollback: Migration rückgängig (`DROP COLUMN` SQLite ≥ 3.35, oder neues Schema)
