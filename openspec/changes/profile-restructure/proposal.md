# Profile & Member Management Restructuring

## Problem Statement

Beide Profile (Nutzer und Mitglied) sind aktuell unstrukturiert:
- **Nutzer-Profil**: 11 separate Boxen ohne klare Kategorisierung, Speichern-Buttons überall
- **Admin-Mitgliedsverwaltung**: Lineares Layout ohne Gruppierung, schwer zu navigieren
- **Daten-Redundanz**: Stammdaten (Name, Adresse, Telefon, Email, Bild) existieren an 2 Stellen
- **Unklare Verantwortung**: Nutzer können Mitgliedsdaten nicht ändern, obwohl sie Besitz darüber hätten

## Solution Overview

### A) Nutzer-Profil (ProfilePage) - Mit Tabs strukturieren
### B) Admin-Mitgliedsverwaltung (MemberDetailPage) - Mit Tabs strukturieren  
### C) Change-Request-Workflow für kritische Stammdaten
   - Nutzer können Änderungen **anfordern** (Draft speichern)
   - Verein muss Änderungen **akzeptieren/ablehnen** (explizit genehmigen)
   - Nur Verein kann Original überschreiben

---

## Detaildesign

### A. NUTZER-PROFIL (ProfilePage.tsx) - Neue Tab-Struktur

```
┌─────────────────────────────────────────────────────────────┐
│ [Konto] [Profil] [Mitgliedsdaten] [Sonstiges]              │
└─────────────────────────────────────────────────────────────┘
```

#### Tab 1: KONTO (editable, Save-Button disabled by default)

**Kontoangaben**
- Name (editable, aus `users.name` oder Nutzer-Tabelle)
- E-Mail (read-only, aus `users.email`)

**Sicherheit**
- Passwort ändern (Form, validiert aktuelles PW)
- E-Mail ändern (Form, versendet Bestätigungs-Link)

**Daten-Quelle**: `GET /profile/account` + `GET /profile/me`  
**Speichert zu**: `PUT /profile/account`, `POST /profile/password`, `POST /profile/email`

---

#### Tab 2: PROFIL (editable, Save-Button disabled by default)

**Profilbild**
- Foto (upload, auto-save ohne separaten Button)

**Kontaktinformationen**
- Adresse (Straße, PLZ, Ort) — editable
- Telefonnummern (add/remove, inline)
- Sichtbarkeit (3 Checkboxes: Telefon, Adresse, Foto)

**Bankdaten**
- IBAN (editable)
- SEPA-Mandat (read-only checkbox, verwaltet über Admin)

**Familie** (read-only)
- Meine Kinder (wenn `role='elternteil'`)
- Meine Elternteile (wenn `role='spieler'`)

**Daten-Quellen**:  
- `GET /profile/me` (Adresse, Telefon, Sichtbarkeit, IBAN)
- `GET /profile/vehicle` (später, nicht relevant hier)
- `GET /family` (Kinder/Elternteile)

**Speichert zu**: 
- `PUT /profile/me` (Adresse, IBAN, Familie)
- `POST /upload/user-photo` (Foto, auto-save)
- `POST /profile/phones` (Telefon, add)
- `DELETE /profile/phones/{id}` (Telefon, remove)
- `PUT /profile/visibility` (Sichtbarkeit)

---

#### Tab 3: MITGLIEDSDATEN

Nur für Nutzer mit `ownMember` (d.h. Spieler).

**Stammdaten** (teils editable mit Draft)
- **Name** (Vorname, Nachname — zusammen als ein Draft)
  - Falls Draft vorhanden: [Original Name] → [Draft Name] mit Buttons [✓] [✗]
- Geburtsdatum, Passnummer, Rückennummer (read-only)
- Position, Geschlecht (read-only)
- Status (read-only)

**Mitgliedsinformationen** (read-only)
- Vereinsfunktion

**Kontakt** (teils editable mit Draft)
- **Adresse** (Straße, Hausnummer, PLZ, Ort — zusammen als ein Draft)
  - Falls Draft vorhanden: [Original-Adresse komplett] → [Draft-Adresse komplett] ⏳
