# Beitragsfrei + Zweitspielrecht Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Zwei boolesche Felder auf `members` ergänzen: `beitragsfrei` (in Bankdaten, admin-only) und `zweitspielrecht` (in Stammdaten unter Stammverein), plus CSV-Import-Ableitung für `beitragsfrei`.

**Architecture:** SQLite-Migration mit `ALTER TABLE ADD COLUMN`, Backend-Erweiterung in `internal/members/handler.go` (Struct, GET, PUT, CSV-Import), Frontend-Checkboxen in den bestehenden Tab-Komponenten.

**Tech Stack:** Go 1.23 · SQLite (modernc.org/sqlite) · React 18 + TypeScript · Tailwind v3

---

## Dateien

| Datei | Aktion |
|-------|--------|
| `internal/db/migrations/035_beitragsfrei_zweitspielrecht.up.sql` | Neu |
| `internal/db/migrations/035_beitragsfrei_zweitspielrecht.down.sql` | Neu |
| `internal/members/handler.go` | Modify: Struct, GET, PUT, CSV-Import |
| `web/src/pages/MemberDetailPage.tsx` | Modify: Interface, form-State, applyMemberToForm |
| `web/src/components/admin/MemberKontaktTab.tsx` | Modify: Interface, Checkbox beitragsfrei |
| `web/src/components/admin/MemberStammdatenTab.tsx` | Modify: Interface, Checkbox zweitspielrecht |

---

## Task 1: DB-Migration

**Files:**
- Create: `internal/db/migrations/035_beitragsfrei_zweitspielrecht.up.sql`
- Create: `internal/db/migrations/035_beitragsfrei_zweitspielrecht.down.sql`

- [ ] **Schritt 1: Up-Migration anlegen**

Inhalt `internal/db/migrations/035_beitragsfrei_zweitspielrecht.up.sql`:

```sql
ALTER TABLE members ADD COLUMN beitragsfrei    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE members ADD COLUMN zweitspielrecht INTEGER NOT NULL DEFAULT 0;
```

- [ ] **Schritt 2: Down-Migration anlegen**

Inhalt `internal/db/migrations/035_beitragsfrei_zweitspielrecht.down.sql`:

```sql
ALTER TABLE members DROP COLUMN beitragsfrei;
ALTER TABLE members DROP COLUMN zweitspielrecht;
```

- [ ] **Schritt 3: Migration lokal anwenden**

```bash
make migrate-up
```

Erwartete Ausgabe: keine Fehler, Migration 035 wird ausgeführt.

- [ ] **Schritt 4: Schema prüfen**

```bash
sqlite3 teamwerk.db ".schema members" | grep -E "beitrag|zweit"
```

Erwartete Ausgabe:
```
beitragsfrei    INTEGER NOT NULL DEFAULT 0,
zweitspielrecht INTEGER NOT NULL DEFAULT 0,
```

- [ ] **Schritt 5: Commit**

```bash
git add internal/db/migrations/035_beitragsfrei_zweitspielrecht.up.sql \
        internal/db/migrations/035_beitragsfrei_zweitspielrecht.down.sql
git commit -m "chore(db): Migration 035 – beitragsfrei + zweitspielrecht"
```

---

## Task 2: Backend — Struct, GET, PUT

**Files:**
- Modify: `internal/members/handler.go`

### 2a: Member-Struct erweitern

- [ ] **Schritt 1: Felder im Struct ergänzen**

In `handler.go`, im `Member`-Struct (ab Zeile 28), nach `SepaMandat`/`SepaMandatDate`/`SepaMandatURL` (ca. Zeile 62):

```go
Beitragsfrei    bool    `json:"beitragsfrei,omitempty"`
Zweitspielrecht bool    `json:"zweitspielrecht,omitempty"`
```

### 2b: GET `/api/members/:id` erweitern

- [ ] **Schritt 2: SELECT-Query erweitern**

