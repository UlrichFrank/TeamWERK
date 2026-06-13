# TeamWERK – Testfall-Report

**Stand:** 2026-06-12  
**Zweck:** Vollständige Beschreibung aller geplanten Testfälle zur Prüfung auf fachliche Korrektheit,  
bevor mit der Implementierung begonnen wird.

---

## Legende

| Symbol | Bedeutung |
|--------|-----------|
| ✅ | Test bereits implementiert |
| 🔲 | Test noch zu implementieren |
| ⚠️ | Fachlicher Hinweis / bekanntes Verhalten |

---

## Überblick: Testabdeckung nach Paket

| Paket | Implementiert | Geplant | Gesamt |
|-------|:---:|:---:|:---:|
| `kader` (Age-Bracket-Logik) | 3 | – | 3 |
| `games` | 12 | 3 | 15 |
| `absences` | 4 | 2 | 6 |
| `trainings` | 8 | 2 | 10 |
| `chat` | 11 | 3 | 14 |
| `notify` | 2 | – | 2 |
| `kader` (Handler) | 0 | 5 | 5 |
| **`auth`** | **0** | **15** | **15** |
| **`duties`** | **0** | **18** | **18** |
| **`members`** | **0** | **10** | **10** |

---

## Paket: `auth`

### Hintergrund

Das Auth-Paket regelt Anmeldung, Token-Rotation, Passwort-Reset, Einladungen und Nutzerverwaltung.

**Fachliche Grundregeln:**
- Nur Nutzer mit `can_login = 1` können sich anmelden. Proxy-Accounts (`can_login = 0`) können keine JWT-Token erhalten.
- E-Mail-Vergleich ist case-insensitive (`LOWER(email)`).
- Refresh-Tokens sind opaque (SHA-256-Hash in DB gespeichert), Cookie ist HttpOnly + Secure + SameSite=Strict.
- Jedes Refresh rotiert den Token: der alte wird gelöscht, ein neuer ausgestellt.
- `ForgotPassword` antwortet **immer** mit HTTP 204 — unabhängig davon, ob die E-Mail existiert (Schutz vor User-Enumeration).
- `ResetPassword` und `ChangePassword` löschen **alle** Refresh-Tokens des Nutzers (erzwungener Logout auf allen Geräten).
- `UpdateUserRole` akzeptiert nur `"admin"` oder `"standard"` — nicht `"trainer"`, `"vorstand"` etc.
- Nur ein Admin kann einem anderen Nutzer die Rolle `"admin"` geben.
- Ein Admin kann sein eigenes Konto nicht löschen.
- `DeleteUser` löscht im selben Transaktion: Refresh-Tokens, offene Einladungstoken, Reset-Tokens, Familienlinks, Dienstzuweisungen, Dienstkonten und den User-Eintrag selbst.

---

### TC-A01 — Login mit korrekten Credentials ✅ (zu implementieren) 🔲

**Route:** `POST /api/auth/login`

| | |
|---|---|
| **Vorbedingung** | User mit bekanntem Passwort existiert, `can_login = 1` |
| **Aktion** | POST `{ email, password }` |
| **Erwartetes Ergebnis** | HTTP 200, Response enthält `access_token` (JWT), Cookie `refresh_token` ist gesetzt (HttpOnly) |
| **DB-Seiteneffekt** | `refresh_tokens` enthält neuen Eintrag; `users.last_login_at` aktualisiert |

---

### TC-A02 — Login mit falschem Passwort 🔲

**Route:** `POST /api/auth/login`

| | |
|---|---|
| **Vorbedingung** | User existiert, `can_login = 1` |
| **Aktion** | POST mit falscher Password-Zeichenkette |
| **Erwartetes Ergebnis** | HTTP 401 |
| **DB-Seiteneffekt** | Kein neuer `refresh_tokens`-Eintrag |

---

### TC-A03 — Login mit unbekannter E-Mail 🔲

**Route:** `POST /api/auth/login`

| | |
|---|---|
| **Aktion** | POST mit E-Mail, die in `users` nicht vorkommt |
| **Erwartetes Ergebnis** | HTTP 401 |

---

### TC-A04 — Login mit Proxy-Account (can_login = 0) 🔲

**Route:** `POST /api/auth/login`

**Hinweis:** Proxy-Accounts werden über `CreateProxyAccount` angelegt und haben `can_login = 0`. Sie sollen sich nicht einloggen können.

| | |
|---|---|
| **Vorbedingung** | User mit `can_login = 0` existiert |
| **Aktion** | POST mit korrektem Passwort (leer, da Proxy kein echtes Passwort hat) |
| **Erwartetes Ergebnis** | HTTP 401 (Query filtert `can_login = 1`) |

