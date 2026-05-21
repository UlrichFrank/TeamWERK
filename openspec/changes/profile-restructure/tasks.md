# Task List: Profile & Member Restructuring

## Phase 1: Backend-Foundation (API + DB)

### T1.1: Database Migration
**Beschreibung:** Erstelle `member_change_drafts` Tabelle mit allen Feldern und Indizes
- [x] New Migration: `internal/db/migrations/00X_member_change_drafts.up.sql`
- [x] Schema: id, member_id, field_name, old_value (JSON), new_value (JSON), created_at, created_by_user_id
- [x] UNIQUE(member_id, field_name) Constraint
- [x] Index auf member_id
- [x] Down-Migration

**Dependencies:** None
**Effort:** 1 Point

---

### T1.2: Draft-API Endpoints (Backend)
**Beschreibung:** Implementiere alle Draft-Management Endpoints in Go
- [x] `GET /members/{id}/change-drafts` — Alle Drafts für Mitglied abrufen
  - Response: `{drafts: []}`
  - Auth: Admin/Vorstand oder Nutzer selbst
- [x] `POST /members/{id}/change-request` — Neuen Draft erstellen/updaten
  - Body: `{field_name, new_value}`
  - Validierung: Feldname muss erlaubt sein (Enum)
  - UPSERT: Überschreibe bestehenden Draft für selbes Feld
  - Response: Draft object
  - Auth: Nutzer selbst
- [x] `POST /members/{id}/change-drafts/{draftId}/accept` — Draft übernehmen
  - Merge: Draft → members.*
  - Delete Draft
  - Log: Qui akzeptiert hat, wann
  - Response: `{status: "accepted"}`
  - Auth: Admin only
- [x] `DELETE /members/{id}/change-drafts/{draftId}` — Draft ablehnen
  - Send rejection email
  - Delete Draft
  - Log
  - Response: `{status: "rejected"}`
  - Auth: Admin only

**Testing:**
- [x] Unit Tests für Draft-JSON Parsing
- [x] Integration Tests für alle Endpoints

**Dependencies:** T1.1
**Effort:** 5 Points

---

### T1.3: Email-Template für Draft-Ablehnung
**Beschreibung:** Erstelle Email-Template für Ablehnung
- [x] File: `internal/mailer/templates/member_change_rejected.txt`
- [x] Template mit Variablen: {UserName}, {FieldName}
- [x] Email-Versand im Reject-Handler integrieren

**Testing:**
- [x] Email wird versendet bei Draft-Ablehnung

**Dependencies:** T1.2
**Effort:** 1 Point

---

## Phase 2: Frontend — Nutzer-Profil

### T2.1: ProfilePage — Tab-Struktur umbauen
**Beschreibung:** Refaktoriere ProfilePage in 4 Tabs (Konto, Profil, Mitgliedsdaten, Sonstiges)
- [x] Entferne: 11 separate Boxen ohne Tabs
- [x] Erstelle: Tab-Navigation mit React State
- [x] State: `activeTab: 'account' | 'profile' | 'member' | 'misc'`
- [x] localStorage: Aktiver Tab wird gespeichert
- [x] Mobile: Responsive Tab-Navigation (Dropdown oder Scroll)
- [x] Conditional Render: Mitgliedsdaten-Tab nur wenn `ownMember` vorhanden

**Testing:**
- [x] E2E: Tab-Wechsel funktioniert
- [x] localStorage persist funktioniert
- [x] Mobile: Tabs sind nutzbar

**Dependencies:** None
**Effort:** 3 Points

---

### T2.2: Konto-Tab implementieren
**Beschreibung:** Implementiere Konto-Tab mit Name, Passwort, Email (Modals)
- [x] Section "Kontoangaben": Name (editable), Email (read-only)
- [x] Section "Sicherheit": 2 Buttons `[Passwort ändern]` + `[E-Mail ändern]`
- [x] Dirty-Flag: Save-Button disabled bei Load, enabled bei Name-Änderung
- [x] Save-Button: Speichert nur Name
- [x] Success-Message: 2s Toast nach erfolgreichem Save
- [x] Error-Handling: Toast bei Fehler

**Modal: Passwort ändern**
- [x] Modal-Dialog mit 3 Input-Felder (aktuell, neu, wiederholen)
- [x] Validierung: Passwörter müssen identisch sein
- [x] Buttons: [Abbrechen] [Passwort ändern]
- [x] Success: "Passwort geändert. Du wirst ausgeloggt…" + Auto-Logout nach 2.5s
- [x] Error: Toast "Aktuelles Passwort nicht korrekt"
- [x] Modal schließt sich nach erfolgreichem Save