In der `Get`-Funktion (ca. Zeile 323), die SELECT-Query um die zwei neuen Spalten ergänzen. Aktuell endet die SELECT-Liste mit `m.welcome_email_sent_at`. Ändern zu:

```go
row := h.db.QueryRowContext(r.Context(), `
    SELECT m.id, m.first_name, m.last_name,
           COALESCE(m.date_of_birth,''), COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
           m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id,
           COALESCE((SELECT GROUP_CONCAT(mcf.function,',') FROM member_club_functions mcf WHERE mcf.member_id=m.id),''),
           m.street, m.zip, m.city, m.home_club, m.join_date, m.iban, m.account_holder,
           m.photo_path, m.photo_visible,
           m.dsgvo_verarbeitung, m.dsgvo_verarbeitung_date,
           m.dsgvo_weitergabe, m.dsgvo_weitergabe_date,
           m.sepa_mandat, m.sepa_mandat_date, m.sepa_mandat_path,
           m.welcome_email_sent_at,
           m.beitragsfrei, m.zweitspielrecht
    FROM members m
    LEFT JOIN users u ON u.id = m.user_id
    WHERE m.id=?`, id)
```

- [ ] **Schritt 3: Scan-Variablen ergänzen**

Nach der bestehenden `var welcomeEmailSentAt sql.NullString` Deklaration (ca. Zeile 347):

```go
var beitragsfrei, zweitspielrecht int64
```

- [ ] **Schritt 4: Scan-Aufruf erweitern**

Am Ende des `row.Scan(...)` Aufrufs (nach `&welcomeEmailSentAt`):

```go
&beitragsfrei, &zweitspielrecht,
```

- [ ] **Schritt 5: Felder setzen**

`zweitspielrecht` ist ein reguläres Stammdaten-Feld (immer sichtbar für privilegierte Rollen). Nach dem Block wo `base.HomeClub` gesetzt wird (ca. Zeile 388–390), ergänzen:

```go
base.Zweitspielrecht = zweitspielrecht == 1
```

`beitragsfrei` ist ein Finanzfeld (nur für Admins). Im `if isAdmin || isOwn`-Block (ca. Zeile 402–418), nach `base.SepaMandat = sepaMandat == 1`:

```go
base.Beitragsfrei = beitragsfrei == 1
```

### 2c: PUT `/api/members/:id` erweitern

- [ ] **Schritt 6: Request-Struct erweitern**

Im `Update`-Handler (ca. Zeile 481), im anonymen `req`-Struct, nach `SepaMandat`/`SepaMandatDate`:

```go
Beitragsfrei    bool   `json:"beitragsfrei"`
Zweitspielrecht bool   `json:"zweitspielrecht"`
```

- [ ] **Schritt 7: Hauptupdate um zweitspielrecht ergänzen**

Im ersten `UPDATE members SET` (ca. Zeile 531), `zweitspielrecht=?` ergänzen und entsprechenden Wert in die Args-Liste:

```go
_, err := h.db.ExecContext(r.Context(),
    `UPDATE members SET
        first_name=?, last_name=?, date_of_birth=?, member_number=?, pass_number=?,
        jersey_number=?, position=?, gender=?,
        street=?, zip=?, city=?, home_club=?,
        status=?,
        photo_visible=?,
        zweitspielrecht=?,
        updated_at=?
    WHERE id=?`,
    req.FirstName, req.LastName, nullableString(req.DateOfBirth), nullableString(req.MemberNumber),
    nullableString(req.PassNumber), req.JerseyNumber, nullableString(req.Position), req.Gender,
    nullableString(req.Street), nullableString(req.Zip), nullableString(req.City), nullableString(req.HomeClub),
    req.Status,
    boolToInt(req.PhotoVisible),
    boolToInt(req.Zweitspielrecht),
    time.Now(), id)
```

- [ ] **Schritt 8: Admin-Update um beitragsfrei ergänzen**

Im zweiten `UPDATE members SET` (admin-only, ca. Zeile 555), `beitragsfrei=?` ergänzen:

