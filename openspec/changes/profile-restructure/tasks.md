# Task List: Profile & Member Restructuring

## Phase 1: Backend-Foundation (API + DB)

### T1.1: Database Migration
**Beschreibung:** Erstelle `member_change_drafts` Tabelle mit allen Feldern und Indizes
- [ ] New Migration: `internal/db/migrations/00X_member_change_drafts.up.sql`
- [ ] Schema: id, member_id, field_name, old_value (JSON), new_value (JSON), created_at, created_by_user_id
- [ ] UNIQUE(member_id, field_name) Constraint
- [ ] Index auf member_id
- [ ] Down-Migration

**Dependencies:** None
**Effort:** 1 Point

---

### T1.2: Draft-API Endpoints (Backend)
**Beschreibung:** Implementiere alle Draft-Management Endpoints in Go
- [ ] `GET /members/{id}/change-drafts` — Alle Drafts für Mitglied abrufen
  - Response: `{drafts: []}`
  - Auth: Admin/Vorstand oder Nutzer selbst
- [ ] `POST /members/{id}/change-request` — Neuen Draft erstellen/updaten
  - Body: `{field_name, new_value}`
  - Validierung: Feldname muss erlaubt sein (Enum)
  - UPSERT: Überschreibe bestehenden Draft für selbes Feld
  - Response: Draft object
  - Auth: Nutzer selbst
- [ ] `POST /members/{id}/change-drafts/{draftId}/accept` — Draft übernehmen
  - Merge: Draft → members.*
  - Delete Draft
  - Log: Qui akzeptiert hat, wann
  - Response: `{status: "accepted"}`
  - Auth: Admin only
- [ ] `DELETE /members/{id}/change-drafts/{draftId}` — Draft ablehnen
  - Send rejection email
  - Delete Draft
  - Log
  - Response: `{status: "rejected"}`
  - Auth: Admin only

**Testing:**
- [ ] Unit Tests für Draft-JSON Parsing
- [ ] Integration Tests für alle Endpoints

**Dependencies:** T1.1
**Effort:** 5 Points

---

### T1.3: Email-Template für Draft-Ablehnung
**Beschreibung:** Erstelle Email-Template für Ablehnung
- [ ] File: `internal/mailer/templates/member_change_rejected.txt`
- [ ] Template mit Variablen: {UserName}, {FieldName}
- [ ] Email-Versand im Reject-Handler integrieren

**Testing:**
- [ ] Email wird versendet bei Draft-Ablehnung

**Dependencies:** T1.2
**Effort:** 1 Point

---

## Phase 2: Frontend — Nutzer-Profil

### T2.1: ProfilePage — Tab-Struktur umbauen
**Beschreibung:** Refaktoriere ProfilePage in 4 Tabs (Konto, Profil, Mitgliedsdaten, Sonstiges)
- [ ] Entferne: 11 separate Boxen ohne Tabs
- [ ] Erstelle: Tab-Navigation mit React State
- [ ] State: `activeTab: 'account' | 'profile' | 'member' | 'misc'`
- [ ] localStorage: Aktiver Tab wird gespeichert
- [ ] Mobile: Responsive Tab-Navigation (Dropdown oder Scroll)
- [ ] Conditional Render: Mitgliedsdaten-Tab nur wenn `ownMember` vorhanden

**Testing:**
- [ ] E2E: Tab-Wechsel funktioniert
- [ ] localStorage persist funktioniert
- [ ] Mobile: Tabs sind nutzbar

**Dependencies:** None
**Effort:** 3 Points

---

### T2.2: Konto-Tab implementieren
**Beschreibung:** Implementiere Konto-Tab mit Name, Passwort, Email (Modals)
- [ ] Section "Kontoangaben": Name (editable), Email (read-only)
- [ ] Section "Sicherheit": 2 Buttons `[Passwort ändern]` + `[E-Mail ändern]`
- [ ] Dirty-Flag: Save-Button disabled bei Load, enabled bei Name-Änderung
- [ ] Save-Button: Speichert nur Name
- [ ] Success-Message: 2s Toast nach erfolgreichem Save
- [ ] Error-Handling: Toast bei Fehler

**Modal: Passwort ändern**
- [ ] Modal-Dialog mit 3 Input-Felder (aktuell, neu, wiederholen)
- [ ] Validierung: Passwörter müssen identisch sein
- [ ] Buttons: [Abbrechen] [Passwort ändern]
- [ ] Success: "Passwort geändert. Du wirst ausgeloggt…" + Auto-Logout nach 2.5s
- [ ] Error: Toast "Aktuelles Passwort nicht korrekt"
- [ ] Modal schließt sich nach erfolgreichem Save