---

### TC-A05 — Token-Refresh mit gültigem Cookie 🔲

**Route:** `POST /api/auth/refresh`

| | |
|---|---|
| **Vorbedingung** | Gültiger Refresh-Token in `refresh_tokens` (nicht abgelaufen) |
| **Aktion** | POST mit Cookie `refresh_token = <plain-token>` |
| **Erwartetes Ergebnis** | HTTP 200, neuer `access_token`, neues Cookie gesetzt |
| **DB-Seiteneffekt** | Alter Token-Hash gelöscht, neuer eingefügt (Token-Rotation) |

---

### TC-A06 — Token-Refresh mit ungültigem Cookie 🔲

**Route:** `POST /api/auth/refresh`

| | |
|---|---|
| **Aktion** | POST mit gefälschtem oder nicht vorhandenem Cookie-Wert |
| **Erwartetes Ergebnis** | HTTP 401 |

---

### TC-A07 — Logout löscht Refresh-Token und löscht Cookie 🔲

**Route:** `POST /api/auth/logout`

| | |
|---|---|
| **Vorbedingung** | User ist eingeloggt, gültiger Refresh-Token in DB |
| **Aktion** | POST mit Cookie |
| **Erwartetes Ergebnis** | HTTP 204, Cookie hat `MaxAge = -1` (Browser löscht Cookie), Token in DB gelöscht |

---

### TC-A08 — Registrierung mit gültigem Einladungstoken 🔲

**Route:** `POST /api/auth/register`

**Hinweis:** Der `token`-Parameter im Request ist der Klartext-Token. Gespeichert wird der SHA-256-Hash. Die Lookup-Bedingung ist `token = hash AND used_at IS NULL AND expires_at > now`.

| | |
|---|---|
| **Vorbedingung** | `invitation_tokens` enthält gültigen, ungenutzten, nicht abgelaufenen Eintrag |
| **Aktion** | POST `{ token, first_name, last_name, password }` |
| **Erwartetes Ergebnis** | HTTP 201, neuer `users`-Eintrag mit korrekter Rolle aus Token, `invitation_tokens.used_at` gesetzt |

**Zusatzfall:** Enthält der Token eine `member_id` (Einladung für bekanntes Mitglied), wird `members.user_id` auf den neuen User aktualisiert — aber nur wenn `members.user_id IS NULL`.

---

### TC-A09 — Registrierung mit abgelaufenem Token 🔲

**Route:** `POST /api/auth/register`

| | |
|---|---|
| **Vorbedingung** | `invitation_tokens` mit `expires_at` in der Vergangenheit |
| **Aktion** | POST mit dem (abgelaufenen) Token |
| **Erwartetes Ergebnis** | HTTP 400 (`"invalid or expired token"`) |

---

### TC-A10 — Registrierung mit bereits benutztem Token 🔲

**Route:** `POST /api/auth/register`

| | |
|---|---|
| **Vorbedingung** | `invitation_tokens` mit `used_at IS NOT NULL` |
| **Aktion** | POST mit dem schon benutzten Token |
| **Erwartetes Ergebnis** | HTTP 400 (Query filtert `used_at IS NULL`) |

---

### TC-A11 — Passwort-Reset: Token anlegen (ForgotPassword immer 204) 🔲

**Route:** `POST /api/auth/forgot-password`

| | |
|---|---|
| **Vorbedingung A** | Bekannte E-Mail-Adresse eines Nutzers mit `can_login = 1` |
| **Erwartetes Ergebnis A** | HTTP 204, `password_reset_tokens`-Eintrag angelegt |
| **Vorbedingung B** | Unbekannte E-Mail-Adresse |
| **Erwartetes Ergebnis B** | **Ebenfalls HTTP 204** — Anti-Enumeration-Design. Kein Token angelegt. |

---

### TC-A12 — Passwort-Reset: Neues Passwort setzen 🔲

**Route:** `POST /api/auth/reset-password`

| | |
|---|---|
| **Vorbedingung** | Gültiger, ungenutzter Reset-Token; User hat aktive Sessions (Refresh-Tokens in DB) |
| **Aktion** | POST `{ token, password: "NeuesPasswort123" }` |
| **Erwartetes Ergebnis** | HTTP 204, Passwort in DB geändert, `password_reset_tokens.used_at` gesetzt, **alle** `refresh_tokens` des Users gelöscht |

---

### TC-A13 — Passwort-Reset: Abgelaufener Token 🔲

**Route:** `POST /api/auth/reset-password`

| | |
|---|---|
| **Vorbedingung** | Reset-Token mit `expires_at` in Vergangenheit |
| **Aktion** | POST mit abgelaufenem Token |
| **Erwartetes Ergebnis** | HTTP 400 |