```go
h.db.ExecContext(r.Context(),
    `UPDATE members SET
        join_date=?, iban=COALESCE(?, iban), account_holder=?,
        dsgvo_verarbeitung=?, dsgvo_verarbeitung_date=?,
        dsgvo_weitergabe=?, dsgvo_weitergabe_date=?,
        sepa_mandat=?, sepa_mandat_date=?,
        beitragsfrei=?
    WHERE id=?`,
    nullableString(req.JoinDate), ibanVal, nullableString(req.AccountHolder),
    boolToInt(req.DsgvoVerarbeitung), nullableString(req.DsgvoVerarbeitungDate),
    boolToInt(req.DsgvoWeitergabe), nullableString(req.DsgvoWeitergabeDate),
    boolToInt(req.SepaMandat), nullableString(req.SepaMandatDate),
    boolToInt(req.Beitragsfrei),
    id)
```

- [ ] **Schritt 9: Backend kompilieren**

```bash
go build ./cmd/teamwerk
```

Erwartete Ausgabe: kein Fehler.

- [ ] **Schritt 10: Commit**

```bash
git add internal/members/handler.go
git commit -m "feat(members): Felder beitragsfrei + zweitspielrecht – GET/PUT"
```

---

## Task 3: Backend — CSV-Import

**Files:**
- Modify: `internal/members/handler.go`

- [ ] **Schritt 1: normalizeStatus erweitern**

In der `Import`-Funktion, in der `normalizeStatus`-Closure (ca. Zeile 1331). Neuen Case vor dem Default einfügen:

```go
normalizeStatus := func(s string) string {
    switch s {
    case "aktiv", "verletzt", "pausiert", "ausgetreten", "passiv", "honorar":
        return s
    case "gekündigt", "Vereinswechsel":
        return "ausgetreten"
    case "kein aktiver Sportler mehr":
        return "passiv"
    case "beitragsfrei":
        return "passiv"
    default:
        return "aktiv"
    }
}
```

- [ ] **Schritt 2: beitragsfrei-Ableitung beim INSERT**

Im INSERT-Block (neues Mitglied, ca. Zeile 1471). Direkt vor `gender := normalizeGender(...)` eine Hilfsvariable hinzufügen:

```go
csvStatusRaw := col(row, "Status")
csvBeitragsfrei := strings.EqualFold(strings.TrimSpace(csvStatusRaw), "beitragsfrei")
```

Dann das INSERT-Statement um `beitragsfrei` erweitern:

```go
res, insErr := h.db.ExecContext(r.Context(),
    `INSERT INTO members (member_number, first_name, last_name, date_of_birth,
                          pass_number, jersey_number, position, status, gender, home_club,
                          street, zip, city, join_date, iban, account_holder, sepa_mandat,
                          beitragsfrei)
     VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
    nullableString(col(row, "Mitgliedsnummer")), firstName, lastName,
    nullableString(dob), nullableString(col(row, "Passnummer")),
    jerseyArg, nullableString(col(row, "Position")), status, gender,
    nullableString(col(row, "Stammverein")),
    nullableString(col(row, "Adresse")), nullableString(col(row, "PLZ")), nullableString(col(row, "Ort")),
    nullableString(joinDate), ibanArg, nullableString(col(row, "Kontoinhaber")),
    normalizeSepa(col(row, "SEPA Mandat")),
    boolToInt(csvBeitragsfrei))
```

Hinweis: `status` und `gender` sowie `csvBeitragsfrei` müssen vor dem INSERT gesetzt sein. Die bestehenden Zeilen `gender := normalizeGender(...)`, `status := normalizeStatus(...)` bleiben unverändert; `csvBeitragsfrei` wird direkt davor gesetzt.

- [ ] **Schritt 3: beitragsfrei-Ableitung beim UPDATE (Bestandsmitglieder)**

Im UPDATE-Block (mode == "update"), in der DB-Lookup-Query (ca. Zeile 1422) `beitragsfrei` ergänzen:

```go
query := `SELECT id, member_number, COALESCE(date_of_birth,''),
                 pass_number, jersey_number, position, status, gender, user_id, home_club,
                 COALESCE(street,''), COALESCE(zip,''), COALESCE(city,''),
                 COALESCE(join_date,''), COALESCE(iban,''), COALESCE(account_holder,''),
                 COALESCE(sepa_mandat,0), COALESCE(beitragsfrei,0)
          FROM members
          WHERE lower(first_name)=lower(?) AND lower(last_name)=lower(?)`