**Modal: E-Mail ändern**
- [ ] Modal-Dialog mit 2 Input-Feldern (neue Email, Passwort-Verifikation)
- [ ] Validierung: Gültige Email-Format
- [ ] Buttons: [Abbrechen] [Bestätigungs-Mail senden]
- [ ] Success: "Bestätigungs-Mail gesendet. Bitte prüfe dein neues Postfach."
- [ ] Error: Toast "Passwort nicht korrekt" oder "E-Mail-Adresse bereits vergeben"
- [ ] Modal schließt sich nach erfolgreichem Submit

**API Calls:**
- [ ] `PUT /profile/account` — Name speichern
- [ ] `POST /profile/password` — Passwort ändern
- [ ] `POST /profile/email` — Email-Change-Request

**Testing:**
- [ ] E2E: Name speichern funktioniert
- [ ] E2E: Passwort-Change Modal öffnet → speichern → Logout
- [ ] E2E: Email-Change Modal öffnet → speichern → Email versendet
- [ ] E2E: Modal abbrechen → kein Save

**Dependencies:** T2.1
**Effort:** 4 Points

---

### T2.3: Profil-Tab implementieren
**Beschreibung:** Implementiere Profil-Tab (Adresse, Telefon, Foto, Bankdaten, Familie, Sichtbarkeit)
- [ ] Profilbild: Upload-Button, Auto-Save zu `/upload/user-photo`
- [ ] Adresse: Straße, PLZ, Ort (editable, Save-Button)
- [ ] Telefonnummern: Add/Remove inline, Save-Button
- [ ] Sichtbarkeit: 3 Checkboxes (Telefon, Adresse, Foto), Save-Button
- [ ] Bankdaten: IBAN editable, SEPA-Mandat read-only, Save-Button
- [ ] Familie: Kinder/Elternteile (read-only, conditional render)
- [ ] Dirty-Flag: Save-Button disabled bis Änderung
- [ ] Success-Message: 2s Toast

**API Calls:**
- [ ] `PUT /profile/me` — Adresse, IBAN, Familie
- [ ] `POST /upload/user-photo` — Foto auto-save
- [ ] `POST /profile/phones` — Phone hinzufügen
- [ ] `DELETE /profile/phones/{id}` — Phone entfernen
- [ ] `PUT /profile/visibility` — Sichtbarkeit speichern
- [ ] `GET /family` — Kinder/Elternteile laden

**Testing:**
- [ ] E2E: Adresse speichern
- [ ] E2E: Telefon add/remove
- [ ] E2E: Foto upload auto-speichern
- [ ] E2E: Sichtbarkeit speichern

**Dependencies:** T2.1
**Effort:** 5 Points

---

### T2.4: Mitgliedsdaten-Tab implementieren (mit Draft)
**Beschreibung:** Implementiere Mitgliedsdaten-Tab mit Draft-System
- [ ] Conditional: Nur anzeigen wenn `ownMember` vorhanden
- [ ] Read-Only Felder: Geb.-Datum, Passnummer, Rückennummer, Position, Geschlecht, Status, Vereinsfunktion
- [ ] Draft-Felder (editable → Draft):
  - Name (Vorname + Nachname zusammen)
  - Adresse (Straße + Hausnummer + PLZ + Ort zusammen)
  - Telefonnummern (alle zusammen)
  - Email
  - Foto
  - IBAN
  - SEPA-Mandat (Checkbox)
  - DSGVO (2 Checkboxes zusammen)
- [ ] Draft-Display: `[Original] → [Draft] ⏳` wenn Draft vorhanden
- [ ] Draft-Cancel Button: Abbrechen-Button bei jedem Draft-Feld
- [ ] Save-Button: Speichert ALLE Drafts (nicht Originals)
- [ ] Dirty-Flag: Save-Button disabled bis Änderung

**API Calls:**
- [ ] `GET /members/{id}/change-drafts` — Drafts laden (bei Tab-Load)
- [ ] `POST /members/{id}/change-request` — Draft erstellen/updaten
- [ ] `DELETE /members/{id}/change-drafts/{id}` — Draft abbrechen

**Validierung:**
- [ ] IBAN Format
- [ ] Adresse: Alle 4 Felder müssen gefüllt sein
- [ ] Name: Min. 2 Zeichen je Feld

**Testing:**
- [ ] E2E: Name ändern → Draft wird erstellt + angezeigt
- [ ] E2E: Adresse ändern (ein Feld) → ganze Adresse wird Draft
- [ ] E2E: Mehrfach ändern selbes Feld → Draft wird überschrieben
- [ ] E2E: Draft abbrechen → Draft wird gelöscht

**Dependencies:** T1.2, T2.1
**Effort:** 8 Points

---

