# Specification Document: Profile & Member Restructuring

## Funktionale Anforderungen

### Nutzer-Profil (ProfilePage)

#### F1: Tab-Navigation
- 4 Tabs: Konto, Profil, Mitgliedsdaten, Sonstiges
- Nur Mitgliedsdaten-Tab für Nutzer mit `ownMember` sichtbar
- Aktiver Tab wird in localStorage gespeichert (Persistenz)

#### F2: Dirty-Flag für Save-Button
- Button `[Speichern]` ist disabled bei Load
- Button wird enabled, wenn **irgendein** Feld im Tab geändert wird
- Button bleibt enabled bis nach erfolgreichem Save (oder Error)
- Nach Save: Button wird wieder disabled, Success-Message für 2s zeigen

#### F3: Draft-Felder (Mitgliedsdaten-Tab)
- Folgende Felder speichern als Draft (nicht Original):
  - Name (Vorname + Nachname)
  - Adresse (Straße + PLZ + Ort + Hausnummer)
  - Telefonnummern (alle zusammen)
  - E-Mail
  - Passfoto
  - IBAN
  - SEPA-Mandat
  - DSGVO (Verarbeitung + Weitergabe)

- Wenn Draft vorhanden: `[Original] → [Draft] ⏳` anzeigen
- Draft-Änderung abbrechen möglich: Button "[Änderung abbrechen]"

#### F4: Foto-Upload Auto-Save
- Profil-Tab: `POST /upload/user-photo` auto-speichern (kein Dialog)
- Mitgliedsdaten-Tab: `POST /upload/member-photo/{memberId}` als Draft speichern

#### F5: Familie (Read-Only)
- Zeige nur Kinder (für `role=elternteil`) oder Elternteile (für `role=spieler`)
- Daten aus `GET /family` endpoint
- Keine Edit-Funktionalität im Nutzer-Profil

---

### Admin-Mitgliederverwaltung (MemberDetailPage)

#### A1: Tab-Navigation
- 5 Tabs: Stammdaten, Kontakt, Datenschutz, Familie, Admin
- Alle Tabs immer sichtbar (nur Admin-Seite)

#### A2: Draft-Anzeige und Handling
- Wenn Draft vorhanden: inline neben Original anzeigen
- Format: `[Original] → [Draft]` mit Buttons `[✓ Accept]` `[✗ Reject]`
- Buttons führen zu API-Calls:
  - Accept: `POST /members/{id}/change-drafts/{draftId}/accept`
  - Reject: `DELETE /members/{id}/change-drafts/{draftId}`

#### A3: Direkte Änderungen (Bypass Draft)
- Admin kann direkt ändern (ohne Draft-Workflow):
  - Status, Position, Vereinsfunktion, SEPA-Dokument
  - Speichert mit `PUT /members/{id}` (normal)

#### A4: Draft-Email bei Ablehnung
- Wenn Admin Reject klickt:
  - Draft wird gelöscht
  - Email an Nutzer versendet:
    ```
    Betreff: Deine Änderung in deinen Mitgliedsdaten
    
    Hallo [Name],
    
    Deine Änderung bei [Feldname] konnte nicht übernommen werden.
    Bitte wende dich an den Verein falls du Fragen hast.
    
    Viele Grüße
    Team Stuttgart
    ```

#### A5: Mitgliederliste (MembersPage)
- Icon `⏳` in Spalte "Änderungen ausstehend" wenn Drafts vorhanden
- Klick auf `⏳` → scrollt zu MemberDetailPage und highlightet Draft-Feld
- Nur Admin/Vorstand sehen diesen Icon

---

## Technische Anforderungen

### T1: Database Schema
- Neue Tabelle `member_change_drafts`:
  - `id` PRIMARY KEY
  - `member_id` FK → members
  - `field_name` VARCHAR (enum-like)
  - `old_value` JSON (TEXT)
  - `new_value` JSON (TEXT)
  - `created_at` TIMESTAMP
  - `created_by_user_id` FK → users
  - UNIQUE(member_id, field_name) — nur 1 Draft pro Feld

### T2: API Endpoints (New/Modified)

#### GET /members/{id}/change-drafts
- Returns: `{drafts: [DraftField[]]}`
- Auth: Admin, Vorstand, oder der Nutzer selbst (für sein Profil)

#### POST /members/{id}/change-request
- Body: `{field_name, new_value}`
- Creates or Updates Draft (UNIQUE constraint)
- Returns: Draft object
- Auth: Nutzer selbst (nur für sein eigenes Member-Profil)

#### POST /members/{id}/change-drafts/{draftId}/accept
- Merges Draft → Original (members.*)
- Deletes Draft
- Returns: `{status: "accepted"}`
- Auth: Admin only