```

Variable ergänzen:

```go
var (
    existingID                                     int
    dbMemberNum, dbPassNum, dbPosition             sql.NullString
    dbDOB, dbGender, dbStatus                      string
    dbJerseyNum                                    sql.NullInt64
    dbUserID                                       sql.NullInt64
    dbHomeClub                                     sql.NullString
    dbStreet, dbZip, dbCity                        string
    dbJoinDate, dbIBAN, dbAccountHolder            string
    dbSepaMandat                                   int
    dbBeitragsfrei                                 int
)
scanErr := h.db.QueryRowContext(r.Context(), query, args...).
    Scan(&existingID, &dbMemberNum, &dbDOB, &dbPassNum, &dbJerseyNum, &dbPosition,
        &dbStatus, &dbGender, &dbUserID, &dbHomeClub,
        &dbStreet, &dbZip, &dbCity,
        &dbJoinDate, &dbIBAN, &dbAccountHolder, &dbSepaMandat, &dbBeitragsfrei)
```

Change-Erkennung nach dem `sepa_mandat`-Block ergänzen:

```go
// beitragsfrei aus CSV-Status ableiten
csvStatusRaw := col(row, "Status")
if csvStatusRaw != "" {
    csvBeitragsfrei := boolToInt(strings.EqualFold(strings.TrimSpace(csvStatusRaw), "beitragsfrei"))
    if csvBeitragsfrei != dbBeitragsfrei {
        setClauses = append(setClauses, "beitragsfrei=?")
        setArgs = append(setArgs, csvBeitragsfrei)
        changes = append(changes, fmt.Sprintf("Beitragsfrei: %v → %v", dbBeitragsfrei == 1, csvBeitragsfrei == 1))
    }
}
```

- [ ] **Schritt 4: Backend kompilieren**

```bash
go build ./cmd/teamwerk
```

Erwartete Ausgabe: kein Fehler.

- [ ] **Schritt 5: Commit**

```bash
git add internal/members/handler.go
git commit -m "feat(members): CSV-Import leitet beitragsfrei aus Status-Feld ab"
```

---

## Task 4: Frontend

**Files:**
- Modify: `web/src/pages/MemberDetailPage.tsx`
- Modify: `web/src/components/admin/MemberKontaktTab.tsx`
- Modify: `web/src/components/admin/MemberStammdatenTab.tsx`

### 4a: MemberDetailPage.tsx

- [ ] **Schritt 1: Member-Interface erweitern**

Im `interface Member` (ca. Zeile 12), nach `sepa_mandat_url?: string`:

```ts
beitragsfrei?: boolean
zweitspielrecht?: boolean
```

- [ ] **Schritt 2: form-State initialisieren**

Im `useState<Omit<Member, 'id'>>` (ca. Zeile 65), in der Initialwert-Liste ergänzen:

```ts
beitragsfrei: false, zweitspielrecht: false,
```

- [ ] **Schritt 3: applyMemberToForm erweitern**

In `applyMemberToForm` (ca. Zeile 96), nach `sepa_mandat_url`:

```ts
beitragsfrei: m.beitragsfrei ?? false,
zweitspielrecht: m.zweitspielrecht ?? false,
```

### 4b: MemberKontaktTab.tsx

- [ ] **Schritt 4: Member-Interface erweitern**

Im lokalen `interface Member` (ca. Zeile 18), nach `account_holder?`:

```ts
beitragsfrei?: boolean
```

- [ ] **Schritt 5: Checkbox im Bankdaten-Block einfügen**

Im `<div className="space-y-3">` (nach dem IBAN-Feld, ca. Zeile 133), vor dem schließenden `</div>`:

```tsx
<label className="flex items-center gap-2 cursor-pointer mt-2">
  <input
    type="checkbox"
    checked={form.beitragsfrei || false}
    onChange={e => onFormChange({ beitragsfrei: e.target.checked })}
    className="w-4 h-4 accent-brand-yellow"
  />
  <span className="text-sm text-brand-text">Beitragsfrei</span>