- **Telefonnummern** (alle zusammen als ein Draft)
  - Falls Draft vorhanden: ⏳ Symbol
- **E-Mail** (editable → Draft)
  - Falls Draft vorhanden: ⏳ Symbol

**Passfoto** (editable → Draft)
- Foto-Thumbnail (upload als Draft)
- Falls Draft vorhanden: ⏳ Symbol

**Kontonummer** (editable → Draft)
- IBAN (editable → Draft)
- Falls Draft vorhanden: ⏳ Symbol

**DSGVO & SEPA** (teils editable mit Draft)
- Datenverarbeitung eingewilligt (checkbox, editable → Draft)
  - Falls Draft vorhanden: ⏳ Symbol
- Datenweitergabe eingewilligt (checkbox, editable → Draft)
  - Falls Draft vorhanden: ⏳ Symbol
- SEPA-Mandat erteilt (checkbox, editable → Draft)
  - Falls Draft vorhanden: ⏳ Symbol
- SEPA-Dokument (Link zum PDF, falls vorhanden, read-only)

**Daten-Quelle**: 
- Original: `GET /profile/me` → Feld `own_member` (kommt von Member-API)
- Draft: `GET /members/{id}/change-drafts`

**Speichert zu**: 
- Draft-Felder: `POST /members/{id}/change-request` (speichert als Draft, nicht Original)
- Foto-Upload: `POST /upload/member-photo/{memberId}` (speichert als Draft)

---

#### Tab 4: SONSTIGES

**Fahrzeug** (editable, Save-Button disabled by default)
- Sitzplätze
- Anmerkungen

**Daten-Quelle**: `GET /profile/vehicle`  
**Speichert zu**: `PUT /profile/vehicle`

---

### B. ADMIN-MITGLIEDSVERWALTUNG (MemberDetailPage.tsx) - Neue Tab-Struktur

```
┌─────────────────────────────────────────────────────────────┐
│ [Stammdaten] [Kontakt] [Datenschutz] [Familie] [Admin]     │
└─────────────────────────────────────────────────────────────┘
```

#### Tab 1: STAMMDATEN (editable)

**Persönliche Daten**
- Vorname (editable direkt)
  - Falls Nutzer-Draft ausstehend: [Original] → [Draft] mit Buttons [✓] [✗]
- Nachname (editable direkt)
  - Falls Nutzer-Draft ausstehend: [Original] → [Draft] mit Buttons [✓] [✗]
- Geburtsdatum (editable direkt)
- Geschlecht (editable direkt)

**Mitgliedsinformationen**
- Mitgliedsnummer
- Passnummer
- Rückennummer
- Positionen (Multi-Select Buttons)
- Status (aktiv/verletzt/pausiert/passiv/ausgetreten)
- Vereinsfunktion (Trainer/Vorstand/Beisitzer)

**Foto**
- Passfoto (upload direkt, auto-save)
  - Falls Nutzer-Draft ausstehend: [Original] + [Draft] Thumbnail mit Buttons [✓] [✗]
- Profilfoto für Mitglieder sichtbar (checkbox)

**Daten-Quelle**: 
- Original: `GET /members/{id}`
- Drafts: `GET /members/{id}/change-drafts`

**Speichert zu**: 
- Direkt: `PUT /members/{id}` (Admin kann direkt ändern)
- Foto-Upload: `POST /upload/member-photo/{id}` (auto-save als Draft oder direkt)
- Accept Draft: `POST /members/{id}/change-drafts/{draftId}/accept` → Original überschreiben, Draft löschen
- Reject Draft: `DELETE /members/{id}/change-drafts/{draftId}` → Draft löschen, Original bleibt

---

#### Tab 2: KONTAKT (editable)

**Adresse** (teils editable mit Draft)
- Straße, Hausnummer, PLZ, Ort (editable direkt, zusammen)
  - Falls Nutzer-Draft ausstehend: [Original-Adresse komplett] → [Draft-Adresse komplett] mit Buttons [✓] [✗]
