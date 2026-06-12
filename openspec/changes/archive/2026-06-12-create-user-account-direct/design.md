## Context

Die Nutzerverwaltung auf `/admin/nutzer` bietet heute zwei Wege, neue Nutzer anzulegen:
1. **Einladungslink** (`POST /auth/invite`) — sendet eine E-Mail mit Registrierungstoken
2. **CSV-Import** — legt Einladungen in Bulk an

Beide Wege setzen E-Mail-Zustellung voraus. Ein direktes Anlegen mit sofort aktiven Zugangsdaten fehlt. Die bestehende `users`-Tabelle hat alle nötigen Felder; es braucht nur einen neuen Handler und ein Frontend-Modal.

## Goals / Non-Goals

**Goals:**
- Vorstand und Admin können einen Account direkt anlegen (email, first_name, last_name, password)
- Das generierte Passwort wird im Modal angezeigt und ist per Clipboard-API kopierbar
- Der Account ist sofort login-fähig (kein E-Mail-Bestätigungsschritt)
- Rolle ist fest `standard` — keine Auswahl im Modal

**Non-Goals:**
- „Passwort beim ersten Login ändern"-Zwang (nicht im Scope)
- Welcome-E-Mail (explizit ausgeschlossen)
- Rollenauswahl im Anlege-Modal

## Decisions

### Passwort-Generierung im Frontend

Das Passwort wird clientseitig erzeugt (`crypto.getRandomValues`, ~16 Zeichen aus Buchstaben + Ziffern + Sonderzeichen). Es wird im Klartextfeld angezeigt und anschließend beim Absenden als Klartext zum Backend gesendet, wo es mit bcrypt gehasht wird.

**Alternative:** Backend generiert und gibt Passwort im Response zurück.
**Warum Frontend:** Das Passwort muss dem Admin ohnehin sichtbar sein, bevor der Request abgeschickt wird (damit er es notieren/kopieren kann). Clientseitige Generierung ist hier einfacher und vermeidet einen zusätzlichen Roundtrip.

### Neuer Endpunkt `POST /api/users`

Liegt in der bestehenden Vorstand-Routegroup (`RequireClubFunction("vorstand")`), die Admin via Rollenhierarchie einschließt. Request-Body: `{ email, first_name, last_name, password }`. Response: `201 { id }` oder `409` bei doppelter E-Mail.

**Kein RETURNING:** SQLite-Kompatibilität — `LastInsertId()` nach `INSERT`.

### Copy-Button via Clipboard API

`navigator.clipboard.writeText()` — funktioniert in modernen Browsern ohne externe Abhängigkeit. Visuelles Feedback (Icon-Wechsel zu `Check` für 2 s) ausreichend.

## Risks / Trade-offs

- **Passwort im Klartext im Transit** → HTTPS auf Prod (Nginx + Let's Encrypt), kein Problem
- **Schwaches Passwort möglich** → Das generierte Passwort ist immer stark; Admin kann es nicht manuell überschreiben (readonly-Feld), optional neu generieren via Button
- **Kein Audit-Log** → Wie alle anderen Admin-Aktionen auch; out of scope