</label>
```

### 4c: MemberStammdatenTab.tsx

- [ ] **Schritt 6: Member-Interface erweitern**

Im lokalen `interface Member` (ca. Zeile 7), nach `home_club?`:

```ts
zweitspielrecht?: boolean
```

- [ ] **Schritt 7: Checkbox nach Stammverein einfügen**

Im `{!isHonorar && (...)}` Block, direkt nach dem Stammverein-Textfeld (ab ca. Zeile 374):

```tsx
{!isHonorar && (
  <div className="mt-4">
    <label className="block text-sm font-medium text-gray-700 mb-1">Stammverein</label>
    <input
      type="text"
      value={form.home_club ?? ''}
      onChange={e => onFormChange({ home_club: e.target.value })}
      placeholder="z. B. TV Cannstatt"
      className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
    />
    <label className="flex items-center gap-2 cursor-pointer mt-2">
      <input
        type="checkbox"
        checked={form.zweitspielrecht || false}
        onChange={e => onFormChange({ zweitspielrecht: e.target.checked })}
        className="w-4 h-4 accent-brand-yellow"
      />
      <span className="text-sm text-brand-text">Zweitspielrecht</span>
    </label>
  </div>
)}
```

Achtung: Die bestehenden zwei separaten `{!isHonorar && (...)}` Blöcke für Stammverein (Textfeld allein) und das neue Zweitspielrecht zu einem einzigen Block zusammenfassen.

- [ ] **Schritt 8: TypeScript-Build prüfen**

```bash
cd web && pnpm tsc --noEmit
```

Erwartete Ausgabe: keine Fehler.

- [ ] **Schritt 9: Commit**

```bash
git add web/src/pages/MemberDetailPage.tsx \
        web/src/components/admin/MemberKontaktTab.tsx \
        web/src/components/admin/MemberStammdatenTab.tsx
git commit -m "feat(members): Checkboxen Beitragsfrei (Bankdaten) + Zweitspielrecht (Stammdaten)"
```

---

## Task 5: Manueller Smoke-Test

- [ ] **Schritt 1: Backend starten**

```bash
go run ./cmd/teamwerk
```

- [ ] **Schritt 2: Frontend starten**

```bash
cd web && pnpm dev
```

- [ ] **Schritt 3: Prüfen — Zweitspielrecht**

1. `http://localhost:5173/mitglieder/29` öffnen → Tab „Stammdaten"
2. Checkbox „Zweitspielrecht" sichtbar unterhalb Stammverein-Feld
3. Haken setzen → Speichern → Seite neu laden → Haken noch gesetzt

- [ ] **Schritt 4: Prüfen — Beitragsfrei**

1. Tab „Bankdaten" öffnen
2. Checkbox „Beitragsfrei" sichtbar unterhalb IBAN-Feld
3. Haken setzen → Speichern → Seite neu laden → Haken noch gesetzt

- [ ] **Schritt 5: Prüfen — CSV-Import**

CSV mit einer Zeile erstellen, in der `Status = "beitragsfrei"`:

```
Vorname;Nachname;Geburtsdatum;Status
Testina;Beitragsfrei;1990-01-01;beitragsfrei
```

Import über Admin-Oberfläche ausführen (mode=update oder append).  
Danach in der DB prüfen:

```bash
sqlite3 teamwerk.db "SELECT status, beitragsfrei FROM members WHERE last_name='Beitragsfrei'"
```

Erwartete Ausgabe: `passiv|1`
