## Context

Der Notification-Code ist heute zweischrittig und an die Aufrufer verteilt:

```go
uids := push.FilterByPushPref(h.db, candidates, "duties")
go push.SendToUsers(h.db, h.cfg, uids, "Titel", "Body", "/url")
```

Email-Versand wird nirgends an die Kategorie-Präferenz gekoppelt — `push.HasEmailEnabled` existiert, wird aber nur im `scheduler` für `duty_reminders` benutzt. Im Profil-UI sind die Email-Toggles für `games`, `trainings`, `duties`, `carpooling` zwar vorhanden, aber kein Backend-Pfad reagiert darauf.

Beim Event-Löschen schickt `DeleteGame` heute eine pauschale `"games"`-Push an Team + Eltern. Dienst-Zugewiesene (oft Eltern in der Kategorie `duties`) sind nicht zwingend in dieser Empfängerliste oder haben Spiel-Push deaktiviert.

`duty_accounts.ist` wird heute beim Fulfill incrementiert (`duties.Fulfill`-Handler) und beim Cash-Substitute mitgepflegt. Es gibt keine Re-Compute-Logik. Beim FK-Cascade-Delete der Assignments bleibt der Ist-Wert stehen → Schiefstand im Konto.

## Goals / Non-Goals

**Goals:**
- Eine einzige Fassade für Push+Email-Notifications, mit Kategorie als first-class Argument
- Bestehende Aufrufer migrieren — verhaltensneutral, solange Nutzer Email nicht anhaken
- Beim Event-Löschen werden die Dienst-Zugewiesenen gezielt und persönlich (Eventname im Body) benachrichtigt
- `duty_accounts.ist` bleibt nach Cascade-Delete korrekt

**Non-Goals:**
- Keine Trainings-Behandlung — Schema hat keinen Bezug, `DeleteSession`/`DeleteSeries` bleiben wie sie sind (außer Fassade-Migration)
- Kein neues `notification_preferences`-Kategorie-Feld
- Keine Änderung des Profil-UI für Notifications
- Keine Migration des Schemas
- Keine Behandlung von `cash_substitute` (Status wird laut Anwender nicht mehr genutzt — Aufräumen in separatem Change)
- Kein Backfill von Konto-Schiefständen aus der Vergangenheit

## Decisions

**Neuer Package-Pfad: `internal/notifications/`** (nicht in `internal/push/` erweitern). Begründung: Die Fassade umfasst Push und Email — `push` als Paketname wäre irreführend. `notifications` bleibt offen für SMS, In-App etc. Das bestehende `push`-Paket bleibt; `notifications.Send` ruft `push.SendToUsers` intern auf.

**Signatur der Fassade:**

```go
func Send(db *sql.DB, cfg *config.Config, uids []int, category, title, body, url string)
```

Identisch zur Summe aus `FilterByPushPref` + `SendToUsers`, plus `category`. Intern:

```go
func Send(db, cfg, uids, category, title, body, url) {
    if len(uids) == 0 { return }
    pushUIDs  := push.FilterByPushPref(db, uids, category)        // existiert
    emailUIDs := filterByEmailPref(db, uids, category)            // neu, default 0
    push.SendToUsers(db, cfg, pushUIDs, title, body, url)         // sync wie heute
    for _, uid := range emailUIDs {
        go sendCategoryEmail(db, cfg, uid, category, title, body, url)
    }
}
```

`sendCategoryEmail` lädt die User-Email aus `users` und ruft das bestehende `mailer.Send`. Email-Inhalt ist Plain-Text: `<body>\n\nDirektlink: https://intern.team-stuttgart.org<url>`. Keine HTML-Templates in diesem Change — bewusst minimal.

**Empfänger-Sammlung beim Event-Delete:**

Vor dem `DELETE FROM games WHERE id=?` wird einmal gequerlt:

```sql
SELECT DISTINCT da.user_id, da.status
FROM duty_assignments da
JOIN duty_slots ds ON ds.id = da.duty_slot_id
WHERE ds.game_id = ?
```

Daraus zwei Listen:
- `assignedUIDs` — alle (für die Notification)
- `fulfilledUIDs` — Subset mit `status='fulfilled'` (für die Konto-Rekomputation)

Anschließend wird die Saison des Events gequerlt (`games.season_id`) und für jeden `fulfilledUID` der `ist`-Wert via Aggregat neu gesetzt:

```sql
UPDATE duty_accounts SET ist = (
    SELECT COALESCE(SUM(dt.hours_value), 0)
    FROM duty_assignments da
    JOIN duty_slots ds   ON ds.id = da.duty_slot_id
    JOIN duty_types dt   ON dt.id = ds.duty_type_id
    WHERE da.user_id = ? AND ds.season_id = ? AND da.status = 'fulfilled'
)
WHERE user_id = ? AND season_id = ?
```