- Eintrittsdatum (read-only)

**Kontaktdaten** (teils editable mit Draft)
- **Telefonnummern** (editable direkt, zusammen als ein Draft)
  - Falls Nutzer-Draft ausstehend: [Original List] → [Draft List] mit Buttons [✓] [✗]
- **E-Mail** (editable direkt)
  - Falls Nutzer-Draft ausstehend: [Original] → [Draft] mit Buttons [✓] [✗]

**Bankdaten** (teils editable mit Draft)
- IBAN (editable direkt)
  - Falls Nutzer-Draft ausstehend: [Original] → [Draft] mit Buttons [✓] [✗]
- SEPA-Mandat erteilt (checkbox + Datum, read-only)
- SEPA-Dokument (upload/anzeigen, nur Admin kann ändern direkt)

**Daten-Quellen**:  
- Original: `GET /members/{id}` (Adresse, IBAN, E-Mail, Telefone, SEPA-Status)
- Drafts: `GET /members/{id}/change-drafts`

**Speichert zu**: 
- Direkt: `PUT /members/{id}` (Admin kann direkt ändern)
- Accept Draft: `POST /members/{id}/change-drafts/{draftId}/accept` → Original überschreiben, Draft löschen
- Reject Draft: `DELETE /members/{id}/change-drafts/{draftId}` → Draft löschen, Original bleibt

---

#### Tab 3: DATENSCHUTZ (editable)

**DSGVO** (teils editable mit Draft — zusammen als ein Draft)
- Datenverarbeitung eingewilligt (checkbox + Datum, editable direkt)
- Datenweitergabe eingewilligt (checkbox + Datum, editable direkt)
- Falls Nutzer-Draft ausstehend: [Original-DSGVO-Status] → [Draft-DSGVO-Status] mit Buttons [✓] [✗]

**SEPA-Mandat** (teils editable mit Draft)
- SEPA-Mandat erteilt (checkbox + Datum, editable direkt)
  - Falls Nutzer-Draft ausstehend: [Original] → [Draft] mit Buttons [✓] [✗]
- SEPA-Dokument (upload/anzeigen, nur Admin kann ändern direkt)

**Daten-Quellen**:  
- Original: `GET /members/{id}`
- Drafts: `GET /members/{id}/change-drafts`

**Speichert zu**: 
- Direkt: `PUT /members/{id}` (Admin kann direkt ändern)
- Accept Draft: `POST /members/{id}/change-drafts/{draftId}/accept` → Original überschreiben, Draft löschen
- Reject Draft: `DELETE /members/{id}/change-drafts/{draftId}` → Draft löschen, Email an Nutzer senden

---

#### Tab 4: FAMILIE (read-only + add/remove)

**Erziehungsberechtigte**
- Liste: Name, E-Mail, Entfernen-Button
- Dropdown zum Hinzufügen (max. 2)

**Daten-Quelle**: `GET /admin/members/{id}/parents`  
**Speichert zu**: `POST /admin/family-links`, `DELETE /admin/family-links`

---

#### Tab 5: ADMIN (editable)

**Nutzer verknüpfen**
- Dropdown: Nutzer auswählen
- Info: "Aktuelle Verknüpfung: [Name]"
- Save-Button

**Daten-Quelle**: `GET /admin/users`, `GET /members/{id}` (feld `user_id`)  
**Speichert zu**: `PUT /admin/members/{id}/user`

---

## Change-Request-Workflow

### Kritische Felder (mit Draft-System)

Diese Felder können vom Nutzer angefordert werden, brauchen aber Verein-Genehmigung:
- **Name** (first_name, last_name — zusammen als ein Draft)
- **Adresse** (street, house_number, zip, city — zusammen als ein Draft)
- **Telefonnummern** (alle numbers — zusammen als ein Draft)
- **E-Mail**
- **Foto/Passfoto**
- **Kontonummer/IBAN**
- **DSGVO** (dsgvo_verarbeitung, dsgvo_weitergabe — zusammen als ein Draft)
- **SEPA-Mandat** (sepa_mandat)

### Draft-Modell

