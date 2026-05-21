# Design Document: Profile & Member Restructuring

## Inhaltsverzeichnis
1. [UI Design](#ui-design)
2. [API Spezifikation](#api-spezifikation)
3. [Datenbank Schema](#datenbank-schema)
4. [State Management](#state-management)

---

## UI Design

### A. Nutzer-Profil (ProfilePage.tsx)

#### Tab Navigation
```
┌─────────────────────────────────────────────────────────────┐
│ [Konto] [Profil] [Mitgliedsdaten] [Sonstiges]              │
└─────────────────────────────────────────────────────────────┘
```

**Mobile**: Horizontal scrollbar oder Dropdown-Select

---

### Tab 1: KONTO

#### Struktur
```
┌─────────────────────────────────────────────────────────┐
│ Kontoangaben                                            │
├─────────────────────────────────────────────────────────┤
│ Name           [_________________]                      │
│ E-Mail (read-only) ulrich.frank@web.de                 │
├─────────────────────────────────────────────────────────┤
│ Sicherheit                                              │
├─────────────────────────────────────────────────────────┤
│ [Passwort ändern]  [E-Mail ändern]                      │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ [Speichern]  (disabled bei load)                        │
│ ✓ Gespeichert  (nach save, 2s)                          │
└─────────────────────────────────────────────────────────┘
```

**Logik:**
- Dirty-Flag: Save-Button disabled bis Name-Änderung
- Passwort + Email: Separate Modal-Dialoge via Buttons
- Save-Button: Speichert nur Name-Änderungen

#### Modal: Passwort ändern

Wird geöffnet beim Klick auf `[Passwort ändern]` Button.

```
┌─────────────────────────────────────────────────────────┐
│  Passwort ändern                                    [×] │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ Aktuelles Passwort [___________________]               │
│ Neues Passwort     [___________________]               │
│ Wiederholen        [___________________]               │
│                                                         │
│ ⚠️ Fehler-Message (falls vorhanden)                     │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ [Abbrechen]  [Passwort ändern]                          │
└─────────────────────────────────────────────────────────┘
```

**Logik:**
- Modal-Dialog bei Button-Klick öffnen
- Validierung: Neue Passwörter müssen identisch sein
- Success: "Passwort geändert. Du wirst ausgeloggt…" + Auto-Logout nach 2.5s
- Error: Toast "Aktuelles Passwort nicht korrekt" oder "Fehler beim Speichern"
- Modal schließt sich nach erfolgreichem Save

#### Modal: E-Mail ändern

Wird geöffnet beim Klick auf `[E-Mail ändern]` Button.

```
┌─────────────────────────────────────────────────────────┐
│  E-Mail-Adresse ändern                             [×] │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ Neue E-Mail-Adresse [___________________]              │
│ Passwort zur Bestätigung [___________________]         │
│                                                         │
│ ⚠️ Fehler-Message (falls vorhanden)                     │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ [Abbrechen]  [Bestätigungs-Mail senden]                │
└─────────────────────────────────────────────────────────┘
```

**Logik:**
- Modal-Dialog bei Button-Klick öffnen
- Validierung: Gültige Email-Adresse
- Success: "Bestätigungs-Mail gesendet. Bitte prüfe dein neues Postfach."
- Error: Toast "Passwort nicht korrekt" oder "E-Mail-Adresse bereits vergeben"
- Modal schließt sich nach erfolgreichem Submit

---

### Tab 2: PROFIL

#### Struktur
```
┌─────────────────────────────────────────────────────────┐
│ Profilbild                                              │
├─────────────────────────────────────────────────────────┤
│ [Foto 80x80] [Upload-Button] "JPEG, PNG, max 5MB"      │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Kontaktinformationen                                    │
├─────────────────────────────────────────────────────────┤
│ Straße         [_________________]                      │
│ PLZ            [_______] Ort [___________________]      │
│                                                         │
│ Telefonnummern                                          │
│ + Mobil        [0711 123456] [×]                        │
│ + Privat       [0711 654321] [×]                        │
│ [+ Nummer hinzufügen]                                   │
│                                                         │
│ Sichtbarkeit für Mitglieder                             │
│ ☑ Telefonnummern sichtbar                               │
│ ☑ Adresse sichtbar                                      │
│ ☑ Profilbild sichtbar                                   │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Bankdaten                                               │
├─────────────────────────────────────────────────────────┤
│ IBAN           [DE__ ____ ____ ____ ____ ____]          │
│                                                         │
│ SEPA-Mandat    ☐ Erteilt (read-only)                    │
│                (verwaltet durch Verein)                │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Familie                                                 │
├─────────────────────────────────────────────────────────┤
│ Meine Kinder (wenn role=elternteil)                     │
│ • Anna Schmidt                                          │
│ • Emma Schmidt                                          │
│                                                         │
│ Meine Elternteile (wenn role=spieler)                   │
│ • Petra Schmidt (petra@example.com)                     │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ [Speichern]  (disabled bei load)                        │
│ ✓ Gespeichert  (nach save, 2s)                          │
└─────────────────────────────────────────────────────────┘
```

**Logik:**
- Foto: Auto-upload bei select (kein separater Save-Button)
- Adresse: Wenn 1 Feld geändert → ganze Adresse wird Draft
- Telefone: add/remove inline, alle zusammen als Draft
- Dirty-Flag für ganze Sektion

---

### Tab 3: MITGLIEDSDATEN

Nur sichtbar wenn `ownMember` vorhanden.

#### Struktur
```
┌─────────────────────────────────────────────────────────┐
│ Persönliche Daten                                       │
├─────────────────────────────────────────────────────────┤
│ Vorname        Max → Maximilian ⏳ (nur wenn Draft)      │
│ Nachname       Mustermann → Muster ⏳ (nur wenn Draft)   │
│ Geburtsdatum   01.01.2005 (read-only)                   │
│ Geschlecht     männlich (read-only)                     │
│ Status         aktiv (read-only)                        │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Mitgliedsinformationen                                  │
├─────────────────────────────────────────────────────────┤
│ Mitgliedsnummer  12345                                  │
│ Passnummer       ABC123                                 │
│ Rückennummer     7                                      │
│ Positionen       Rückraum Mitte, Linksaußen             │
│ Vereinsfunktion  –                                      │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Kontakt                                                 │
├─────────────────────────────────────────────────────────┤
│ Adresse        Hauptstr. 1, 70000 Stuttgart             │
│                → Neue Str. 5, 71000 Ludwigsburg ⏳      │
│                (nur wenn Draft)                        │
│                                                         │
│ Telefonnummern +49 711 123456                           │
│                → +49 711 654321 ⏳ (nur wenn Draft)      │
│                                                         │
│ E-Mail         max@example.com                          │
│                → max.neu@example.com ⏳ (nur wenn Draft) │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Passfoto                                                │
├─────────────────────────────────────────────────────────┤
│ [Foto 80x80] [Upload-Button] ⏳ (nur wenn Draft)        │
│              "JPEG, PNG, max 5MB"                       │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Kontonummer                                             │
├─────────────────────────────────────────────────────────┤
│ IBAN           DE12 3456 7890 1234 5678 90              │
│                → DE98 7654 3210 9876 5432 10 ⏳          │
│                (nur wenn Draft)                        │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Datenschutz                                             │
├─────────────────────────────────────────────────────────┤
│ Datenverarbeitung ☑ ja                                  │
│ Datenweitergabe   ☑ ja                                  │
│                   → ☑ nein ⏳ (nur wenn Draft)           │
│ SEPA-Mandat       ☐ nein                                │
│                   → ☑ ja ⏳ (nur wenn Draft)             │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ [Speichern]  (disabled bei load)                        │
│ ✓ Gespeichert  (nach save, 2s)                          │
└─────────────────────────────────────────────────────────┘
```

**Logik:**
- Alle Draft-Felder zeigen: [Original] → [Draft] mit ⏳-Symbol
- Nur editable Felder können Drafts haben
- Speichern-Button: speichert alle Drafts (nicht Originals)

---

### Tab 4: SONSTIGES

```
┌─────────────────────────────────────────────────────────┐
│ Fahrzeug                                                │
├─────────────────────────────────────────────────────────┤
│ Sitzplätze     [5] ↑↓                                    │
│ Anmerkungen    [Hänger vorhanden____________]           │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ [Speichern]  (disabled bei load)                        │
│ ✓ Gespeichert  (nach save, 2s)                          │
└─────────────────────────────────────────────────────────┘
```

---

## B. Admin-Mitgliedsverwaltung (MemberDetailPage.tsx)

#### Tab Navigation
```
┌─────────────────────────────────────────────────────────────┐
│ [Stammdaten] [Kontakt] [Datenschutz] [Familie] [Admin]     │
└─────────────────────────────────────────────────────────────┘
```

### Tab 1: STAMMDATEN

```
┌─────────────────────────────────────────────────────────┐
│ Persönliche Daten                                       │
├─────────────────────────────────────────────────────────┤
│ Vorname        [Max_________]                           │
│                ↓ Nutzer-Anfrage:                        │
│                Maximilian [✓ Accept] [✗ Reject]         │
│                                                         │
│ Nachname       [Mustermann_]                           │
│                ↓ Nutzer-Anfrage:                        │
│                Muster [✓ Accept] [✗ Reject]             │
│                                                         │
│ Geburtsdatum   [01.01.2005]                             │
│ Geschlecht     [m ◉ f ○ u ○]                           │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Mitgliedsinformationen                                  │
├─────────────────────────────────────────────────────────┤
│ Mitgliedsnummer [12345_____]                            │
│ Passnummer      [ABC123____]                            │
│ Rückennummer    [7_________]                            │
│ Positionen      [Rückraum Mitte ✓] [Linksaußen ✓]...   │
│ Status          [aktiv ◉] [verletzt ○] [pausiert ○]... │
│ Vereinsfunktion [– ◉] [Trainer ○] [Vorstand ○]...      │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Foto                                                    │
├─────────────────────────────────────────────────────────┤
│ [Foto 80x80] [Upload-Button]                            │
│                ↓ Nutzer-Anfrage:                        │
│                [Draft 80x80] [✓ Accept] [✗ Reject]      │
│                                                         │
│ ☑ Sichtbar für Mitglieder                               │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ [Speichern]  (direkt speichern, nicht Draft)            │
│ ✓ Gespeichert  (nach save, 2s)                          │
└─────────────────────────────────────────────────────────┘
```

### Tab 2: KONTAKT

```
┌─────────────────────────────────────────────────────────┐
│ Adresse                                                 │
├─────────────────────────────────────────────────────────┤
│ Straße     [Hauptstr._____] Hausnr. [1__]               │
│ PLZ        [70000_] Ort [Stuttgart________]             │
│                                                         │
│            ↓ Nutzer-Anfrage:                            │
│            Neue Str. 5, 71000 Ludwigsburg               │
│            [✓ Accept] [✗ Reject]                        │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Kontaktdaten                                            │
├─────────────────────────────────────────────────────────┤
│ Telefonnummern +49 711 123456                           │
│                +49 711 654321                           │
│                ↓ Nutzer-Anfrage:                        │
│                +49 711 654321 (nur diese)               │
│                [✓ Accept] [✗ Reject]                    │
│                                                         │
│ E-Mail     [max@example.com___________]                 │
│            ↓ Nutzer-Anfrage:                            │
│            max.neu@example.com [✓] [✗]                  │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Bankdaten                                               │
├─────────────────────────────────────────────────────────┤
│ IBAN       [DE12 3456 7890 1234 5678 90]               │
│            ↓ Nutzer-Anfrage:                            │
│            DE98 7654 3210 9876 5432 10 [✓] [✗]          │
│                                                         │
│ SEPA-Mandat ☐ erteilt (am ________)                     │
│ Dokument    [Link] [Upload]                             │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ [Speichern]                                             │
│ ✓ Gespeichert                                           │
└─────────────────────────────────────────────────────────┘
```

### Tab 3: DATENSCHUTZ

```
┌─────────────────────────────────────────────────────────┐
│ DSGVO & SEPA                                            │
├─────────────────────────────────────────────────────────┤
│ Datenverarbeitung ☑ ja (am 01.01.2024)                  │
│ Datenweitergabe   ☑ ja (am 01.01.2024)                  │
│                                                         │
│                   ↓ Nutzer-Anfrage:                     │
│                   ☐ nein [✓ Accept] [✗ Reject]          │
│                                                         │
│ SEPA-Mandat       ☐ nein                                │
│                   ↓ Nutzer-Anfrage:                     │
│                   ☑ ja (am ______) [✓] [✗]              │
│ Dokument          [Link] [Upload]                       │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ [Speichern]                                             │
│ ✓ Gespeichert                                           │
└─────────────────────────────────────────────────────────┘
```

### Tab 4: FAMILIE

```
┌─────────────────────────────────────────────────────────┐
│ Erziehungsberechtigte                                   │
├─────────────────────────────────────────────────────────┤
│ • Maria Schmidt (maria@example.com) [Entfernen]         │
│ • Peter Schmidt (peter@example.com) [Entfernen]         │
│                                                         │
│ Hinzufügen (max. 2): [Dropdown] [Hinzufügen]            │
└─────────────────────────────────────────────────────────┘
```

### Tab 5: ADMIN

```
┌─────────────────────────────────────────────────────────┐
│ Nutzer verknüpfen                                       │
├─────────────────────────────────────────────────────────┤
│ Aktuell verknüpft: Max Mustermann (max@example.com)     │
│                                                         │
│ Ändern: [Dropdown mit allen Nutzern] [Speichern]        │
│         oder [– keine Verknüpfung –]                    │
│                                                         │
│ [Speichern]                                             │
│ ✓ Gespeichert                                           │
└─────────────────────────────────────────────────────────┘
```

---

## Mitgliederliste (MembersPage.tsx) - Draft-Indikator

```
┌────────────────────────────────────────────────────────────┐
│ Name           | Status  | Einritt | Änderungen ausstehend │
├────────────────────────────────────────────────────────────┤
│ Max Mustermann | aktiv   | 2024-01 | ⏳ (clickable)         │
│ Anna Schmidt   | aktiv   | 2024-02 | –                    │
│ Peter Müller   | verletzt| 2023-06 | ⏳ (clickable)         │
└────────────────────────────────────────────────────────────┘
```

- Klick auf ⏳ → scrollt zu entsprechendem Draft in Detail-Seite

---

## API Spezifikation

### Draft-Management

#### GET /members/{id}/change-drafts
```
Response:
{
  "drafts": [
    {
      "id": 1,
      "field_name": "name",
      "old_value": {"first_name": "Max", "last_name": "Mustermann"},
      "new_value": {"first_name": "Maximilian", "last_name": "Muster"},
      "created_at": "2026-05-21T10:30:00Z",
      "created_by_user_id": 5
    },
    {
      "id": 2,
      "field_name": "address",
      "old_value": {"street": "Hauptstr.", "house_number": "1", "zip": "70000", "city": "Stuttgart"},
      "new_value": {"street": "Neue Str.", "house_number": "5", "zip": "71000", "city": "Ludwigsburg"},
      "created_at": "2026-05-21T11:00:00Z",
      "created_by_user_id": 5
    }
  ]
}
```

#### POST /members/{id}/change-request
Request:
```json
{
  "field_name": "name",  // oder "address", "phones", "email", etc.
  "new_value": {"first_name": "Maximilian", "last_name": "Muster"}
}
```

Response: Draft object (wie oben)

#### POST /members/{id}/change-drafts/{draftId}/accept
```
Response:
{
  "status": "accepted",
  "message": "Änderung übernommen"
}
```

Backend:
- Draft → Original (members.*)
- Draft löschen
- Nutzer benachrichtigung optional

#### DELETE /members/{id}/change-drafts/{draftId}
```
Response:
{
  "status": "rejected",
  "message": "Änderung abgelehnt"
}
```

Backend:
- Draft löschen
- Email an Nutzer: "Deine Änderung bei [field] konnte nicht übernommen werden. Bitte wende dich an den Verein."

---

## Datenbank Schema

### member_change_drafts (NEW)

```sql
CREATE TABLE member_change_drafts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  member_id INTEGER NOT NULL,
  field_name VARCHAR(50) NOT NULL,
  old_value TEXT NOT NULL,  -- JSON
  new_value TEXT NOT NULL,  -- JSON
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by_user_id INTEGER,
  
  FOREIGN KEY (member_id) REFERENCES members(id),
  FOREIGN KEY (created_by_user_id) REFERENCES users(id),
  UNIQUE(member_id, field_name)
);

CREATE INDEX idx_member_drafts ON member_change_drafts(member_id);
```

---

## State Management

### Nutzer-Profil

```typescript
interface DraftField {
  field_name: 'name' | 'address' | 'phones' | 'email' | 'photo_url' | 'iban' | 'sepa_mandat' | 'dsgvo'
  old_value: any
  new_value: any
  created_at: string
}

interface ProfileState {
  // Tab: Konto
  accountName: string
  accountNameChanged: boolean
  
  // Tab: Profil
  address: {street, zip, city}
  addressChanged: boolean
  phones: Phone[]
  phonesChanged: boolean
  visibility: {phones, address, photo}
  visibilityChanged: boolean
  iban: string
  ibanChanged: boolean
  
  // Tab: Mitgliedsdaten (Drafts)
  memberDrafts: DraftField[] // Alle ausstehenden Drafts
  
  // UI State
  saving: boolean
  saved: boolean
  error?: string
}
```

### Admin-Mitgliederverwaltung

```typescript
interface MemberDetailState {
  member: Member
  memberDrafts: DraftField[] // Drafts für dieses Mitglied
  
  // Editable Felder (direkt, nicht Draft)
  form: {
    first_name, last_name, date_of_birth, ...
  }
  formChanged: boolean
  
  // UI State
  saving: boolean
  saved: boolean
  error?: string
}

// Accept/Reject Handlers
handleAcceptDraft(draftId: number)
handleRejectDraft(draftId: number)
```

---

## Fehlerbehandlung

### Draft-Fehler
- Accept fehlgeschlagen: Toast "Fehler bei Übernahme der Änderung"
- Reject fehlgeschlagen: Toast "Fehler beim Ablehnen der Änderung"
- Email-Versand fehlgeschlagen: Silently (Draft wird trotzdem gelöscht, Log-Entry)

### Validierung
- IBAN: Format-Validierung vor Draft-Speicherung
- Adresse: Alle 4 Felder müssen ausgefüllt sein
- Name: Mindestens 2 Zeichen

---

## Responsive Design

### Mobile (< 640px)
- Tabs als Dropdown-Select oder horizontale Scroll-Bar
- Draft-Display: Kompakter, ggf. nur Symbol ohne Text
- Buttons [✓][✗]: größer (mindestens 44px)

### Desktop (≥ 640px)
- Tabs immer sichtbar
- Draft-Display: vollständig mit Text

