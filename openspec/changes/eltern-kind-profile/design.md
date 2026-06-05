## Context

Eltern (Rolle `elternteil`) sind via `family_links` (parent_user_id → member_id) mit ihren Kindern verknüpft. Die Kinder sind `members`-Einträge, die optional einen eigenen User-Account besitzen (`members.user_id`). Kontaktdaten wie Adresse liegen teilweise in `members` (street, zip, city), Telefonnummern und Sichtbarkeit jedoch in `user_phones` / `user_visibility` (user-gebunden).

Die bestehende Route `PUT /api/members/:id` für Mitgliedsdaten ist auf admin+vorstand beschränkt. `ProfilePage.tsx` kennt bereits die `children`-Liste, nutzt sie aber nur zur Anzeige. Die Sidebar-Navigation ist statisch.

## Goals / Non-Goals

**Goals:**
- Eltern sehen in der Sidebar dynamische Einträge für jedes Kind (z.B. „Jannes Profil")
- Route `/profil/kind/:memberId` zeigt Kindprofil mit Tabs: Kontakt, Mitgliedsdaten, Bankdaten
- Eltern können alle drei Tab-Bereiche für ihre Kinder bearbeiten
- Backend prüft family_links-Berechtigung auf jedem Endpunkt

**Non-Goals:**
- Kein „Konto"-Tab für Kinder (Passwort/E-Mail gehören dem Kind selbst)
- Kein „Sonstiges"-Tab für Kinder (Dienst-Erinnerungen sind benutzerbezogen)
- Keine Änderung an admin/vorstand-Rechten

## Decisions

### 1. Dedizierte `/api/profile/kind/:memberId/...` Endpunkte statt Anpassung von `PUT /api/members/:id`

Eltern-Schreibzugriff über eigene Endpunkte, nicht über die Admin-Route.

**Warum:** Die Admin-Route prüft `claims.Role == "admin"` für sensible Felder (IBAN, DSGVO). Statt diese Logik zu verkomplizieren, werden neue Endpunkte mit klarer Autorisierung via `family_links` angelegt — einfacher, sicherer, keine Regression.

**Alternativen:** `PUT /api/members/:id` mit Role-Erweiterung — abgelehnt, da die Role-Prüfung in der Mitte des Handlers verstreut ist.

### 2. Dynamische Kind-Einträge in AppShell via eigenem Fetch

AppShell ruft `GET /api/profile/me` für `elternteil`-Nutzer beim Mount auf, um die Kinderliste zu erhalten.

**Warum:** Kein neuer Endpunkt nötig — `/api/profile/me` liefert `children` bereits. AppShell bekommt nur Kinder-Name + ID, kein State-Manager nötig. Fetch nur wenn `user.role === 'elternteil'`.

**Alternativen:** Separater `GET /api/profile/children` Endpunkt — Mehraufwand ohne Vorteil. Globaler Context für Kinder — Overengineering für diesen Use-Case.

### 3. ChildProfilePage als neue Seite, Tab-Komponenten wiederverwendet

`ChildProfilePage.tsx` lädt das Kindprofil via `GET /api/profile/kind/:memberId` und rendert dieselben Tab-Komponenten (`ProfileMemberTab`, `ProfileBankTab`, `ProfileProfilTab`) mit angepassten API-Pfaden.

**Warum:** Code-Wiederverwendung. Die Tab-Komponenten kennen ihren API-Pfad als Prop — minimale Anpassung.

**Alternativen:** Seite komplett neu bauen — unnötige Duplizierung.

### 4. Adress-Bearbeitung für Kinder schreibt in `members.street/zip/city`

Die `members`-Tabelle hat bereits `street`, `zip`, `city`. Telefonnummern und Sichtbarkeit werden nur angezeigt/bearbeitet, wenn das Kind einen eigenen User-Account hat (`member.user_id != null`).

**Warum:** Einheitliches Datenmodell ohne neue Spalten. Kinder ohne Account können trotzdem eine Adresse haben.

## Risks / Trade-offs

**Zwei Adress-Quellen für Kinder mit User-Account** → Der Elternteil bearbeitet `members.street/zip/city`; das Kind selbst bearbeitet `users.street/zip/city`. Zwei verschiedene Felder.
*Mitigation:* Im Kind-Profil wird explizit `members`-Adresse bearbeitet. Die User-Adresse bleibt beim Kind. Langfristig: Konsolidierung in `members` für alle.

**Telefonnummern nur bei Kindern mit Account** → Ohne `user_id` kein Telefon-Tab.
*Mitigation:* Telefon-Sektion im Kontakt-Tab nur wenn `member.user_id != null`. Klar kommunizieren.

**Nav-Fetch erzeugt zusätzlichen API-Call beim App-Start** → Nur für `elternteil`, kleines JSON-Response (nur Namen + IDs).
*Mitigation:* Tolerierbar; Caching via React-State (kein Re-Fetch bei Navigation).

## Migration Plan

1. Neue Backend-Endpunkte hinzufügen (keine DB-Migration nötig)
2. Frontend: `ChildProfilePage.tsx` + Route + AppShell-Dynamik
3. Deployment via `make deploy` (kein Datenbankschema-Änderung)

## Open Questions

- Sollen Bankdaten (IBAN) nur für Eltern sichtbar sein, oder auch für das Kind selbst via dem eigenen Profil? (Aktuell: Kind sieht IBAN nicht im eigenen Profil — konsistent lassen)