---

### TC-A14 — Nur Admin kann Admin-Rolle vergeben 🔲

**Route:** `PUT /api/admin/users/{id}/role`

⚠️ **Hinweis:** `UpdateUserRole` akzeptiert nur `"admin"` oder `"standard"` als Wert — nicht `"trainer"`, `"vorstand"` etc. (Diese Zuordnung erfolgt über `member_club_functions`, nicht über den `users.role`-Wert.)

| | |
|---|---|
| **Fall A** | Admin setzt Rolle auf `"admin"` → HTTP 204 |
| **Fall B** | Nicht-Admin versucht, Rolle auf `"admin"` zu setzen → HTTP 403 |
| **Fall C** | Ungültige Rolle (z.B. `"trainer"`) → HTTP 400 |
| **Fall D** | User-ID nicht vorhanden → HTTP 404 |

---

### TC-A15 — Nutzer löschen: Selbstlöschung verboten, Cascade 🔲

**Route:** `DELETE /api/admin/users/{id}`

| | |
|---|---|
| **Fall A** | Admin versucht eigenes Konto zu löschen → HTTP 400 (`"cannot delete your own account"`) |
| **Fall B** | Admin löscht fremdes Konto → HTTP 204; `refresh_tokens`, `family_links`, `duty_assignments`, `duty_accounts` des Nutzers sind ebenfalls entfernt (Transaktion) |

---

## Paket: `duties`

### Hintergrund

Das Duties-Paket verwaltet Diensttypen, Dienst-Slots, Zuweisungen (Assignments) und Dienstkonten (Accounts).

**Fachliche Grundregeln:**
- Ein Slot hat `slots_total` und `slots_filled`. Claim erhöht `slots_filled`, Unclaim verringert es.
- Geclaimte Slots haben Status `"pending"`. Fulfillierte Slots haben Status `"fulfilled"`.
- Ein erfüllter Slot kann nicht mehr unclaimed werden.
- `Claim` legt automatisch einen `duty_accounts`-Eintrag an, falls noch keiner für die aktive Saison existiert.
- `Fulfill` und `CashSubstitute` aktualisieren **nicht direkt** `duty_accounts.ist` — dieser Wert wird von der Game-Delete-Logik bei Kaskadenlöschung neu berechnet. Im normalen Betrieb ist `ist` eine manuell gepflegte Größe.
- Jeder über `CreateSlot` oder `UpdateSlot` manuell angelegte/bearbeitete Slot bekommt `is_custom = 1` gesetzt.
- `DeleteSlot` benachrichtigt alle bereits eingetragenen Nutzer per Push-Notification.
- Board-Sichtbarkeit: `audiences = NULL` bedeutet sichtbar für alle. Ist ein Audience-Array gesetzt, sieht nur die passende Gruppe den Slot.
- Audience-Bypass: Admins und Nutzer mit Vereinsfunktionen `vorstand`, `vorstand_beisitzer` oder `trainer` sehen alle Slots unabhängig vom Audience-Filter.

---

### TC-D01 — Freien Slot claimen 🔲

**Route:** `POST /api/duty-board/{slotId}/claim`

| | |
|---|---|
| **Vorbedingung** | Slot mit `slots_total = 2`, `slots_filled = 0` |
| **Aktion** | Eingeloggter User postet ohne Body |
| **Erwartetes Ergebnis** | HTTP 204 |
| **DB-Seiteneffekte** | `duty_assignments` enthält Eintrag mit `status = "pending"`; `duty_slots.slots_filled = 1` |
| **Seiteneffekt Konto** | `duty_accounts` für diesen User in aktiver Saison existiert danach (ggf. neu angelegt mit `soll=0, ist=0`) |

---

### TC-D02 — Vollen Slot claimen 🔲

**Route:** `POST /api/duty-board/{slotId}/claim`

| | |
|---|---|
| **Vorbedingung** | Slot mit `slots_total = 1`, `slots_filled = 1` |
| **Aktion** | User versucht zu claimen |
| **Erwartetes Ergebnis** | HTTP 409 Conflict (`"slot full or not found"`) |

---

### TC-D03 — Denselben Slot zweimal claimen 🔲

**Route:** `POST /api/duty-board/{slotId}/claim`

| | |
|---|---|
| **Vorbedingung** | User hat den Slot bereits geclaimt (`duty_assignments` Eintrag existiert) |
| **Aktion** | User versucht erneut zu claimen |
| **Erwartetes Ergebnis** | HTTP 409 (`"already claimed"`) — verletzt UNIQUE-Constraint `(duty_slot_id, user_id)` |

---

### TC-D04 — Slot freigeben (Unclaim) 🔲

