## Context

Das Berechtigungsmodell von TeamWERK kennt zwei orthogonale Dimensionen:

1. **System-Rolle (`users.role`)** — entscheidet über Plattform-weite Privilegien. Werte: `admin` (Vollzugriff, umgeht alle Vereinsfunktions-Checks) oder `standard` (alle anderen). Wird im JWT-Claim `role` mitgeführt. Geprüft via `auth.RequireRole(...)`.
2. **Vereinsfunktion (`member_club_functions.function`)** — beschreibt die *fachliche* Rolle eines Members im Verein. Werte: `spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`, `kassierer`, `sportliche_leitung`. Ein Member kann beliebig viele Funktionen haben. Wird im JWT-Claim `club_functions: string[]` mitgeführt. Geprüft via `auth.RequireClubFunction(...)` bzw. `claims.HasFunction(...)`.

Zusätzlich gibt es den Marker `isParent` im Claim — abgeleitet aus `family_links` und **keine** Vereinsfunktion.

Vor dem Refactor in Migration 002 gab es nur eine Dimension: `users.role` konnte `admin`, `vorstand`, `trainer`, `elternteil` oder `spieler` sein. Der Refactor reduzierte das auf `admin`/`standard` und führte `member_club_functions` ein. Die Daten-Migration war sauber, aber **Code, SQL, Frontend und Doku** wurden nur teilweise nachgezogen. Der Audit identifiziert die verbleibenden Altreste:

- Backend: 2 SQL-Queries (`scheduler.go`, `duties/handler.go`) suchen nach `WHERE u.role IN (alte Werte)` — liefern leere Ergebnismengen, weil nach Migration 002 niemand mehr diese `role`-Werte trägt.
- Backend: 1 Claim-Vergleich (`absences/handler.go:544`) gegen `claims.Role == "trainer"` — semantisch tot.
- Frontend: 4 Pages prüfen `user.role === '<alte rolle>'`.
- Phantom-Vereinsfunktion `sportvorstand` an 2 Stellen — existiert nirgendwo im Schema.
- DB: `duty_types.target_role`-CHECK-Constraint enthält noch alte Misch-Werte (`'admin'` ist semantisch sinnlos).
- Tests: 1 Token fixture mit ungültigem `role="elternteil"`, 1 Test schickt `{"role":"trainer"}` ohne 400 zu prüfen.
- Doku: `CLAUDE.md` beschreibt das alte 5-Rollen-Modell.

## Goals / Non-Goals

**Goals:**
- Kein Code-Pfad sucht mehr nach `role IN ('trainer'|'vorstand'|'sportliche_leitung'|'elternteil'|'spieler')`.
- Kein Code-Pfad vergleicht `claims.Role` oder `user.role` mit einer der fünf historischen Rollen.
- Die Phantom-Funktion `sportvorstand` ist nirgends mehr referenziert.
- `duty_types.target_role` CHECK akzeptiert nur valide Vereinsfunktionen + `elternteil` (Familien-Marker).
- Tests sichern die Invariante `users.role ∈ {admin, standard}` ab.
- `CLAUDE.md` dokumentiert das aktuelle Zwei-Dimensionen-Modell.

**Non-Goals:**
- **Keine** Änderung des JWT-Schemas (Claim-Felder bleiben gleich).
- **Keine** Umbenennung von `member_club_functions` oder `users.role`.
- **Keine** API-Vertragsänderungen (HTTP-Routen, Request/Response-Schemata).
- **Kein** Refactor der polymorphen `NavItem.roles`-Strings — der Mechanismus „roles-string matched entweder System-Rolle ODER Vereinsfunktion" bleibt, wird aber dokumentiert (`requirementToken` Helper). Ein vollständiger Split wäre größerer Aufwand und steht außerhalb des Scopes dieses Cleanups.
- **Keine** UI-Texte oder Übersetzungen anfassen, wo „Rolle" in Bedeutung „Vereinsfunktion" verwendet wird (das ist Wortschatz, nicht Logik).

## Decisions

### Decision 1: Eltern-Auflösung im Scheduler über `family_links`

**Wahl:** Wenn `duty_slots.target_role = 'elternteil'`, sucht der Scheduler nach Usern, die in `family_links` als `parent_user_id` eines Members mit Vereinsfunktion `spieler` im aktiven Saison-Kader des Teams stehen.

**Alternative:** Ein eigenes Boolean-Flag `users.is_parent` in der DB persistieren. **Verworfen**, weil `family_links` bereits Single Source of Truth ist und ein redundantes Flag würde drift erzeugen. Der Claim-Marker `isParent` ist nur eine *abgeleitete* JWT-Optimierung.

### Decision 2: Spieler-Auflösung via `member_club_functions`, nicht via Kader

**Wahl:** Für `target_role='spieler'` joinen wir `users → members → member_club_functions(function='spieler')`. Wenn ein Team angegeben ist, zusätzlich `→ kader_members → kader (team_id, season_id)`.

**Alternative:** Direkt über `kader_members` ohne `member_club_functions`-Check. **Verworfen**, weil ein Member im Kader auch ein Trainer-Beisitzer sein könnte (Funktions-Mehrwertigkeit). Der explizite Funktions-Filter ist sauberer und drückt die Intention aus.

### Decision 3: CHECK-Constraint-Änderung via `CREATE TABLE _new`

**Wahl:** SQLite erlaubt kein `ALTER TABLE … MODIFY CHECK`. Standard-Pattern:

