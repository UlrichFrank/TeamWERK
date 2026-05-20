## Why

Die Mitglieder- und Nutzerdaten in TeamWERK sind aktuell zu dünn: Vereinsrelevante Daten wie Adresse, Eintrittsdatum, IBAN, DSGVO-Einwilligungen und SEPA-Mandat fehlen vollständig, während Nutzer keine Kontaktdaten (Telefon, Profilbild) hinterlegen können. Das erschwert Vereinsarbeit und Kommunikation und verhindert eine datenschutzkonforme Datenhaltung.

## What Changes

**Mitglied (Admin-Domäne):**
- Adressfelder: `street`, `zip`, `city` — Nutzer-Adresse wird immer bevorzugt (auch wenn am Mitglied lokal gesetzt); bei Abweichung: ⚠-Icon + Tooltip zeigt beide Adressen. Nutzer ohne verknüpften Account → reine Mitgliederadresse.
- `join_date` (Eintrittsdatum)
- `iban` (nur Admin sichtbar, App-Level-Control)
- `photo_path` + `photo_visible` (Passfoto für Identifikation: Trainer/Vorstand/Admin sehen es immer; andere Nutzer nur wenn `photo_visible=true` explizit freigegeben — Admin-Toggle)
- `dsgvo_verarbeitung` + `dsgvo_verarbeitung_date`
- `dsgvo_weitergabe` + `dsgvo_weitergabe_date` (betrifft Name + Foto)
- `sepa_mandat` (bool) + `sepa_mandat_date` + `sepa_mandat_path` (PDF-Dokument-Upload des unterschriebenen Mandats)

**Nutzer (selbst verwaltbar):**
- Neue Tabelle `user_phones` (user_id, label, number, sort_order)
- Adressfelder inline auf `users`: `street`, `zip`, `city`
- `photo_path` (Profilbild, Filesystem)
- Neue Tabelle `user_visibility` (phones_visible, address_visible, photo_visible) — grobe Sichtbarkeit: ein Toggle pro Datentyp, sichtbar für alle Teammitglieder oder niemanden

**Family-Links-Sichtbarkeit:**
- Andere Elternteile (Rolle `elternteil`) sehen fremde `family_links` nicht mehr
- Trainer, Spieler, Vorstand, Admin sehen Eltern-Kind-Beziehungen weiterhin vollständig
- Kein DB-Schema-Change — reiner Rollencheck im Backend

**Dateiablage:**
- Upload-Endpoints: `POST /api/upload/member-photo/{id}`, `POST /api/upload/user-photo`
- Auslieferung: `GET /api/uploads/{filename}` (nur für eingeloggte Nutzer)
- Speicherort: `storage/uploads/` auf dem VPS-Filesystem, Pfad in DB

## Capabilities

### New Capabilities
- `member-extended-data`: Adresse, Eintrittsdatum, IBAN, DSGVO-Felder, SEPA-Mandat auf Mitglied
- `member-photo`: Passfoto-Upload und -Anzeige für Mitglieder
- `user-contact-data`: Telefonnummern, optionale Adresse, Profilbild auf Nutzer, mit Sichtbarkeitssteuerung
- `file-upload`: Generische Dateiablage (Filesystem), Upload- und Auslieferungs-Endpoints
- `family-link-visibility`: Elternteile sehen fremde Eltern-Kind-Beziehungen nicht

### Modified Capabilities
- `member-management`: Mitglied-API gibt erweiterte Felder zurück; IBAN nur für Admin

## Impact

- **DB-Migrationen**: 2 neue Migrations (members erweitern, neue Tabellen user_phones + user_visibility)
- **Backend**: `internal/members/handler.go` (neue Felder, IBAN-Guard), neues Package `internal/upload/`, `internal/members/handler.go` family-links Rollencheck
- **Frontend**: `MemberDetailPage.tsx` (neue Felder, Upload-Widget), neues Profil-Abschnitt in user-seitigem Profile für Telefon/Bild/Adresse/Sichtbarkeit
- **Filesystem**: `storage/uploads/` muss auf VPS existieren und beschreibbar sein (www-data)
- **Keine neuen externen Abhängigkeiten**