**Route:** `DELETE /api/duty-board/{slotId}/claim`

| | |
|---|---|
| **Vorbedingung** | User hat Slot mit `status = "pending"` geclaimt |
| **Aktion** | DELETE |
| **Erwartetes Ergebnis** | HTTP 204 |
| **DB-Seiteneffekte** | `duty_assignments`-Eintrag gelöscht; `duty_slots.slots_filled` um 1 verringert |

---

### TC-D05 — Fulfillierter Slot kann nicht unclaimed werden 🔲

**Route:** `DELETE /api/duty-board/{slotId}/claim`

| | |
|---|---|
| **Vorbedingung** | Assignment für diesen User hat `status = "fulfilled"` |
| **Aktion** | DELETE |
| **Erwartetes Ergebnis** | HTTP 409 Conflict (`"already fulfilled"`) |

---

### TC-D06 — Unclaim ohne vorherige Zuweisung 🔲

**Route:** `DELETE /api/duty-board/{slotId}/claim`

| | |
|---|---|
| **Vorbedingung** | Slot existiert, aber kein `duty_assignments`-Eintrag für diesen User |
| **Aktion** | DELETE |
| **Erwartetes Ergebnis** | HTTP 404 |

---

### TC-D07 — Elternteil claimt für Proxy-Kind 🔲

**Route:** `POST /api/duty-board/{slotId}/claim`

**Hinweis:** Ein Proxy-Kind hat `can_login = 0` und ist über `family_links` mit dem Elternteil verknüpft. Nur für solche Proxy-Kinder darf ein Elternteil einen Slot claimen.

| | |
|---|---|
| **Vorbedingung** | Elternteil-User verknüpft mit Kind-User (kind hat `can_login = 0`) via `family_links` |
| **Aktion** | POST mit Body `{ "user_id": <kind_user_id> }` |
| **Erwartetes Ergebnis** | HTTP 204, Assignment für Kind-User angelegt |

---

### TC-D08 — Claim für fremden User ohne Proxy-Berechtigung 🔲

**Route:** `POST /api/duty-board/{slotId}/claim`

| | |
|---|---|
| **Vorbedingung** | User B existiert, ist aber kein Proxy-Kind von User A |
| **Aktion** | User A POST mit Body `{ "user_id": <user_b_id> }` |
| **Erwartetes Ergebnis** | HTTP 403 Forbidden |

---

### TC-D09 — Admin sieht alle Slots der aktiven Saison (Board) 🔲

**Route:** `GET /api/duty-board`

| | |
|---|---|
| **Vorbedingung** | Aktive Saison mit 5 Slots für verschiedene Teams |
| **Aktion** | Admin GET /duty-board |
| **Erwartetes Ergebnis** | HTTP 200, alle 5 Slots im Response |

---

### TC-D10 — Normaler User sieht nur Slots seines Teams 🔲

**Route:** `GET /api/duty-board`

| | |
|---|---|
| **Vorbedingung** | User ist Mitglied in Team A (`player_memberships`), nicht in Team B. Je 3 Slots für Team A und Team B existieren. |
| **Aktion** | GET /duty-board |
| **Erwartetes Ergebnis** | Nur die 3 Slots von Team A; Team-B-Slots sind nicht enthalten |

---

### TC-D11 — Audience-Filter: eltern-Slot für Elternteil sichtbar 🔲

**Route:** `GET /api/duty-board`

| | |
|---|---|
| **Vorbedingung** | Slot mit `audiences = ["eltern"]`, User hat mindestens einen Eintrag in `family_links` |
| **Aktion** | GET /duty-board |
| **Erwartetes Ergebnis** | Slot ist enthalten |

---

### TC-D12 — Audience-Filter: eltern-Slot für User ohne Kinder unsichtbar 🔲

**Route:** `GET /api/duty-board`

| | |
|---|---|
| **Vorbedingung** | Slot mit `audiences = ["eltern"]`, User hat **keine** Einträge in `family_links` |
| **Aktion** | GET /duty-board |
| **Erwartetes Ergebnis** | Slot ist **nicht** enthalten |

---

### TC-D13 — Audience-Bypass für Trainer 🔲

**Route:** `GET /api/duty-board`

**Hinweis:** User mit Vereinsfunktion `trainer` sieht alle Slots, unabhängig vom Audience-Filter.

| | |
|---|---|
| **Vorbedingung** | Slot mit `audiences = ["eltern"]`, User hat Eintrag in `member_club_functions` mit `function = "trainer"` |
| **Aktion** | GET /duty-board |
| **Erwartetes Ergebnis** | Slot ist enthalten (Bypass greift) |

---

### TC-D14 — view=mine zeigt nur eigene Slots 🔲

