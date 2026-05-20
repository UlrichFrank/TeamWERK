## Context

Die `members`-Tabelle hat bisher nur `position` (Spielposition) als beschreibendes Feld, aber keine Vereinsfunktion. Die `kader`-Tabelle hat kein Trainer-Feld. Die bestehende `users.role` ist ein Berechtigungskonzept und soll nicht für Vereinsfunktionen missbraucht werden — Berechtigungen und Vereinsrollen sind orthogonal. Die Kader-Karte hat `overflow-hidden` auf dem Container, was native Dropdowns clippt.

## Goals / Non-Goals

**Goals:**
- `members.club_function` als neues Feld: `trainer | vorstand | vorstand_beisitzer` (nullable)
- Vereinsfunktion im Mitgliederprofil (MemberDetailPage) anzeigen und bearbeiten
- Kader-Trainer werden aus Mitgliedern mit `club_function = 'trainer'` gewählt (nicht aus `users`)
- Mehrere Trainer pro Kader via Junction-Tabelle `kader_trainers`
- Dropdown-Clipping und Fokus-Bug beheben

**Non-Goals:**
- Automatische Synchronisation von `club_function` mit `users.role`
- Berechtigungsänderungen durch Vereinsfunktion
- E-Mail-Benachrichtigung bei Trainerzuweisung
- Vereinsfunktion auf Mobile-Kader-Cards (vorerst nur Desktop / Detail)

## Decisions

### D1: `club_function` als CHECK-Constraint TEXT auf `members`

Nullable TEXT mit `CHECK(club_function IN ('trainer','vorstand','vorstand_beisitzer'))`. Einfache Migration via `ALTER TABLE members ADD COLUMN`. Kein neues Lookup-Table nötig — die Werte sind stabil und überschaubar.

**Alternative:** Eigene `club_functions`-Tabelle (zu komplex für drei stabile Werte).

### D2: `kader_trainers` referenziert `members.id`, nicht `users.id`

Ein Mitglied kann Trainer sein ohne App-Account. Die Vereinsfunktion ist Mitglied-zentriert, nicht Nutzer-zentriert. Damit ist auch die Kader-Trainer-Ansicht losgelöst vom Berechtigungssystem.

Schema: `kader_trainers (kader_id FK kader, member_id FK members, PRIMARY KEY(kader_id, member_id), ON DELETE CASCADE beidseitig)`

**Alternative (abgelehnt):** `users.id` — würde Vereinsfunktion an App-Account koppeln, was der Nutzeranforderung widerspricht.

### D3: Trainer-Add-Select zeigt nur Mitglieder mit `club_function = 'trainer'`

Beim Laden der Kader-Seite: `GET /api/members?club_function=trainer` — neuer Query-Parameter. Das Ergebnis ist eine kleine Liste (typisch < 20 Personen), kein Paginierungsbedarf. Bereits zugewiesene Trainer werden client-seitig herausgefiltert.

### D4: `overflow-hidden` nur am Card-Header-Block, nicht am gesamten Container

`overflow-hidden` vom äußersten Kader-Karten-`div` entfernen. `rounded-xl` behält seine Wirkung. Die gelbe Border-t-4 clippt nicht, da sie an der oberen Kante liegt. Zusätzlich `key={k.dedicated_birth_year ?? 'empty'}` auf dem Jahrgangs-`<select>` als React-re-mount-Absicherung.

### D5: Chip-Liste + Add-Select für Trainer in Kader-Karte

Zugewiesene Trainer als Chip-Reihe (Name + ×-Button) unterhalb des Mode-Toggles. Darunter ein `<select>` "Trainer hinzufügen…" der nur noch nicht zugewiesene Mitglieder mit `club_function=trainer` anzeigt. Wählt man aus, feuert `PUT /api/admin/kader/{id}` mit `trainers_add`. Das × feuert `trainers_remove`.

### D6: Vereinsfunktion in MemberDetailPage als Select-Feld

Im Bearbeiten-Formular von `MemberDetailPage` ein neues Select `Vereinsfunktion` mit Optionen: `– keine –`, `Trainer`, `Vorstand`, `Vorstands-Beisitzer`. Speichert via `PUT /api/members/{id}` mit `club_function`.

## Risks / Trade-offs

- **Migration 015** (`ALTER TABLE members ADD COLUMN club_function`): Non-destructive, bestehende Rows → NULL.
- **Migration 016** (CREATE TABLE kader_trainers): Non-destructive, keine bestehenden Daten.
- **`role`-Parameter auf `GET /api/members`** wird zu `club_function`-Parameter — der bestehende `role`-Filter (falls vorhanden) bleibt unberührt, da es sich um `members` nicht `users` handelt.
- **Mitglieder ohne App-Account** können Trainer-Funktion tragen und Kadern zugewiesen werden — korrekt und gewollt.