### T2.5: Sonstiges-Tab implementieren
**Beschreibung:** Implementiere Sonstiges-Tab (Fahrzeug)
- [ ] Fahrzeug-Section: Sitzplätze (Number Input), Anmerkungen (Text)
- [ ] Save-Button: Disabled bis Änderung
- [ ] Dirty-Flag
- [ ] Success-Message

**API Calls:**
- [ ] `PUT /profile/vehicle` — Fahrzeug speichern

**Testing:**
- [ ] E2E: Fahrzeug speichern

**Dependencies:** T2.1
**Effort:** 2 Points

---

## Phase 3: Frontend — Admin-Mitgliederverwaltung

### T3.1: MemberDetailPage — Tab-Struktur umbauen
**Beschreibung:** Refaktoriere MemberDetailPage in 5 Tabs (Stammdaten, Kontakt, Datenschutz, Familie, Admin)
- [ ] Entferne: Lineares Layout
- [ ] Erstelle: Tab-Navigation
- [ ] State: `activeTab: 'stammdaten' | 'kontakt' | ...`
- [ ] localStorage: Aktiver Tab wird gespeichert
- [ ] Conditional: Familie + Admin nur für existierende Members (nicht neu)

**Testing:**
- [ ] E2E: Tab-Wechsel
- [ ] E2E: localStorage persist

**Dependencies:** None
**Effort:** 3 Points

---

### T3.2: Stammdaten-Tab mit Draft-Handling
**Beschreibung:** Implementiere Stammdaten-Tab mit Draft-Accept/Reject
- [ ] Direkt-Editable Felder: Vorname, Nachname, Geb.-Datum, Geschlecht, Passnummer, Rückennummer, Positionen, Status, Vereinsfunktion
- [ ] Draft-Display: Wenn Draft für Name vorhanden, zeige `[Original] → [Draft]` mit `[✓] [✗]` Buttons
- [ ] Foto-Draft: Thumbnail von Draft + `[✓] [✗]` Buttons
- [ ] Save-Button: Speichert direkt zu members.* (nicht Draft)
- [ ] Dirty-Flag für direkte Änderungen

**API Calls:**
- [ ] `GET /members/{id}/change-drafts` — Drafts laden
- [ ] `PUT /members/{id}` — Direkte Änderungen speichern
- [ ] `POST /members/{id}/change-drafts/{id}/accept` — Draft akzeptieren
- [ ] `DELETE /members/{id}/change-drafts/{id}` — Draft ablehnen

**Testing:**
- [ ] E2E: Draft accept → Original wird updated
- [ ] E2E: Draft reject → Email wird versendet
- [ ] E2E: Direkte Änderung (z.B. Status) → speichert sofort

**Dependencies:** T1.2, T3.1
**Effort:** 5 Points

---

### T3.3: Kontakt-Tab mit Draft-Handling
**Beschreibung:** Implementiere Kontakt-Tab (Adresse, Telefon, Email, IBAN)
- [ ] Direkt-Editable: Adresse (alle 4 Felder), Telefonnummern (add/remove), Email, IBAN, SEPA-Mandat-Checkbox
- [ ] Draft-Display: Für Adresse, Telefone, Email, IBAN (wenn Draft vorhanden)
- [ ] Format: `[Original] → [Draft]` mit `[✓] [✗]` Buttons
- [ ] Save-Button: Speichert direkte Änderungen zu members.*
- [ ] Dirty-Flag

**API Calls:**
- [ ] `GET /members/{id}/change-drafts`
- [ ] `PUT /members/{id}` — Direkte Änderungen
- [ ] `POST /members/{id}/change-drafts/{id}/accept`
- [ ] `DELETE /members/{id}/change-drafts/{id}`

**Testing:**
- [ ] E2E: Adresse-Draft accept
- [ ] E2E: Telefon-Draft reject → Email versendet
- [ ] E2E: Admin ändert Email direkt (kein Draft)

**Dependencies:** T1.2, T3.1
**Effort:** 5 Points

---

### T3.4: Datenschutz-Tab mit Draft-Handling
**Beschreibung:** Implementiere Datenschutz-Tab (DSGVO + SEPA)
- [ ] Direkt-Editable: DSGVO-Checkboxes (2), SEPA-Checkbox (1)
- [ ] Draft-Display: Wenn Draft vorhanden, zeige `[Original-Status] → [Draft-Status]` mit `[✓] [✗]`
- [ ] Save-Button: Speichert direkte Änderungen
- [ ] Dirty-Flag

**API Calls:**
- [ ] `GET /members/{id}/change-drafts`
- [ ] `PUT /members/{id}` — Direkt speichern
- [ ] `POST /members/{id}/change-drafts/{id}/accept`
- [ ] `DELETE /members/{id}/change-drafts/{id}`

**Testing:**
- [ ] E2E: DSGVO-Draft accept
- [ ] E2E: SEPA-Draft reject

**Dependencies:** T1.2, T3.1
**Effort:** 3 Points