```sql
CREATE TABLE duty_types_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    -- … übrige Spalten unverändert …
    target_role TEXT CHECK(target_role IN (
        'spieler','elternteil','trainer','vorstand',
        'sportliche_leitung','vorstand_beisitzer','kassierer'
    ))
);
INSERT INTO duty_types_new SELECT
    id, name, …,
    CASE target_role WHEN 'admin' THEN 'vorstand' ELSE target_role END
FROM duty_types;
DROP TABLE duty_types;
ALTER TABLE duty_types_new RENAME TO duty_types;
-- Indizes neu anlegen
```

**Alternative:** Schlicht CHECK weglassen und Validierung im Go-Handler machen. **Verworfen**, weil die DB die letzte Verteidigungslinie ist und ein DB-Test (`TestDutyType_TargetRole_RejectsAdmin`) leicht zu schreiben ist.

### Decision 4: `'elternteil'` bleibt als `target_role`-Wert erhalten

**Wahl:** `'elternteil'` ist zwar **keine** Vereinsfunktion in `member_club_functions`, bleibt aber ein gültiger `target_role`-Wert in `duty_types`. Er wird **im Scheduler-Code** in eine Query über `family_links` transformiert.

**Alternative:** `target_role` komplett auf `member_club_functions.function`-Werte einschränken und für Eltern-Dienste ein neues Feld einführen. **Verworfen**, weil die Migration komplexer würde und kein Mehrwert entstünde. `target_role` ist eine *Zielgruppen-Beschreibung*, nicht direkt eine Funktion — die Mischung ist semantisch ok, solange dokumentiert.

### Decision 5: Phantom `sportvorstand` → `vorstand`

**Wahl:** Wo `sportvorstand` geprüft wird, ersetzen wir durch `vorstand`. Begründung: Die ursprüngliche Intention war „Vorstand-Mitglied das auch sportliche Belange verantwortet". Da `sportliche_leitung` heute eine eigene Funktion ist, deckt die Kombination `vorstand` (Vereinsverwaltung) plus `sportliche_leitung` (Trainings/Sport) den Anwendungsfall ab. Die `IsTrainerLike()`-Methode reicht aus.

**Alternative:** Neue Funktion `sportvorstand` ins Schema aufnehmen. **Verworfen**, weil noch nie produktiv vergeben — würde nur den Funktions-Strauss erweitern, ohne dass ein Anwender davon profitiert.

### Decision 6: `NavItem.roles` bleibt polymorph

**Wahl:** Der bestehende AppShell-Check `user.role === r || hasFunction(user, r)` bleibt — es ist eine *Convenience*-Polymorphie, die mit den valid distinkten Werten (`admin`/`standard` vs. die sechs Funktionen) kollisionsfrei funktioniert. Wir dokumentieren das in einem JSDoc-Kommentar an `NavItem.roles` und benennen die internen Helper klarer (`matchesRequirement`).

**Alternative:** `NavItem` splittet in `systemRoles?: ('admin'|'standard')[]` und `clubFunctions?: string[]`. **Verworfen für diesen Change** als zu breite Mechanik-Änderung. Falls später gewünscht, separater Refactor.

## Risks / Trade-offs

- **[Risiko: Produktions-Daten mit `duty_types.target_role='admin'`]** → Vor `make migrate-remote-up` einmalig prüfen: `SELECT id, name, target_role FROM duty_types WHERE target_role='admin';`. Die Migration mappt automatisch auf `'vorstand'`, aber dokumentiert sollte sein, dass nach dem Upgrade dieser Slot-Typ jetzt Vorstands-Mitglieder adressiert.
- **[Risiko: Stille SSE-Events]** → `dashboardReminder` und `dutyReminder` haben bislang stumm 0 Empfänger zurückgegeben, weil die SQL-Queries leer waren. Nach dem Fix gehen plötzlich tatsächlich Erinnerungen raus. Operator-Hinweis vor Deploy: kurz erklären, dass ein einmaliger „Backfill" an Reminder-Mails möglich ist.
- **[Risiko: Test-Korrektur in `trainings/handler_test.go:361` deckt versteckten Bug auf]** → Falls der Test mit korrektem Token-Setup andere Assertions bricht, ist das ein Hinweis darauf, dass der Code dort an `claims.Role == "elternteil"` hängt. Das ist gewünscht — wir wollen genau diese Fehler finden. Im Zweifel als separates Task tracken.
- **[Trade-off: `NavItem.roles` bleibt polymorph]** → Neue Devs könnten verwirrt sein, ob ein String ein System-Role-Wert oder eine Vereinsfunktion ist. Mitigation: JSDoc und ein kurzer Abschnitt in `docs/berechtigungen.md`.

## Migration Plan

1. **Pre-flight (manuell auf VPS):** `sqlite3 /var/lib/teamwerk/teamwerk.db "SELECT id, name, target_role FROM duty_types WHERE target_role NOT IN ('spieler','elternteil','trainer','vorstand','sportliche_leitung','vorstand_beisitzer','kassierer');"`. Falls Treffer → Operator entscheidet vor Migration, ob die Auto-Mappierung passt.
2. **Code-Änderungen** (Task-Gruppen A–D) landen in einem PR mit Migration 042. Pro Datei ein Conventional-Commit.
3. **`make deploy`** zieht Migration automatisch nach (golang-migrate im Binary).
4. **Rollback:** Migration 042 hat `.down.sql`, das den alten CHECK wiederherstellt und `'sportliche_leitung'`/`'vorstand_beisitzer'`/`'kassierer'` als `'vorstand'` zurückmappt. Code-Rollback via `git revert`.
5. **Smoke-Test post-deploy:** Dashboard → Dienstbörse → Eintrag mit `target_role='spieler'`/`'elternteil'`/`'trainer'` aufmachen → schauen, dass Reminder-Cron einen Tag später Empfänger findet (oder via `teamwerk scheduler:run` manuell triggern).

## Open Questions

- Keine. Alle Entscheidungen sind getroffen, das Scope ist eng gefasst.