**Modal: E-Mail ändern**
- [x] Modal-Dialog mit 2 Input-Feldern (neue Email, Passwort-Verifikation)
- [x] Validierung: Gültige Email-Format
- [x] Buttons: [Abbrechen] [Bestätigungs-Mail senden]
- [x] Success: "Bestätigungs-Mail gesendet. Bitte prüfe dein neues Postfach."
- [x] Error: Toast "Passwort nicht korrekt" oder "E-Mail-Adresse bereits vergeben"
- [x] Modal schließt sich nach erfolgreichem Submit

**API Calls:**
- [x] `PUT /profile/account` — Name speichern
- [x] `POST /profile/password` — Passwort ändern
- [x] `POST /profile/email` — Email-Change-Request

**Testing:**
- [x] E2E: Name speichern funktioniert
- [x] E2E: Passwort-Change Modal öffnet → speichern → Logout
- [x] E2E: Email-Change Modal öffnet → speichern → Email versendet
- [x] E2E: Modal abbrechen → kein Save

**Dependencies:** T2.1
**Effort:** 4 Points

---

### T2.3: Profil-Tab implementieren
**Beschreibung:** Implementiere Profil-Tab (Adresse, Telefon, Foto, Bankdaten, Familie, Sichtbarkeit)
- [x] Profilbild: Upload-Button, Auto-Save zu `/upload/user-photo`
- [x] Adresse: Straße, PLZ, Ort (editable, Save-Button)
- [x] Telefonnummern: Add/Remove inline, Save-Button
- [x] Sichtbarkeit: 3 Checkboxes (Telefon, Adresse, Foto), Save-Button
- [x] Bankdaten: IBAN editable, SEPA-Mandat read-only, Save-Button
- [x] Familie: Kinder/Elternteile (read-only, conditional render)
- [x] Dirty-Flag: Save-Button disabled bis Änderung
- [x] Success-Message: 2s Toast

**API Calls:**
- [x] `PUT /profile/me` — Adresse, IBAN, Familie
- [x] `POST /upload/user-photo` — Foto auto-save
- [x] `POST /profile/phones` — Phone hinzufügen
- [x] `DELETE /profile/phones/{id}` — Phone entfernen
- [x] `PUT /profile/visibility` — Sichtbarkeit speichern
- [x] `GET /family` — Kinder/Elternteile laden

**Testing:**
- [x] E2E: Adresse speichern
- [x] E2E: Telefon add/remove
- [x] E2E: Foto upload auto-speichern
- [x] E2E: Sichtbarkeit speichern

**Dependencies:** T2.1
**Effort:** 5 Points

---

### T2.4: Mitgliedsdaten-Tab implementieren (mit Draft)
**Beschreibung:** Implementiere Mitgliedsdaten-Tab mit Draft-System
- [x] Conditional: Nur anzeigen wenn `ownMember` vorhanden
- [x] Read-Only Felder: Geb.-Datum, Passnummer, Rückennummer, Position, Geschlecht, Status, Vereinsfunktion
- [x] Draft-Felder (editable → Draft):
  - Name (Vorname + Nachname zusammen)
  - Adresse (Straße + Hausnummer + PLZ + Ort zusammen)
  - Telefonnummern (alle zusammen)
  - Email
  - Foto
  - IBAN
  - SEPA-Mandat (Checkbox)
  - DSGVO (2 Checkboxes zusammen)
- [x] Draft-Display: `[Original] → [Draft] ⏳` wenn Draft vorhanden
- [x] Draft-Cancel Button: Abbrechen-Button bei jedem Draft-Feld
- [x] Save-Button: Speichert ALLE Drafts (nicht Originals)
- [x] Dirty-Flag: Save-Button disabled bis Änderung

**API Calls:**
- [x] `GET /members/{id}/change-drafts` — Drafts laden (bei Tab-Load)
- [x] `POST /members/{id}/change-request` — Draft erstellen/updaten
- [x] `DELETE /members/{id}/change-drafts/{id}` — Draft abbrechen

**Validierung:**
- [x] IBAN Format
- [x] Adresse: Alle 4 Felder müssen gefüllt sein
- [x] Name: Min. 2 Zeichen je Feld

**Testing:**
- [x] E2E: Name ändern → Draft wird erstellt + angezeigt
- [x] E2E: Adresse ändern (ein Feld) → ganze Adresse wird Draft
- [x] E2E: Mehrfach ändern selbes Feld → Draft wird überschrieben
- [x] E2E: Draft abbrechen → Draft wird gelöscht