**Route:** `GET /api/duty-board?view=mine`

| | |
|---|---|
| **Vorbedingung** | 5 Slots existieren, User hat 2 davon geclaimt |
| **Aktion** | GET /duty-board?view=mine |
| **Erwartetes Ergebnis** | Genau 2 Slots im Response |

---

### TC-D15 — Dienstkonten: Admin sieht alle, User nur eigene 🔲

**Route:** `GET /api/duty-accounts`

| | |
|---|---|
| **Vorbedingung** | 3 `duty_accounts`-Einträge für 3 verschiedene User |
| **Fall A** | Admin-Request → alle 3 Einträge, jeder mit `balance = soll - ist` |
| **Fall B** | Standard-User-Request → nur der eigene Eintrag |

---

### TC-D16 — Slot anlegen setzt is_custom = 1 🔲

**Route:** `POST /api/duty-slots`

| | |
|---|---|
| **Aktion** | Trainer POST mit Slot-Daten |
| **Erwartetes Ergebnis** | HTTP 201; `duty_slots.is_custom = 1` für den neuen Slot |

**Fachliche Bedeutung:** `is_custom = 1` schützt den Slot vor automatischer Überschreibung durch Auto-Regen.

---

### TC-D17 — Slot bearbeiten setzt is_custom = 1 🔲

**Route:** `PUT /api/duty-slots/{id}`

| | |
|---|---|
| **Vorbedingung** | Slot mit `is_custom = 0` (auto-generiert) |
| **Aktion** | Trainer PUT mit geänderten Feldern |
| **Erwartetes Ergebnis** | HTTP 204; `duty_slots.is_custom = 1` danach |

---

### TC-D18 — Slot löschen benachrichtigt eingetragene User 🔲

**Route:** `DELETE /api/duty-slots/{id}`

| | |
|---|---|
| **Vorbedingung** | Slot mit 2 eingetragenen Usern (pending Assignments), `push_subscriptions` für diese User vorhanden |
| **Aktion** | Trainer DELETE |
| **Erwartetes Ergebnis** | HTTP 204; Slot gelöscht; beide User haben Push-Notification erhalten (`notify.Send` wurde mit beiden User-IDs aufgerufen) |

---

## Paket: `members`

### Hintergrund

Das Members-Paket verwaltet Mitglieds-Stammdaten, Familien-Links und Proxy-Accounts.

**Fachliche Grundregeln:**
- Ausgetretene Mitglieder (`status = 'ausgetreten'`) erscheinen in der Mitgliederliste **nicht** mehr.
- `wideSearch` (alle Mitglieder sehen): Admin, Vorstand, `sportliche_leitung`, oder Trainer der gezielt nach Trainern sucht (`?club_function=trainer`). Alle anderen: nur Mitglieder des eigenen Teams.
- Suche ist serverseitig: Name, Position, Passnummer, Trikotnummer, Adresse, Status, E-Mail.
- `CreateFamilyLink` erlaubt maximal 2 Erziehungsberechtigte pro Mitglied (HTTP 409 bei Überschreitung).
- Doppeltes Anlegen desselben Links wird durch `INSERT OR IGNORE` silently ignoriert (kein Fehler).
- `CreateProxyAccount` erstellt einen User mit `can_login = 0` und leerem Passwort. Ist das Mitglied schon mit einem User verknüpft, schlägt der Aufruf mit HTTP 409 fehl.
- `DeleteFamilyLink` gibt HTTP 404 zurück, wenn der Link nicht existiert.

---

### TC-M01 — Mitgliederliste: Paginierung 🔲

**Route:** `GET /api/members?limit=10&offset=10`

| | |
|---|---|
| **Vorbedingung** | 25 aktive Mitglieder in DB |
| **Aktion** | Vorstand GET mit `limit=10&offset=10` |
| **Erwartetes Ergebnis** | 10 Einträge im `items`-Array, `total = 25` |

---

### TC-M02 — Mitgliederliste: Suche nach Name 🔲

**Route:** `GET /api/members?search=...`

| | |
|---|---|
| **Vorbedingung** | Mitglied "Anna Müller" und Mitglied "Karl Schmidt" |
| **Aktion** | GET `?search=müller` |
| **Erwartetes Ergebnis** | Nur "Anna Müller" im Result |

---

### TC-M03 — Mitgliederliste: Ausgetretene Mitglieder nicht sichtbar 🔲

**Route:** `GET /api/members`

| | |
|---|---|
| **Vorbedingung** | 3 Mitglieder: 2 mit `status = "aktiv"`, 1 mit `status = "ausgetreten"` |
| **Aktion** | Vorstand GET /api/members |
| **Erwartetes Ergebnis** | Nur 2 Mitglieder im Result |

