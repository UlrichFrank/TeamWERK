# Design — Event-Log

## Decision 1: Einfügepunkt vor den Präferenz-Filtern

`notify.Send` filtert heute in zwei Schritten und sendet dann. Der Log-Fan-out sitzt
**oberhalb** beider Filter:

```go
var Send = func(db, cfg, userIDs []int, category, title, body, url string, opts ...Option) {
    if len(userIDs) == 0 { return }
    o := applyOptions(opts)

    eventlog.Record(db, userIDs, category, title, body, url)   // ◄── ungefiltert

    pushUIDs := userIDs
    if !o.skipPushPref { pushUIDs = push.FilterByPushPref(db, userIDs, category) }
    ...
}
```

Das ist der ganze Kern des Changes. `userIDs` beantwortet „wen betrifft das?",
`notification_preferences` beantwortet „auf welchem Weg erfährt er es?" — zwei Fragen, die
bisher unbeabsichtigt gekoppelt waren.

**Warum nicht danach:** hinter dem Filter stünde im Log nur, was tatsächlich gepusht wurde.
Dann wäre der Log für genau die Nutzer leer, für die er am wertvollsten ist — die mit
abgeschalteten Pushes.

## Decision 2: Denormalisierter Fan-out, eine Zeile je Empfänger

`user_events` trägt `title`/`body`/`url` je Empfänger dupliziert, statt über eine
`events` + `event_recipients`-Normalisierung zu gehen.

Begründung: `seen_at` ist ohnehin pro Empfänger, die Join-Zeile ist also unvermeidbar. Die
Normalisierung spart nur den Text — bei einem Vereins-Fan-out von ~180 Empfängern × ~120 Byte
sind das ~22 KB pro Meldung, bei drei Tagen Retention eine Größenordnung, die auf dem VPS
(1 GB RAM, SQLite) nicht messbar ist. Der Preis wäre ein Join auf jedem Dashboard-Abruf und
eine zweite Tabelle mit eigenem Retention-Verhalten.

## Decision 3: Empfängermenge wird beim Schreiben eingefroren

Der Log protokolliert, **was gesendet wurde** — nicht, was heute gelten würde. Beim Lesen
wird nie gegen den aktuellen Kader/die aktuelle Vereinsfunktion nachgefiltert.

Das ist bewusst gegen eine naheliegende „Verbesserung" gerichtet, die in beide Richtungen
lügen würde:

- Wer das Team verlässt, **hat** die Push bekommen. Die Zeile zu verstecken behauptet, er
  sei nie informiert worden.
- Wer neu dazukommt, bekäme rückwirkend Meldungen zu sehen, die nie an ihn gingen.

Konsequenz, die benannt sein will: eine zu weit gefasste Audience-Funktion
(`teamMembersAndParents`, `eligibleDutyUsers`, …) ist heute ein flüchtiger Fehlversand,
nach diesem Change ein drei Tage lang nachlesbarer Eintrag. Die Funktionen verdienen mehr
Sorgfalt als bisher.

## Decision 4: `seen_at` bei Auslieferung, exakt auf den gelieferten Zeilen

Die Stempelung passiert serverseitig in `GET /api/dashboard`, in derselben Transaktion wie
das Lesen, und zwar über die **IDs der tatsächlich zurückgegebenen Zeilen** — nicht über
`WHERE user_id = ?`.

Das ist der Unterschied zwischen korrekt und subtil kaputt: der Response ist auf 30
Einträge gedeckelt. Ein `UPDATE … WHERE user_id = ?` würde auch die Zeilen 31+ stempeln,
die der Nutzer nie gesehen hat — sie verfielen ungesehen.

`seen_at` wird nur gesetzt, wo es `NULL` ist (`WHERE seen_at IS NULL`), der zweite Abruf
verschiebt die Uhr also nicht.

**Ein GET schreibt.** Bewusst: das Broadcast-Gate prüft nur `POST/PUT/PATCH/DELETE`, es
entsteht also keine Broadcast-Pflicht, und ein Stempel ist idempotent. Die Alternative
(eigener `POST /api/events/seen`) kostet einen Roundtrip für keinen fachlichen Gewinn.

## Decision 5: Retention — drei Tage nach `seen_at`, plus eine Sicherheitskappe

```sql
DELETE FROM user_events
 WHERE (seen_at IS NOT NULL AND seen_at   < datetime('now','-3 days'))
    OR (seen_at IS     NULL AND created_at < datetime('now','-90 days'));
```