---

### T3.5: Familie-Tab und Admin-Tab (Umzug)
**Beschreibung:** Verschiebe Familie und Admin-Funktionen in neue Tabs
- [ ] Familie-Tab: Erziehungsberechtigte (Liste + Hinzufügen)
- [ ] Admin-Tab: Nutzer-Verknüpfung
- [ ] Conditional: Nur für existierende Members

**Testing:**
- [ ] E2E: Familie add/remove
- [ ] E2E: Nutzer-Verknüpfung

**Dependencies:** T3.1
**Effort:** 2 Points

---

## Phase 4: UI Polish & Integration

### T4.1: Draft-Indikator in MembersPage (Liste)
**Beschreibung:** Zeige ⏳-Icon in Mitgliederliste wenn Drafts ausstehend
- [ ] Spalte "Änderungen ausstehend" mit ⏳-Icon wenn Drafts vorhanden
- [ ] Klick auf ⏳ → navigiert zu MemberDetailPage + scrollt zu Draft-Feld
- [ ] Nur Admin/Vorstand sehen Icon

**API Calls:**
- [ ] Modify `/members` GET um Draft-Count zu integrieren
- [ ] Oder separate API-Call pro Member (ggf. Performance-Optimierung nötig)

**Testing:**
- [ ] E2E: ⏳-Icon wird angezeigt
- [ ] E2E: Klick navigiert korrekt

**Dependencies:** T1.2, T3.2-3.4
**Effort:** 3 Points

---

### T4.2: Mobile-Responsive Design
**Beschreibung:** Stelle sicher dass alles auf Mobile funktioniert
- [ ] Tabs: Scrollbar oder Dropdown
- [ ] Draft-Display: Kompakter Layout
- [ ] Buttons: Min. 44px height
- [ ] Form-Fields: Full-width auf Mobile

**Testing:**
- [ ] E2E auf Mobile (< 640px)

**Dependencies:** T2.2-2.5, T3.2-3.5
**Effort:** 2 Points

---

### T4.3: Error-Handling & Edge Cases
**Beschreibung:** Handle alle Error-Szenarien und Edge Cases
- [ ] Draft-Speichern fehlgeschlagen: Toast Error
- [ ] Draft-Accept fehlgeschlagen: Toast Error (Draft bleibt)
- [ ] Draft-Reject fehlgeschlagen: Toast Error (Draft bleibt)
- [ ] Email-Versand fehlgeschlagen: Silently loggen
- [ ] Mehrfach-Änderung selbes Feld: Draft überschreiben
- [ ] Validierungsfehler: Toast mit Hinweis

**Testing:**
- [ ] Unit-Tests für Validierung
- [ ] Error-Handling Tests

**Dependencies:** T2.4, T3.2-3.4
**Effort:** 2 Points

---

### T4.4: Integration Tests (End-to-End)
**Beschreibung:** Schreibe E2E Tests für kritische Flows
- [ ] Nutzer ändert Name → Admin sieht Draft → Admin akzeptiert → Nutzer sieht "Gespeichert"
- [ ] Nutzer ändert Adresse → Admin lehnt ab → Nutzer kriegt Email
- [ ] Admin ändert direkt → Speichert sofort (kein Draft)
- [ ] Mehrfach-Änderung selbes Feld → Überschreiben

**Testing:**
- [ ] Cypress oder Playwright Tests

**Dependencies:** T2.4, T3.2-3.4
**Effort:** 4 Points

---

## Phase 5: Finalization

### T5.1: Code Review & Refactor
**Beschreibung:** Code-Review und Refactoring
- [ ] React-Komponenten aufräumen
- [ ] Duplicate-Code eliminieren
- [ ] Performance-Optimierungen (Memoization, etc.)

**Dependencies:** T2.2-2.5, T3.2-3.5
**Effort:** 2 Points

---

### T5.2: Documentation
**Beschreibung:** Update README und API-Docs
- [ ] API-Endpoints dokumentieren
- [ ] Nutzer-Dokumentation (wie Drafts funktionieren)
- [ ] Admin-Dokumentation (wie Draft-Workflow funktioniert)

**Dependencies:** All
**Effort:** 1 Point

---

## Summary

| Phase | Tasks | Total Effort |
|-------|-------|--------|
| **1. Backend** | T1.1 - T1.3 | 7 Points |
| **2. Frontend (Nutzer)** | T2.1 - T2.5 | 22 Points |
| **3. Frontend (Admin)** | T3.1 - T3.5 | 18 Points |
| **4. Polish & Integration** | T4.1 - T4.4 | 11 Points |
| **5. Finalization** | T5.1 - T5.2 | 3 Points |
| **TOTAL** | | **61 Points** |

**Estimated Timeline:** 3-4 Sprints (à 2 Wochen)