```
members.first_name         ← "Golden Record" (vom Verein gepflegt)
member_change_drafts.first_name ← "Draft" (Nutzer-Anfrage, pending)

Workflow:
  1. Nutzer ändert Feld → speichert in member_change_drafts
  2. Nutzer sieht: Original + Symbol (⏳ Änderung ausstehend)
  3. Verein sieht: Draft + Buttons [✓] [✗]
  4. Verein klickt [✓] → Draft → members.*, Draft gelöscht
  5. Verein klickt [✗] → Draft gelöscht, members.* unverändert
```

### Datenbank: member_change_drafts

```sql
CREATE TABLE member_change_drafts (
  id INTEGER PRIMARY KEY,
  member_id INTEGER FK,
  field_name VARCHAR -- Werte: 'name'|'address'|'phones'|'email'|'photo_url'|'iban'|'sepa_mandat'|'dsgvo'
  old_value JSON,     -- JSON mit allen Komponenten des Feldes
  new_value JSON,     -- JSON mit allen Komponenten des Feldes
  created_at TIMESTAMP,
  created_by_user_id INTEGER FK, -- der Nutzer, der die Änderung angefordert hat
  UNIQUE(member_id, field_name)   -- nur ein Draft pro Feld
);
```

**Beispiele:**

```json
// name
field_name: 'name'
old_value: {first_name: "Max", last_name: "Mustermann"}
new_value: {first_name: "Maximilian", last_name: "Muster"}

// address
field_name: 'address'
old_value: {street: "Hauptstr.", house_number: "1", zip: "70000", city: "Stuttgart"}
new_value: {street: "Neue Str.", house_number: "5", zip: "71000", city: "Ludwigsburg"}

// phones
field_name: 'phones'
old_value: [{label: "Mobil", number: "0711 123456"}]
new_value: [{label: "Mobil", number: "0711 654321"}, {label: "Arbeit", number: "0711 999999"}]

// dsgvo
field_name: 'dsgvo'
old_value: {verarbeitung: false, weitergabe: false}
new_value: {verarbeitung: true, weitergabe: true}
```

### Datenfluss & Redundanzen

| Feld | Original | Draft (field_name) | Admin kann ändern? | Nutzer kann anfordern? |
|------|----------|-------|-------|-------|
| **Name** | `members.first_name, last_name` | `'name'` | ✓ Direkt | ✓ Draft |
| **Adresse** | `members.street, house_number, zip, city` | `'address'` | ✓ Direkt | ✓ Draft |
| **Telefonnummern** | `member_phones.*` | `'phones'` | ✓ Direkt | ✓ Draft |
| **E-Mail** | `members.email` | `'email'` | ✓ Direkt | ✓ Draft |
| **Foto** | `members.photo_url` | `'photo_url'` | ✓ Direkt | ✓ Draft |
| **IBAN** | `members.iban` | `'iban'` | ✓ Direkt | ✓ Draft |
| **SEPA-Mandat** | `members.sepa_mandat` | `'sepa_mandat'` | ✓ Direkt | ✓ Draft |
| **DSGVO** | `members.dsgvo_verarbeitung, dsgvo_weitergabe` | `'dsgvo'` | ✓ Direkt | ✓ Draft |
| **Status** | `members.status` | – | ✓ Direkt | ✗ Nein |
| **Position** | `members.position` | – | ✓ Direkt | ✗ Nein |

### Datenfluss-Diagramm