Die erste Zeile ist die Anforderung. Die zweite ist eine **Abweichung**, die begründet sein
muss: streng genommen wurde „ungesehen bleibt liegen" ohne Obergrenze festgelegt. Ein
Account, der nie wieder eingeloggt wird (ausgetretenes Mitglied, verwaister Kinder-Account),
sammelt dann unbegrenzt Zeilen — auf einem 1-GB-VPS ein realer Betriebspfad in eine volle
Platte.

90 Tage sind so gewählt, dass die fachliche Aussage („du verlierst nichts, nur weil du im
Urlaub warst") vollständig erhalten bleibt: die längste plausible Abwesenheit liegt weit
darunter. Wer 90 Tage nicht in der App war, hat den Log nicht verloren, sondern den Bezug
zur Saison.

Der Job läuft im bestehenden Scheduler (`internal/scheduler/eventlog_retention.go`),
Minutentakt wie die anderen. Kein `notification_log`-Idempotenzschutz nötig — ein `DELETE`
ist von Natur aus wiederholbar.

## Decision 6: Zwei funktionale Optionen statt sechs Handbauten

```go
type Option func(*options)

// NoEmail unterdrückt den E-Mail-Zweig vollständig. Für Meldungen, die
// bewusst push-only sind (Mitfahr-Paarungen, Spielbericht-Freigabe) oder
// deren Aufrufer eine eigene, reichhaltigere Mail baut (Dienst-Reminder).
func NoEmail() Option

// SkipPushPref sendet Push unabhängig von notification_preferences.
// Ausschließlich für Meldungen, deren Nichtzustellung Datenverlust bedeutet
// (Video-Löschwarnung). E-Mail bleibt präferenzgesteuert.
func SkipPushPref() Option
```

Variadisch, damit die 20 bestehenden Aufrufstellen **unverändert** bleiben. `Send` ist
weiterhin eine package-`var` (Test-Seam); die acht Test-Doubles müssen ihre Signatur um
`...Option` erweitern — mechanisch.

Zuordnung der sechs Umbauten:

| Stelle | Neue Form |
|---|---|
| `matchreports/notify.go` | `notify.Send(…, "operativ", …, notify.NoEmail())` |
| `carpooling/paarungen_handler.go` ×3 | `notify.Send(…, "carpooling", …, notify.NoEmail())` |
| `scheduler.go:342` Video-Löschwarnung | `notify.Send(…, "sonstiges", …, notify.SkipPushPref())` — E-Mail-Zweig entfällt, die Fassade filtert ihn bereits korrekt |
| `scheduler.go:553` Dienst-Reminder | `notify.Send(…, "duty_reminders", …, notify.NoEmail())`; die Sonder-Mail bleibt daneben stehen |
| `videos/worker.go:304` | `workerConfig.notifySend` statt `pushSend`/`emailSend`; die Vorfilterung in `runNotify` entfällt, die Fassade macht sie |

Der Dienst-Reminder behält seinen `notification_log`-Idempotenzschutz **vor** dem
`notify.Send`-Aufruf — sonst schriebe ein zweiter Cron-Lauf doppelte Log-Zeilen. Dasselbe
gilt für die Video-Löschwarnung.

## Decision 7: Architektur-Test statt Disziplin

`internal/arch/pushfanout_test.go` parst alle `internal/`-Packages und meldet jeden Aufruf
von `push.SendToUsers` außerhalb von `internal/notify` und `internal/push`. Allowlist mit
Begründung, und — wie beim Broadcast-Gate — ein verwaister Eintrag lässt den Test
fehlschlagen.

Erwarteter Allowlist-Inhalt nach diesem Change: **leer**, bis auf
`push.SendToUserWithBadge` in `internal/chat` (eigener Kanal, eigener Badge, bewusst nicht
im Log). Damit ist die Chat-Ausnahme genau einmal und an sichtbarer Stelle dokumentiert.

Ohne diesen Test ist der Change eine Momentaufnahme: die siebte Domäne baut wieder von Hand
nach, und die Lücke fällt niemandem auf, weil ein fehlender Log-Eintrag nichts kaputt macht
— er fehlt nur.

## Decision 8: Chat-Ausschluss mechanisch über den CHECK-Constraint

`user_events.category` bekommt einen `CHECK` über die acht Nicht-Chat-Kategorien. Wer
später Chat in den Log holen will, braucht eine Migration — genau die richtige Reibung für
eine Scope-Entscheidung, und sie steht dann im Schema statt in einem Kommentar.

Der `CHECK` fängt nebenbei Tippfehler in `category` ab, die heute nur dazu führen würden,
dass `FilterByPushPref` niemanden matcht (still keine Push).

## Decision 9: Keine neue Hub-Verdrahtung

`notify.Send` bekommt **keinen** `*hub.EventHub`. Live-Aktualisierung entsteht als
Nebenprodukt bestehender Garantien:

```
   Mutations-Route  ──► h.hub.Broadcast("games")   ← vom Broadcast-Gate erzwungen
                    └─► notify.Send(...)           ← schreibt den Log

   DashboardPage: useLiveUpdates(e => { if (e === 'games' || …) load(true) })
                                                   ← existiert bereits
```

Jede Meldung aus einem HTTP-Handler wird also live sichtbar, ohne dass eine Zeile
Hub-Code dazukommt. Scheduler-Meldungen (Erinnerungen) erscheinen beim nächsten Laden —
akzeptiert, weil sie ohnehin per Push kommen.

Die Section muss dafür in die bestehende `useLiveUpdates`-Bedingung von `DashboardPage`
aufgenommen werden; neue Event-Namen entstehen nicht.

## Decision 10: Der Absagegrund kippt

`docs/agent/06-gotchas.md` hält heute fest, dass der Grund nirgends persistiert wird, und
`TestDeleteGame_GrundWirdNichtPersistiert` prüft das per DB-Scan. Beides wird umgeschrieben.

Die ursprüngliche Begründung bleibt gültig — Löschen soll wirklich löschen, es gibt kein
`games.status='cancelled'`. Was sich ändert, ist die Aussage über den **Meldungstext**: er
existiert jetzt drei Tage lang serverseitig, statt nur im Zustellkanal. Der neue Test
(`TestDeleteGame_GrundStehtImEventLog`) prüft die Umkehrung positiv, und der Gotcha nennt
die Retention als die Grenze, die vorher „gar nicht gespeichert" war.

Das betrifft vier Meldungstypen: Termin abgesagt, Training abgesagt, Trainingsserie
beendet, Dienst entfällt.

## Verworfen: Erinnerungen aus dem Log filtern

Sieben Meldungstypen sind zeitbezogen und nach dem Termin wertlos. Sie wegzulassen wäre die
naheliegende Kuratierung — und genau der Bruch des Vertrags „alles, was dich betrifft".
Eine handverlesene Ausnahmeliste macht aus einer Garantie eine Geschmacksfrage („wieso
steht die Trainingserinnerung nicht drin?").

Die saubere Lösung wäre eine zweite Uhr (`relevant_until` = Eventzeit, was zuerst abläuft
gewinnt). Sie ist bewusst nicht Teil dieses Changes: rein additiv, jederzeit nachrüstbar,
und ob sich das Nachhängen in der Praxis überhaupt stört, weiß man erst mit Nutzung.
Sortierung nach `created_at DESC` dämpft es (Erinnerungen rutschen nach unten).

## Schema

```sql
CREATE TABLE user_events (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category   TEXT     NOT NULL
               CHECK (category IN ('games','trainings','duties','duty_reminders',
                                   'carpooling','membership','operativ','sonstiges')),
    title      TEXT     NOT NULL,
    body       TEXT     NOT NULL DEFAULT '',
    url        TEXT     NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    seen_at    DATETIME
);

CREATE INDEX idx_user_events_user_created ON user_events(user_id, created_at DESC);
CREATE INDEX idx_user_events_retention    ON user_events(seen_at, created_at);
```

`ON DELETE CASCADE` auf `users` — ein gelöschter Nutzer nimmt seinen Log mit.

Kein `ref_type`/`ref_id`: der Log referenziert bewusst nichts. Die Meldung ist zum
Sendezeitpunkt eingefroren, weil die referenzierten Objekte (gelöschte Spiele, entfernte
Dienst-Slots) danach nicht mehr existieren — genau die Fälle, in denen Nachlesen am
wichtigsten ist. `url` ist ein Sprungziel, kein Fremdschlüssel, und darf ins Leere zeigen.

## Paketstruktur

Neues Foundation-Package `internal/eventlog` (`Record`, `ListForUser`, `Purge`), damit
`notify` (Foundation, schreibt), `dashboard` (Domain, liest) und `scheduler` (Domain,
räumt auf) es teilen können, ohne dass Domain→Domain-Kanten entstehen. Eintrag in
`internal/arch/arch_test.go` unter `foundation`.