#### DELETE /members/{id}/change-drafts/{draftId}
- Deletes Draft
- Sends rejection email to user
- Returns: `{status: "rejected"}`
- Auth: Admin only

### T3: Email Template
- Template: `email_templates/member_change_rejected.txt`
- Variablen: `{userName}`, `{fieldName}`

### T4: Frontend State Management
- ProfilePage: Separate state per Tab (können unabhängig dirty sein)
- MemberDetailPage: Globaler Draft-State aus API
- Re-fetch Drafts nach Accept/Reject

### T5: Validierung
- IBAN: `modernc.org/sqlite` oder extern validieren
- Adresse: Alle 4 Felder müssen vorhanden sein
- Name: Vorname + Nachname, je min. 2 Zeichen
- Email: Standard email pattern
- Telefon: Beliebiges Format (Freitext)

### T6: Fehlerbehandlung
- Draft-Speichern fehlgeschlagen: Toast Error
- Draft-Accept fehlgeschlagen: Toast Error (Draft bleibt)
- Draft-Reject fehlgeschlagen: Toast Error (Draft bleibt)
- Email-Versand fehlgeschlagen: Silently loggen, Draft wird trotzdem gelöscht

---

## Edge Cases

### EC1: Mehrfach-Änderung Selben Feldes
- Wenn Nutzer Feld X zweimal ändert, wird erster Draft überschrieben
- `UNIQUE(member_id, field_name)` erzwingt das
- Frontend: Toast "Änderung aktualisiert"

### EC2: Admin ändert gleichzeitig wie Nutzer Draft anfordert
- Conflict möglich bei Adresse: Admin setzt Adresse direkt, Nutzer fordert Änderung an
- Lösung: Nutzer sieht "Original-Adresse" im Draft, Admin sieht Update auf der Seite
- Keine explizite Konflikt-Auflösung nötig (Admin kann Reject wählen)

### EC3: Nutzer ohne Mitgliedschaft sieht Mitgliedsdaten-Tab
- Tab wird nicht angezeigt (Conditional Rendering)
- `if (!ownMember) return null` für Tab

### EC4: Telefonnummern-Draft mit unterschiedlichen Längen
- Wenn Nutzer 2 Nummern hatte, jetzt nur 1 hinzufügt
- Draft speichert: `old_value: [{...}, {...}]` und `new_value: [{...}]`
- Admin sieht: "2 Nummern → 1 Nummer"

---

## Performance Anforderungen

### P1: Draft-Abfrage
- `GET /members/{id}/change-drafts` sollte <100ms sein
- Index auf `member_id` obligatorisch

### P2: Seiten-Load
- ProfilePage: `GET /profile/me` + `GET /members/{id}/change-drafts` parallel
- MemberDetailPage: `GET /members/{id}` + `GET /members/{id}/change-drafts` parallel

### P3: Dirty-Flag Check
- Lokal im Frontend (keine API-Calls nötig)

---

## Security Anforderungen

### S1: Authorization
- `POST /members/{id}/change-request`: Nur der Nutzer selbst (verifyUser == memberId.user_id)
- `POST /members/{id}/change-drafts/{draftId}/accept`: Nur Admin
- `DELETE /members/{id}/change-drafts/{draftId}`: Nur Admin
- `GET /members/{id}/change-drafts`: Admin oder der Nutzer selbst

### S2: Input Validation
- IBAN: Standard IBAN Validierung
- Email: Valid email format
- Name: XSS-Protection (HTML escaping)
- Telefon: Beliebig (Freitext, aber escapen)

### S3: Rate Limiting
- `/change-request`: Max 10 Requests pro Minute pro Nutzer
- `/change-drafts/{id}/accept`: Keine Limitierung für Admin

---

## Lokalisierung

### L1: Feldnamen in Fehlermeldungen
- "Name", "Adresse", "Telefonnummern", "E-Mail", "Foto", "IBAN", "SEPA-Mandat", "DSGVO"
- Email-Template: Feldname deutsch

---

## Testing Strategie

### Unit Tests
- Draft-JSON Parsing/Serialization
- Dirty-Flag Logik
- Validierung IBAN, Email, Name

### Integration Tests
- Draft erstellen → Accept → Original updated
- Draft erstellen → Reject → Email versendet
- Mehrfach-Änderung selbes Feld → Überschreiben

### E2E Tests
- Nutzer ändert Name → Admin sieht Draft → Admin akzeptiert → Nutzer sieht "Gespeichert"
- Nutzer ändert Adresse → Admin lehnt ab → Nutzer kriegt Email

