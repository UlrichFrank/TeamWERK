## Context

Die Profil-Seite hat vier Tabs: Konto, Profil, Mitgliedsdaten, Sonstiges. Das Change-Request-System (`member_change_drafts`) ist bereits vollständig implementiert mit `extractFieldValue` / `applyDraftToMember` in `internal/members/drafts.go`. `new_value` und `old_value` sind `TEXT`-Felder, die beliebige JSON-Strukturen speichern. Der UNIQUE-Constraint `(member_id, field_name)` stellt sicher, dass pro Feld nur ein offener Draft existiert.

## Goals / Non-Goals

**Goals:**
- Vorname/Nachname aus dem Konto-Tab entfernen
- Vorname/Nachname und IBAN in den Profil-Tab integrieren
- Für verknüpfte Mitglieder: ein atomarer „profil"-Bundle-Request für alle Profilfelder
- Profil-Tab zeigt „Speichern" (kein Mitglied) oder „Änderung anfordern" (Mitglied)
- Mitgliedsdaten-Tab: Editier-Sektionen (Name, IBAN) entfernen; offene Profil-Anfragen anzeigen

**Non-Goals:**
- Keine Änderung an der Datenbankstruktur — `member_change_drafts` bleibt wie sie ist
- Keine Änderung an anderen Change-Request-Typen (address, photo_url, dsgvo, etc.)
- Kein Zusammenführen von `users.street` und `members.street` — Adresse bleibt in `users` für Nicht-Mitglieder

## Decisions

### field_name: "profil" als neuer Bundle-Typ

Das bestehende `extractFieldValue`/`applyDraftToMember`-Pattern wird um den Fall `"profil"` erweitert:

- `extractFieldValue("profil")`: liest `first_name`, `last_name`, `street`, `zip`, `city`, `iban` aus `members`
- `applyDraftToMember("profil")`: schreibt alle diese Felder in einem UPDATE auf `members`

Der UNIQUE-Constraint `(member_id, field_name)` sorgt automatisch dafür, dass es nur einen offenen Profil-Draft geben kann (UPSERT).

`allowedFields` in `CreateChangeRequestHandler` wird um `"profil"` erweitert.

### PUT /profile/me bekommt first_name/last_name

Für Nicht-Mitglieder (kein Mitglied verknüpft) soll der Profil-Tab `first_name`/`last_name` direkt in `users` speichern. `UpdateProfile`-Handler wird entsprechend erweitert. Für verknüpfte Mitglieder schickt das Frontend stattdessen den Bundle-Request — `PUT /profile/me` wird also nur für Nicht-Mitglieder mit Name aufgerufen.

### Admin-Darstellung des "profil"-Drafts

In `MemberStammdatenTab.tsx` (Admin-Ansicht) werden offene Drafts bereits angezeigt. Der neue `"profil"`-Typ wird dort mit einer lesbaren Aufschlüsselung aller geänderten Felder dargestellt (Name, Adresse, IBAN).

## Risks / Trade-offs

- **Adresse im Bundle:** `members.street/zip/city` wird durch `applyDraftToMember("profil")` geschrieben, aber `users.street/zip/city` bleibt unverändert. Das ist intentional — Nicht-Mitglieder haben Adresse nur in users, Mitglieder nur in members (canonical). Konsistenz-Problem existiert schon heute und wird hier nicht gelöst.
- **Offener Draft sperrt Formular:** Wenn ein Profil-Draft offen ist, soll das Formular im Frontend schreibgeschützt sein (mit Hinweis + Zurückziehen). Kein Datenverlust möglich.
