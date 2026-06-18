# Manueller Testplan — permissions-cleanup (Frontend-Capabilities)

Verifiziert, dass die Sichtbarkeit von Buttons, Tabs und Nav-Einträgen jetzt
serverseitig aus `GET /api/me` (`capabilities` + `nav`) bzw. aus per-Item
`can.*` kommt — und **nicht** mehr aus `user.role`/`clubFunctions` im Frontend.

Automatisiert bereits grün: `go test ./...`, 318 Frontend-Tests
(`pnpm test`, inkl. `AppShell.permissions.test.tsx`), `tsc -b`.
Dieser Plan deckt das ab, was nur im echten Browser sichtbar wird.

---

## 1. Vorbereitung

### Lokaler Start
```bash
go run ./cmd/teamwerk          # Terminal 1 (:8080)
cd web && pnpm dev             # Terminal 2 (:5173)
```
Öffnen: http://localhost:5173

### Personas durchspielen
Empfohlen: als **Admin** einloggen und über **Impersonation** in die jeweilige
Person wechseln (Admin-only: `POST /api/impersonate/{id}`, im UI über die
Nutzerverwaltung / „Anmelden als"). Nach jedem Wechsel prüfen:

> **Wichtig:** Nach Login / Token-Refresh / Impersonation-Start **und** -Stop
> muss die UI die Capabilities neu laden. Konkret heißt das: kurz nach dem
> Wechsel sollten die unten erwarteten Buttons/Tabs/Nav-Einträge passen — ohne
> manuelles Reload. (AuthContext ruft `/api/me` bei jedem User-Wechsel neu auf.)

Benötigte Testpersonen (je eine mit genau dieser Vereinsfunktion):

| Persona | Rolle / Funktion |
|---|---|
| Admin | `role = admin` |
| Vorstand | `vorstand` (ohne trainer/sl) |
| Trainer | `trainer` |
| Sportliche Leitung | `sportliche_leitung` |
| Spieler | `spieler` |
| Elternteil | kein Funktions-Tag, `is_parent = true` |
| Vorstand + Elternteil | `vorstand`, `is_parent = true` |

---

## 2. Smoke-Checks (alle Personas)

- [ ] Nach Login keine Konsolenfehler (`/api/me` lädt, Netzwerk-Tab: 200).
- [ ] Sidebar erscheint, kein „alle Punkte blitzen kurz auf und verschwinden"
      über mehrere Sekunden (kurzes Fallback-Aufblitzen beim allerersten Laden
      ist ok — danach exakte Sichtbarkeit).
- [ ] Logout → erneuter Login als andere Persona zeigt **deren** Sidebar
      (kein Nachhängen der alten Capabilities).

---

## 3. Sidebar-Navigation (`policy.NavFor`)

Für jede Persona prüfen, dass **genau** diese Verwaltungs-Punkte sichtbar sind
(immer sichtbar bei allen: Dashboard, Kalender, Termine, Mein Team, Dokumente,
Dienste, Mitfahrten, Nachrichten; „Mein Profil" bei allen **außer Admin**):

| Nav-Punkt | sichtbar für |
|---|---|
| Mitglieder | Admin, Vorstand |
| Nutzerverwaltung | Admin, Vorstand |
| Diensttypen, Dienstplan-Vorlagen, Veranstaltungsorte, Einstellungen | Admin, Vorstand |
| Kader | Admin, Vorstand, Trainer, Sportliche Leitung |

- [ ] **Spieler / Elternteil / Beisitzer / Kassierer:** Modul „Verwaltung" fehlt komplett.
- [ ] **Trainer / Sportliche Leitung:** nur „Kader" unter Verwaltung, **kein** „Mitglieder".
- [ ] **Admin:** sieht „Mitglieder" etc., aber **kein** „Mein Profil".

---

## 4. Capability-gesteuerte UI

Legende der erwarteten Sichtbarkeit (`A`=Admin, `V`=Vorstand, `T`=Trainer,
`SL`=Sportliche Leitung, `S`=Spieler, `E`=Elternteil):

### 4.1 Mitglieder (`manage_members` → A, V)
Seite `/mitglieder` ist ohnehin nur für A/V nav-sichtbar; falls direkt
aufgerufen:
- [ ] **A, V:** „+ Neu"-Button, Import/Export-Menü, Filter „Ohne App-Account" /
      „Mit Änderungsantrag", Aktions-Spalte vorhanden.
- [ ] In der Liste: Stift-/Papierkorb-Icon pro Zeile erscheint gemäß
      `member.can.edit` / `member.can.delete` (für A/V bei allen Zeilen).
- [ ] **Mitglied-Detail (`/mitglieder/:id`):** Tabs „Datenschutz", „Familie",
      „Admin" nur für A/V sichtbar.

### 4.2 SEPA-Mandat löschen (`manage_members` **oder** Elternteil)
Mitglied-Detail → Tab Datenschutz → SEPA-Dokument:
- [ ] **A, V:** „Löschen"-Button am SEPA-Dokument sichtbar.
- [ ] **Elternteil** (beim eigenen Kind): Löschen-Button sichtbar.
- [ ] **Spieler (fremd):** nicht sichtbar.

### 4.3 Kalender — Event-Verwaltung (`manage_games` → A, V, T, SL)
Seite `/kalender`:
- [ ] **A, V, T, SL:** „Event anlegen"-Button (+) sichtbar; Team-Abwesenheiten
      einblendbar; Events editierbar (Stift im Info-Popover).
- [ ] **Spieler / Elternteil:** statt „Event" nur „Abwesenheit eintragen"
      (sofern Spieler oder Elternteil); keine Team-Abwesenheiten-Ansicht.

> **Regressionsfokus:** **Vorstand** soll hier jetzt Events anlegen/bearbeiten
> können (vorher im Frontend teilweise ausgeblendet). Bitte gezielt prüfen,
> dass das gewollt ist und funktioniert.

### 4.4 Kalender — Training anlegen (`manage_trainings` → A, T, SL; **nicht** reiner Vorstand)
Kalender → „Event anlegen" → Auswahl der Event-Art:
- [ ] **A, T, SL:** Option „Training" im Wizard vorhanden.
- [ ] **Reiner Vorstand (ohne trainer/sl):** Option „Training" **fehlt**
      (Spiel-Optionen bleiben sichtbar).

### 4.5 Termine — Anwesenheit/Aufstellung (`manage_trainings` + per-Item `can.manage_lineup`)
Seite `/termine` und Detail `/termine/:type/:id`:
- [ ] **Spieler / Elternteil:** sehen die RSVP-/Zu-/Absage-Buttons (sie „antworten").
- [ ] **Trainer / SL / Admin:** sehen statt RSVP die Trainer-Ansicht
      (Anwesenheiten verwalten); bei **Spielen** zusätzlich Aufstellung
      (`can.manage_lineup`).
- [ ] **Reiner Vorstand:** bei **Trainings** keine Trainer-Verwaltung
      (manage_trainings fehlt). Bei **Spielen** richtet sich „Aufstellung
      verwalten" nach `game.can.manage_lineup` (kommt vom Server).

### 4.6 Dienste-Board (`/dienste`)
Zwei getrennte Dinge nicht verwechseln:

**Dienst übernehmen & erfüllen (alle eingeloggten Nutzer, inkl. Vorstand):**
- [ ] **Alle (Spieler, Elternteil, Vorstand, Trainer, …):** an einem offenen Slot
      „Eintragen" sichtbar; danach „Austragen". So bekommt z.B. ein **Vorstand**
      seinen Kasse-/Einkauf-Dienst und erfüllt ihn — dieser Ablauf ist nicht
      capability-gegated.

**Slot-Verwaltung — Bearbeiten/Löschen (`manage_duties` → A, V, T, SL):**
- [ ] **A, V, T, SL:** Löschen-Aktion (Papierkorb) an Slots sichtbar.
- [ ] **Spieler / Elternteil / Beisitzer / Kassierer:** keine Löschen-Aktion.

> **Regressionsfokus (behoben):** **Vorstand** sieht jetzt die Slot-Verwaltung
> (vorher Designloch — Vorstand war ausgeschlossen, obwohl das Backend
> `duty-slots`-Mutationen für Vorstand erlaubt). Bitte bestätigen.

> Hinweis: Das administrative „Erfüllt-für-andere markieren"
> (`/duty-assignments/{id}/fulfill`, Backend: Trainer + Sportliche Leitung) hat
> derzeit **keine** eigene Schaltfläche im Frontend — daher hier nichts zu testen.

### 4.7 Dokumente (`manage_documents` → **nur Admin**)
Seite `/dokumente`:
- [ ] **Admin:** Anlegen/Bearbeiten/Löschen von Ordnern & Dateien möglich
      (Schreibaktionen sichtbar).
- [ ] **Vorstand / Trainer / Spieler:** keine globalen Verwaltungs-Aktionen;
      Schreibrecht nur, wo die Ordner-ACL (`can_write`) es erlaubt.

### 4.8 Nachrichten / Broadcasts
Seite `/chat`:
- [ ] **Broadcast senden** (`broadcast_messages` → A, V, T, SL): Button sichtbar;
      **Spieler / Elternteil:** nicht sichtbar.
- [ ] **Broadcast-Zielgruppe „alle"** (`broadcast_all` → A, V): im Broadcast-Dialog
      verfügbar; Trainer/SL sehen nur die eingeschränkte Variante.
- [ ] **Fremde Nachricht löschen** (`moderate_chat` → **nur Admin**): Lösch-Aktion
      an fremden Nachrichten nur als Admin. Eigene Nachrichten kann jeder löschen.

---

## 5. Negativ-/Sicherheitscheck (Backend bleibt maßgeblich)

Die `can.*`-Flags steuern nur die **Sichtbarkeit**. Stichprobe, dass das
Backend weiterhin gatet:
- [ ] Als **Spieler** direkt `/mitglieder` in der URL aufrufen → keine
      Mitgliederliste (RoleRoute/Backend blockt), kein Datenleck.
- [ ] Als **reiner Vorstand** versuchen, ein Training anzulegen (z.B. über alte
      URL/Direktaufruf) → Server lehnt mit 403 ab (Route Trainer+SL).

---

## 6. Ergebnis festhalten

- Getestete Version / Commit: `__________`
- Browser / Gerät: `__________`
- Auffälligkeiten: `__________`

Bei Abweichung zwischen erwarteter und tatsächlicher Sichtbarkeit zuerst prüfen:
`GET /api/me` im Netzwerk-Tab → stimmen `capabilities` und `nav` für die Persona
mit der Tabelle in `openspec/specs/me-capabilities/spec.md` überein?