---

### TC-M04 — Mitgliederliste: Trainer sieht nur eigenes Team 🔲

**Route:** `GET /api/members`

| | |
|---|---|
| **Vorbedingung** | Trainer ist über `kader_trainers` mit Kader von Team A verknüpft. Team A hat 3 Mitglieder, Team B hat 2 Mitglieder. |
| **Aktion** | Trainer GET /api/members |
| **Erwartetes Ergebnis** | 3 Mitglieder (nur Team A) |

---

### TC-M05 — Familienlink anlegen: Erfolgsfall 🔲

**Route:** `POST /api/admin/family-links`

| | |
|---|---|
| **Vorbedingung** | Elternteil-User und Mitglied existieren; Mitglied hat noch keine Eltern verknüpft |
| **Aktion** | POST `{ parent_user_id, member_id }` |
| **Erwartetes Ergebnis** | HTTP 204, `family_links`-Eintrag existiert |

---

### TC-M06 — Familienlink: Maximal 2 Erziehungsberechtigte 🔲

**Route:** `POST /api/admin/family-links`

| | |
|---|---|
| **Vorbedingung** | Mitglied hat bereits 2 Elternteile verknüpft |
| **Aktion** | Versuch, dritten Elternteil hinzuzufügen |
| **Erwartetes Ergebnis** | HTTP 409 (`"maximal zwei Erziehungsberechtigte erlaubt"`) |

---

### TC-M07 — Familienlink: Duplikat wird ignoriert 🔲

**Route:** `POST /api/admin/family-links`

| | |
|---|---|
| **Vorbedingung** | Familienlink zwischen Elternteil A und Kind B existiert bereits |
| **Aktion** | Gleiche POST-Anfrage erneut |
| **Erwartetes Ergebnis** | HTTP 204 (kein Fehler, `INSERT OR IGNORE`), weiterhin genau 1 Eintrag |

---

### TC-M08 — Familienlink löschen: Nicht existierender Link 🔲

**Route:** `DELETE /api/admin/family-links`

| | |
|---|---|
| **Aktion** | DELETE `{ parent_user_id, member_id }` für nicht existierenden Link |
| **Erwartetes Ergebnis** | HTTP 404 |

---

### TC-M09 — Proxy-Account anlegen 🔲

**Route:** `POST /api/admin/members/{id}/proxy-account`

| | |
|---|---|
| **Vorbedingung** | Mitglied ohne verknüpften User (`user_id IS NULL`) |
| **Aktion** | POST `{ "email": null }` |
| **Erwartetes Ergebnis** | HTTP 201, Response enthält `user_id`; neuer User hat `can_login = 0`; `members.user_id` auf neuen User gesetzt |

---

### TC-M10 — Proxy-Account ablehnen wenn Mitglied schon Account hat 🔲

**Route:** `POST /api/admin/members/{id}/proxy-account`

| | |
|---|---|
| **Vorbedingung** | Mitglied hat bereits `user_id IS NOT NULL` |
| **Aktion** | POST |
| **Erwartetes Ergebnis** | HTTP 409 (`"member already has an account"`) |

---

---

## Paket: `kader` (Handler)

### Hintergrund

Die reine Age-Bracket-Logik ist bereits getestet. Die Handler-Logik für `AutoAssign` und `MemberSuggestions` ist bisher ungetestet. Beide Funktionen arbeiten mit DHB-Jahrgangsbrackets und Geschlechtsfiltern.

**Fachliche Grundregeln:**
- `AutoAssign` weist alle passenden aktiven Mitglieder (`status != 'ausgetreten'`) einem Kader zu, die ins Jahrgangsbracket der Altersklasse passen.
- Bei Kader mit `dedicated_birth_year` wird exakt nach diesem Jahrgang gefiltert (nicht nach Bracket).
- Geschlechter: Kader `"m"` enthält Mitglieder mit `gender = "m"` **oder** `gender = "u"` (unbekannt). Kader `"mixed"` enthält alle.
- `MemberSuggestions` liefert max. 20 Vorschläge; mit `filter_age_bracket=false` wird der Bracket-Filter deaktiviert.
- `already_in_kader = true` wenn das Mitglied bereits im Kader ist.

---

### TC-K01 — AutoAssign: Mitglieder im korrekten Bracket werden zugewiesen 🔲

**Route:** `POST /api/admin/kader/auto-assign`

| | |
|---|---|
| **Vorbedingung** | Saison 2025/26; Kader A-Jugend; 2 Mitglieder Jg. 2007/2008 (im Bracket), 1 Mitglied Jg. 2005 (außerhalb) |
| **Aktion** | POST `{ kader_ids: [<id>] }` |
| **Erwartetes Ergebnis** | HTTP 200; `kader_members` enthält genau 2 Einträge |