Aggregat-basiertes Re-Compute ist robuster als Decrement, weil ein einzelner gelöschter Eventbeitrag mehrere Dienste enthalten kann und Off-by-one-Bugs ausgeschlossen sind.

**Reihenfolge in der Transaction:**

```
BEGIN
  SELECT assignees + fulfilled-Liste + season_id
  DELETE FROM games WHERE id=?           -- triggert FK-Cascade auf duty_slots/duty_assignments
  UPDATE duty_accounts.ist (für fulfilledUIDs)
COMMIT
THEN broadcast + notifications.Send(...)
```

Die Konto-Rekomputation läuft *nach* dem Cascade-Delete, weil sonst die zu löschenden Assignments noch in die Summe einfließen. Sie läuft *in derselben Transaction*, damit Konto und Dienste atomic konsistent bleiben.

**Notification-Text:**

```
Titel: „Dienst entfällt"
Body : „Dein Dienst zum {eventName} am {dd.mm.yyyy} wurde gelöscht."
URL  : „/dienste"
```

`eventName` ist `opponent` für Spiele, `event_name` (sofern vorhanden auf `games`) für generische. Wenn `opponent` leer ist, Fallback auf „Termin am {Datum}".

**Migration der bestehenden Aufrufer:**

Eins-zu-eins-Ersatz. Beispiel `trainings.DeleteSession`:

```go
// vorher
uids := push.FilterByPushPref(h.db, h.teamMembersAndParents(teamID), "trainings")
go push.SendToUsers(h.db, h.cfg, uids, "Training abgesagt", "...", "/training")

// nachher
notifications.Send(h.db, h.cfg, h.teamMembersAndParents(teamID),
    "trainings", "Training abgesagt", "...", "/training")
```

Die Fassade kümmert sich um Push-Filterung und Goroutine-Versand intern. Das `go` vor dem alten Aufruf entfällt.

**`?delete_slots`-Cleanup:**

Im Backend wird die Verzweigung weggekürzt — die FK-Cascade greift in beiden Branches identisch. Im Frontend werden zwei Stellen geputzt:

- `web/src/components/GameEditModal.tsx:80`
- `web/src/pages/SpieltagDetailPage.tsx:210`

Backend ignoriert den Query-Param weiterhin still — keine Breaking Change für externe Aufrufer.

## Risks / Trade-offs

**Atomicity der Konto-Rekomputation.** Wenn die Transaction nach dem Cascade-Delete platzt (z.B. DB-Timeout während `UPDATE duty_accounts`), bleiben Dienste gelöscht aber Konto inkonsistent. Mitigation: kein extra Rückkehrweg, das Transaction-Rollback fängt es ab — SQLite WAL ist verlässlich. Plus: das Aggregat ist idempotent, ein zweiter Lauf gleicht's wieder aus.

**Email-Versand-Volumen.** Aktuell ist `email_enabled` per Default 0 für alle Kategorien. Solange Nutzer es nicht im Profil einschalten, ändert sich nichts. Wenn eines Tages 50 Eltern für „Dienste" Email anhaken, schickt ein einziger Eventdelete bis zu 50 Emails — SMTP-Limits beim Mittwald-Provider beachten. Mitigation: `go`-Routine pro Mail, kein Sammelversand; im Stress-Fall ggf. später throttlen.

**Doppelte Notifications.** Ein Dienst-Zugewiesener, der auch im Spiel-Responder-Topf landet, kriegt heute schon zwei Pushes vom selben Event (eine als Spieler, eine als Dienst). Das bleibt so — die Kategorien sind aus Nutzersicht semantisch verschieden („dein Spiel" vs. „dein Dienst"). Eine Deduplication würde die Fassade aufblähen und gehört nicht in diesen Change.

**Migration verändert subtile Reihenfolge.** Die alten Aufrufer machten den Push-Send in einer separaten Goroutine *nach* `h.hub.Broadcast`. Die Fassade behält diese Reihenfolge, weil Broadcast vom Aufrufer kommt und der Fassade-Call danach steht. Trotzdem alle Aufrufstellen einzeln verifizieren — Reviewer-Augenmerk.

**Trainings ohne Dienste — bewusst unverändert.** Wenn später Dienste an Trainings hängen sollen, ist ein separater Change mit Schema-Migration nötig (`duty_slots.training_session_id` als nullable FK). Dieser Change zementiert das nicht weiter, lässt aber die Tür offen.