**Dependencies:** T1.2, T2.1
**Effort:** 8 Points

---

### T2.5: Sonstiges-Tab implementieren
**Beschreibung:** Implementiere Sonstiges-Tab (Fahrzeug)
- [x] Fahrzeug-Section: Sitzplätze (Number Input), Anmerkungen (Text)
- [x] Save-Button: Disabled bis Änderung
- [x] Dirty-Flag
- [x] Success-Message

**API Calls:**
- [x] `PUT /profile/vehicle` — Fahrzeug speichern

**Testing:**
- [x] E2E: Fahrzeug speichern

**Dependencies:** T2.1
**Effort:** 2 Points

---

## Phase 3: Frontend — Admin-Mitgliederverwaltung

### T3.1: MemberDetailPage — Tab-Struktur umbauen
**Beschreibung:** Refaktoriere MemberDetailPage in 5 Tabs (Stammdaten, Kontakt, Datenschutz, Familie, Admin)
- [x] Entferne: Lineares Layout
- [x] Erstelle: Tab-Navigation
- [x] State: `activeTab: 'stammdaten' | 'kontakt' | ...`
- [x] localStorage: Aktiver Tab wird gespeichert
- [x] Conditional: Familie + Admin nur für existierende Members (nicht neu)

**Testing:**
- [x] E2E: Tab-Wechsel
- [x] E2E: localStorage persist

**Dependencies:** None
**Effort:** 3 Points

---

### T3.2: Stammdaten-Tab mit Draft-Handling
**Beschreibung:** Implementiere Stammdaten-Tab mit Draft-Accept/Reject
- [x] Direkt-Editable Felder: Vorname, Nachname, Geb.-Datum, Geschlecht, Passnummer, Rückennummer, Positionen, Status, Vereinsfunktion
- [x] Draft-Display: Wenn Draft für Name vorhanden, zeige `[Original] → [Draft]` mit `[✓] [✗]` Buttons
- [x] Foto-Draft: Thumbnail von Draft + `[✓] [✗]` Buttons
- [x] Save-Button: Speichert direkt zu members.* (nicht Draft)
- [x] Dirty-Flag für direkte Änderungen

**API Calls:**
- [x] `GET /members/{id}/change-drafts` — Drafts laden
- [x] `PUT /members/{id}` — Direkte Änderungen speichern
- [x] `POST /members/{id}/change-drafts/{id}/accept` — Draft akzeptieren
- [x] `DELETE /members/{id}/change-drafts/{id}` — Draft ablehnen

**Testing:**
- [x] E2E: Draft accept → Original wird updated
- [x] E2E: Draft reject → Email wird versendet
- [x] E2E: Direkte Änderung (z.B. Status) → speichert sofort

**Dependencies:** T1.2, T3.1
**Effort:** 5 Points

---

### T3.3: Kontakt-Tab mit Draft-Handling
**Beschreibung:** Implementiere Kontakt-Tab (Adresse, Telefon, Email, IBAN)
- [x] Direkt-Editable: Adresse (alle 4 Felder), Telefonnummern (add/remove), Email, IBAN, SEPA-Mandat-Checkbox
- [x] Draft-Display: Für Adresse, Telefone, Email, IBAN (wenn Draft vorhanden)
- [x] Format: `[Original] → [Draft]` mit `[✓] [✗]` Buttons
- [x] Save-Button: Speichert direkte Änderungen zu members.*
- [x] Dirty-Flag

**API Calls:**
- [x] `GET /members/{id}/change-drafts`
- [x] `PUT /members/{id}` — Direkte Änderungen
- [x] `POST /members/{id}/change-drafts/{id}/accept`
- [x] `DELETE /members/{id}/change-drafts/{id}`

**Testing:**
- [x] E2E: Adresse-Draft accept
- [x] E2E: Telefon-Draft reject → Email versendet
- [x] E2E: Admin ändert Email direkt (kein Draft)

**Dependencies:** T1.2, T3.1
**Effort:** 5 Points

---

### T3.4: Datenschutz-Tab mit Draft-Handling
**Beschreibung:** Implementiere Datenschutz-Tab (DSGVO + SEPA)
- [x] Direkt-Editable: DSGVO-Checkboxes (2), SEPA-Checkbox (1)
- [x] Draft-Display: Wenn Draft vorhanden, zeige `[Original-Status] → [Draft-Status]` mit `[✓] [✗]`
- [x] Save-Button: Speichert direkte Änderungen
- [x] Dirty-Flag