---

### TC-K02 — AutoAssign: Ausgetretene Mitglieder werden nicht zugewiesen 🔲

| | |
|---|---|
| **Vorbedingung** | 1 Mitglied im Bracket mit `status = "ausgetreten"`, 1 mit `status = "aktiv"` |
| **Erwartetes Ergebnis** | Nur das aktive Mitglied wird zugewiesen |

---

### TC-K03 — AutoAssign: Kader mit dedicated_birth_year 🔲

| | |
|---|---|
| **Vorbedingung** | Kader mit `dedicated_birth_year = 2008`; Mitglieder Jg. 2007, 2008, 2009 |
| **Erwartetes Ergebnis** | Nur Jg. 2008 wird zugewiesen (exakter Jahrgang, kein Bracket) |

---

### TC-K04 — MemberSuggestions: Bracket-Filter aktiv 🔲

**Route:** `GET /api/admin/kader/{id}/member-suggestions`

| | |
|---|---|
| **Vorbedingung** | Kader A-Jugend Saison 2025/26; 1 Mitglied Jg. 2007 (im Bracket), 1 Mitglied Jg. 2005 (außerhalb) |
| **Aktion** | GET ohne `filter_age_bracket=false` |
| **Erwartetes Ergebnis** | Nur das Mitglied Jg. 2007 in `suggestions`; `already_in_kader = false` für beide |

---

### TC-K05 — MemberSuggestions: Bracket-Filter deaktiviert 🔲

| | |
|---|---|
| **Vorbedingung** | Wie TC-K04 |
| **Aktion** | GET `?filter_age_bracket=false` |
| **Erwartetes Ergebnis** | Beide Mitglieder in `suggestions` |

---

## Paket: `games` (Erweiterung)

### TC-G-EXT01 — ListTeamsForUser: Trainer sieht nur eigene Teams 🔲

**Route:** `GET /api/teams`

| | |
|---|---|
| **Vorbedingung** | Trainer mit Kader-Verknüpfung für Team A, kein Kader für Team B |
| **Erwartetes Ergebnis** | Nur Team A im Response |

---

### TC-G-EXT02 — ListTeamsForUser: Admin sieht alle Teams 🔲

| | |
|---|---|
| **Erwartetes Ergebnis** | Alle Teams im Response inkl. inaktiver Teams |

---

### TC-G-EXT03 — ListTeamsForUser: Spieler sieht nur eigene Teams 🔲

| | |
|---|---|
| **Vorbedingung** | Spieler ist über `team_memberships` in Team A, nicht in Team B |
| **Erwartetes Ergebnis** | Nur Team A |

---

## Paket: `trainings` (Erweiterung)

### TC-T-EXT01 — GetAttendances: Gespeicherte Anwesenheiten lesen 🔲

**Route:** `GET /api/training-sessions/{id}/attendances`

| | |
|---|---|
| **Vorbedingung** | Session mit 2 Mitgliedern, 1 davon als `present = true` eingetragen |
| **Erwartetes Ergebnis** | Response enthält 1 Eintrag mit `present = true` |

---

### TC-T-EXT02 — Respond: Elternteil antwortet für Kind 🔲

**Route:** `POST /api/training-sessions/{id}/respond`

| | |
|---|---|
| **Vorbedingung** | Elternteil-Token; Mitglied via `family_links` verknüpft |
| **Aktion** | POST `{ status: "confirmed", member_id: <kind_id> }` |
| **Erwartetes Ergebnis** | HTTP 204; `training_responses` Eintrag für das Kind angelegt (nicht für Elternteil) |

---

## Paket: `absences` (Erweiterung)

### TC-AB-EXT01 — Unauthorisierter Zugriff: Fremdes Mitglied 🔲

**Route:** `POST /api/absences`

| | |
|---|---|
| **Vorbedingung** | User A versucht, Abwesenheit für Mitglied B zu erstellen; kein Familienlink |
| **Aktion** | POST mit `member_ids: [B]` |
| **Erwartetes Ergebnis** | HTTP 403 oder das Mitglied wird aus der Verarbeitung ausgeschlossen |

---

### TC-AB-EXT02 — Preview: Keine Abwesenheiten im Zeitraum 🔲

**Route:** `GET /api/absences/preview`

| | |
|---|---|
| **Vorbedingung** | Kind ohne Trainings oder Spiele im Zeitraum |
| **Erwartetes Ergebnis** | HTTP 200, leeres Array |

---

## Paket: `chat` (Erweiterung)

### TC-CH-EXT01 — LeaveConversation: Mitglied verlässt Gruppe 🔲

