## Context

Das `user_visibility`-Modell hat drei Felder: `phones_visible`, `address_visible`, `photo_visible`. Email ist Pflichtfeld in `users.email` — jeder User hat eine. Die Change fügt `email_visible` als viertes optionales Feld hinzu, nach dem gleichen Muster wie die anderen.

## Goals / Non-Goals

**Goals:**
- `email_visible` nahtlos ins bestehende Modell integrieren
- E-Mail im PersonChip-Tooltip als klickbaren mailto:-Link
- Default opt-out (false) wie alle anderen Sichtbarkeitsfelder

**Non-Goals:**
- `new_email` (pending E-Mail-Wechsel) freigeben — nur `users.email` (bestätigte Adresse)
- E-Mail in anderen Kontexten exponieren (nur PersonChip-Tooltip)

## Decisions

### Entscheidung 1: ALTER TABLE statt neue Tabelle

`email_visible` wird direkt zu `user_visibility` hinzugefügt. SQLite unterstützt `ALTER TABLE ... ADD COLUMN` mit DEFAULT ohne Table-Rebuild. Bestehende Rows erhalten automatisch `email_visible = 0`.

### Entscheidung 2: mailto:-Link im Tooltip

E-Mail als `<a href="mailto:...">` — auf Mobile öffnet das direkt die Mail-App, kein Copy-Paste nötig. Styling: gleiche Textfarbe wie andere Kontaktzeilen, `underline` für Klickbarkeit.

### Entscheidung 3: Position in der Checkbox-Liste

Reihenfolge in `ProfileProfilTab`: Telefon → Adresse → Profilbild → **E-Mail** (am Ende, da am seltensten freigegeben).

### Entscheidung 4: Klickbare Kontaktdaten im Tooltip

Telefonnummern werden als `tel:`-Link (Dialer) gerendert, mit einem separaten `WhatsApp`-Link daneben (`https://wa.me/<E.164>`). E-Mail als `mailto:`-Link.

**WhatsApp-Nummernormalisierung:** `toWhatsAppNumber()` entfernt alle Nicht-Ziffern und konvertiert deutsche Vorwahl-0 zu Länderpräfix 49. Deckt die typischen Eingabeformate ab: `+49 151 ...`, `0151 ...`, `0049 151 ...`. Internationale Nummern mit gesetztem Länderpräfix funktionieren automatisch (das `+` wird durch `replace(/\D/g, '')` entfernt).

## Risks / Trade-offs

- **Kein Risiko für bestehende Daten:** `DEFAULT 0` bedeutet opt-out für alle existierenden User — keine ungewollte E-Mail-Exposition
- **UPSERT muss alle 4 Felder schreiben:** Wenn ein alter Client (ohne email_visible) den UPSERT sendet, fehlt das Feld → es würde auf 0 gesetzt. Kein Problem da Frontend und Backend immer synchron deployed werden