**API Calls:**
- [x] `GET /members/{id}/change-drafts`
- [x] `PUT /members/{id}` — Direkt speichern
- [x] `POST /members/{id}/change-drafts/{id}/accept`
- [x] `DELETE /members/{id}/change-drafts/{id}`

**Testing:**
- [x] E2E: DSGVO-Draft accept
- [x] E2E: SEPA-Draft reject

**Dependencies:** T1.2, T3.1
**Effort:** 3 Points

---

### T3.5: Familie-Tab und Admin-Tab (Umzug)
**Beschreibung:** Verschiebe Familie und Admin-Funktionen in neue Tabs
- [x] Familie-Tab: Erziehungsberechtigte (Liste + Hinzufügen)
- [x] Admin-Tab: Nutzer-Verknüpfung
- [x] Conditional: Nur für existierende Members

**Testing:**
- [x] E2E: Familie add/remove
- [x] E2E: Nutzer-Verknüpfung

**Dependencies:** T3.1
**Effort:** 2 Points

---

## Phase 4: UI Polish & Integration

### T4.1: Draft-Indikator in MembersPage (Liste)
**Beschreibung:** Zeige ⏳-Icon in Mitgliederliste wenn Drafts ausstehend
- [x] Spalte "Änderungen ausstehend" mit ⏳-Icon wenn Drafts vorhanden
- [x] Klick auf ⏳ → navigiert zu MemberDetailPage + scrollt zu Draft-Feld
- [x] Nur Admin/Vorstand sehen Icon

**API Calls:**
- [x] Modify `/members` GET um Draft-Count zu integrieren
- [x] Oder separate API-Call pro Member (ggf. Performance-Optimierung nötig)

**Testing:**
- [x] E2E: ⏳-Icon wird angezeigt
- [x] E2E: Klick navigiert korrekt

**Dependencies:** T1.2, T3.2-3.4
**Effort:** 3 Points

---

### T4.2: Mobile-Responsive Design
**Beschreibung:** Stelle sicher dass alles auf Mobile funktioniert
- [x] Tabs: Scrollbar oder Dropdown
- [x] Draft-Display: Kompakter Layout
- [x] Buttons: Min. 44px height
- [x] Form-Fields: Full-width auf Mobile

**Testing:**
- [x] E2E auf Mobile (< 640px)

**Dependencies:** T2.2-2.5, T3.2-3.5
**Effort:** 2 Points

---

### T4.3: Error-Handling & Edge Cases
**Beschreibung:** Handle alle Error-Szenarien und Edge Cases
- [x] Draft-Speichern fehlgeschlagen: Toast Error
- [x] Draft-Accept fehlgeschlagen: Toast Error (Draft bleibt)
- [x] Draft-Reject fehlgeschlagen: Toast Error (Draft bleibt)
- [x] Email-Versand fehlgeschlagen: Silently loggen
- [x] Mehrfach-Änderung selbes Feld: Draft überschreiben
- [x] Validierungsfehler: Toast mit Hinweis

**Testing:**
- [x] Unit-Tests für Validierung
- [x] Error-Handling Tests

**Dependencies:** T2.4, T3.2-3.4
**Effort:** 2 Points

---

### T4.4: Integration Tests (End-to-End)
**Beschreibung:** Schreibe E2E Tests für kritische Flows
- [x] Nutzer ändert Name → Admin sieht Draft → Admin akzeptiert → Nutzer sieht "Gespeichert"
- [x] Nutzer ändert Adresse → Admin lehnt ab → Nutzer kriegt Email
- [x] Admin ändert direkt → Speichert sofort (kein Draft)
- [x] Mehrfach-Änderung selbes Feld → Überschreiben

**Testing:**
- [x] Cypress oder Playwright Tests

**Dependencies:** T2.4, T3.2-3.4
**Effort:** 4 Points

---

## Phase 5: Finalization

### T5.1: Code Review & Refactor
**Beschreibung:** Code-Review und Refactoring
- [x] React-Komponenten aufräumen
- [x] Duplicate-Code eliminieren
- [x] Performance-Optimierungen (Memoization, etc.)

**Dependencies:** T2.2-2.5, T3.2-3.5
**Effort:** 2 Points

---

### T5.2: Documentation
**Beschreibung:** Update README und API-Docs
- [x] API-Endpoints dokumentieren
- [x] Nutzer-Dokumentation (wie Drafts funktionieren)
- [x] Admin-Dokumentation (wie Draft-Workflow funktioniert)

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