**Route:** `DELETE /api/chat/conversations/{id}/members/me`

| | |
|---|---|
| **Vorbedingung** | User ist normales Mitglied (nicht Creator) einer Gruppe |
| **Aktion** | DELETE |
| **Erwartetes Ergebnis** | HTTP 204; `left_at` gesetzt; System-Nachricht "hat die Gruppe verlassen" |

---

### TC-CH-EXT02 — LeaveConversation: Direkt-Chat kann nicht verlassen werden 🔲

⚠️ **Hinweis:** `LeaveConversation` prüft NICHT, ob der User Creator ist — auch der Creator kann eine Gruppe verlassen. Nur Direkt-Conversations können nicht verlassen werden.

| | |
|---|---|
| **Vorbedingung** | User ist Mitglied einer Direkt-Conversation |
| **Aktion** | DELETE /api/chat/conversations/{id}/members/me |
| **Erwartetes Ergebnis** | HTTP 400 (`"cannot leave direct conversation"`) |

---

### TC-CH-EXT03 — CreateConversation: Direkt-Chat doppelt anlegen 🔲

**Route:** `POST /api/chat/conversations`

| | |
|---|---|
| **Vorbedingung** | Direkt-Conversation zwischen User A und B existiert bereits |
| **Aktion** | POST erneut für gleiche Partner |
| **Erwartetes Ergebnis** | Bestehende Conversation zurückgegeben (kein Duplikat) |

---

## Bereits implementierte Tests (Referenz)

### Paket: `kader`

| Test | Beschreibung |
|------|---|
| ✅ `TestComputeAgeBrackets` | DHB-konforme Altersklassen-Jahrgangsbereiche für Saison 2024–2026 |
| ✅ `TestBirthYearInBracket` | Grenzwerte: In-Bracket, Out-of-Bracket, unbekannte Klasse |
| ✅ `TestNoBracketOverlap` | Keine Überschneidungen zwischen Altersklassen in einer Saison |

### Paket: `games`

| Test | Beschreibung |
|------|---|
| ✅ `TestListGames_ReturnsGamesInRange` | Spiele der aktiven Saison werden zurückgegeben |
| ✅ `TestListGames_EmptyRange` | Leere Liste wenn keine Spiele |
| ✅ `TestCreateGame_AdminOK` | Admin kann Spiel anlegen (HTTP 201) |
| ✅ `TestCreateGame_UnauthorizedForbidden` | Nicht-autorisierter User erhält HTTP 403 |
| ✅ `TestCreateGame_ResponseIncludesRegenSummary` | Response enthält `regen_summary` nach Auto-Regen |
| ✅ `TestCreateGame_AutoRegenSkipsAdjacentDay` | adjacent_day_behavior=skip verhindert Slot auf Vortag |
| ✅ `TestCreateGame_CustomSlotNotAffectedByRegen` | `is_custom=1` Slots bleiben bei Auto-Regen unverändert |
| ✅ `TestCreateGame_GenericEventCanBeCreated` | Generisches Event ohne Template |
| ✅ `TestUpdateGame_TimeChangeRegenSlots` | Zeitänderung verschiebt Template-Slots |
| ✅ `TestDeleteGame_CascadesAndRollsBackFulfilledHours` | Löschen rollt erfüllte Stunden im Konto zurück |
| ✅ `TestDeleteGame_NoDutiesNoCrash` | Spiel ohne Slots kann sauber gelöscht werden |
| ✅ `TestDeleteGame_NeighborDayRegen` | Nach Löschen wird Nachbartagslot erzeugt (skip aufgehoben) |

---

## Offene Fragen (zur fachlichen Klärung)

Diese Punkte sind beim Lesen des Codes aufgefallen und sollten vor der Test-Implementierung geklärt werden:

1. **`duty_accounts.ist` wird nicht automatisch aktualisiert:** `Fulfill()` setzt nur `duty_assignments.status = 'fulfilled'`. Der `ist`-Wert in `duty_accounts` wird nur bei Game-Delete neu berechnet. Ist das gewolltes Verhalten? Gibt es einen Scheduler-Job, der `ist` periodisch synchronisiert?

2. **`UpdateUserRole` akzeptiert nur `"admin"` oder `"standard"`:** Vereinsfunktionen wie `"trainer"`, `"vorstand"` werden über `member_club_functions` verwaltet, nicht über `users.role`. Ist das für alle Nutzer, die die Seite `/admin/users` bedienen, klar?

3. **`DeleteSlot` prüft keine Rolle:** Jeder eingeloggte User könnte in der Route einen Slot löschen, falls er die ID kennt. Ist die Autorisierung im Router (Middleware) sichergestellt?