```
┌─────────────────────────────────────────────────────────────┐
│                        NUTZER-PROFIL                         │
└─────────────────────────────────────────────────────────────┘
         │
         ├─ users.name, users.email
         │
         ├─ users.street, users.zip, users.city (Adresse)
         │   └─ Falls Member existiert:
         │      └─ ⚠️ KONFLIKT mit members.street/zip/city?
         │
         ├─ user_phones.* (Telefonnummern)
         │
         ├─ user_visibility.* (Sichtbarkeit)
         │
         ├─ users.iban (IBAN — nur Nutzer)
         │
         └─ members.* (Member-Daten)
            ├─ members.first_name, last_name, date_of_birth
            ├─ members.position, status, jersey_number
            ├─ members.street, zip, city (Adresse — KONFLIKT?)
            ├─ members.photo_url (Passfoto)
            ├─ members.sepa_mandat, sepa_mandat_date (SEPA)
            ├─ members.dsgvo_verarbeitung, dsgvo_weitergabe
            └─ family_links.* (Kinder/Elternteile)

┌─────────────────────────────────────────────────────────────┐
│                  ADMIN-MITGLIEDSVERWALTUNG                   │
└─────────────────────────────────────────────────────────────┘
         │
         └─ members.* (alle Felder)
            └─ user_linked (Verknüpfung zu users.id)
               └─ users.email, user_phones.* (anzeigen, read-only)
```

---

## UI-Details: Change-Draft-Anzeige

### Nutzer-Seite (Member mit Draft)

```
┌──────────────────────────────────────────────────────┐
│ Adresse                                              │
├──────────────────────────────────────────────────────┤
│ Straße:  Hauptstraße 1  →  Neue Straße 5  ⏳        │
│ PLZ:     70000          →  71000           ⏳        │
│ Ort:     Stuttgart       →  Ludwigsburg    ⏳        │
│                                                      │
│ [Änderung abbrechen]                                │
└──────────────────────────────────────────────────────┘
```

### Admin-Seite (Member-Tabelle)

In der Mitgliederliste ein kleines **⏳-Icon** zeigen, wenn es Drafts gibt:

```
┌──────────────────────────────────────────────────────┐
│ Name           | Status  | Einritt | Änderungen    │
├──────────────────────────────────────────────────────┤
│ Max Mustermann | aktiv   | 2024-01 | ⏳             │
│ Anna Schmidt   | aktiv   | 2024-02 | –             │
└──────────────────────────────────────────────────────┘
```

### Admin-Seite (Member-Detail)

In der Kontakt/Stammdaten-Seite der Draft inline zeigen mit Buttons:

```
┌──────────────────────────────────────────────────────┐
│ Adresse                                              │
├──────────────────────────────────────────────────────┤
│ Straße: Hauptstraße 1                                │
│         ↓ angeforderte Änderung:                     │
│         Neue Straße 5     [✓ Accept] [✗ Reject]     │
│                                                      │
│ PLZ:    70000                                        │
│         ↓ angeforderte Änderung:                     │
│         71000              [✓ Accept] [✗ Reject]     │
└──────────────────────────────────────────────────────┘
```

---

## Draft-Spezifikationen

### Ein Draft pro Feld

- Wenn Nutzer ein Feld mehrmals ändert, wird der alte Draft **überschrieben** (nicht angehängt)
- Pro Feld nur der **neueste Draft** existiert
- DB: `member_change_drafts` hat UNIQUE-Constraint auf `(member_id, field_name)`

### Ablehnung = Email an Nutzer

Wenn Admin einen Draft **ablehnt**:
- Draft wird gelöscht
- **E-Mail an Nutzer**: "Deine Änderung bei [Fieldname] konnte nicht übernommen werden. Bitte wende dich an den Verein."
- Template-Text noch zu definieren

### Annahme = Kein Email

Wenn Admin akzeptiert:
- Draft → Original überschreiben
- Draft wird gelöscht
- Keine Email (implizit akzeptiert)

---

## Implementation Notes

- **Dirty-Flag**: Speichern-Button ist disabled, aktiviert sich nur bei Änderung (außer Drafts)
- **Draft-Buttons**: [✓] und [✗] sind klein, inline neben den Änderungen
- **Icons**: ⏳ zeigt "Änderung ausstehend", ist klickbar → scrollt zu Draft-Feld
- **Auto-Save** Foto-Upload: Foto wird als Draft gespeichert, nicht direkt
- **Responsive**: Tabs auf Mobile als swipebar oder Dropdown
- **Fehlerbehandlung**: Auf Save-Fehler Toast-Feedback zeigen
- **Unique Draft**: Pro (member_id, field_name) nur 1 Draft möglich

