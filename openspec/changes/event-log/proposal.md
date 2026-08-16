# Event-Log auf dem Dashboard

## Why

Wer eine Push-Benachrichtigung wegwischt, verliert die Information endgültig. Es gibt im
System keine Stelle, an der man nachlesen kann, was einem mitgeteilt wurde — nicht einmal
bei Absagen, wo der Grund („Halle gesperrt") ausschließlich im zugestellten Push-Text lebt
und bewusst nirgends persistiert wird.

Push ist dabei nicht nur wegwischbar, sondern an **sechs unabhängigen Stellen** verlustbehaftet:

```
   Empfänger im Audience
        │
        ├─ push_enabled = 0                      → weg
        ├─ cfg.VAPIDPrivateKey == ""             → weg (ganze Instanz)
        ├─ keine Zeile in push_subscriptions     → weg (nie erlaubt / iOS
        │                                            nicht als PWA installiert)
        ├─ HTTP 410 → Subscription gelöscht      → weg
        ├─ sendNotification-Fehler               → weg (kein Retry)
        └─ TTL: 3600 — Gerät > 1 h offline       → weg (Push-Dienst verwirft)
        │
        ▼
   tatsächlich zugestellt
```

Die `TTL: 3600` in `push.SendToUsers` ist die härteste davon: wer das Handy einen Abend
liegen lässt, verliert die Meldung **unabhängig von jeder Einstellung**. E-Mail ist per
Default aus. Für die meisten Nutzer ist Push damit der einzige Träger — und ein
unzuverlässiger.

Der Event-Log dreht das um: er wird der **vollständige** Kanal, Push der Beschleuniger.

## What Changes

### Ein Log-Eintrag je Empfänger, geschrieben vor jeder Filterung

Neue Tabelle `user_events` (Migration `050`). Der Fan-out passiert in `notify.Send`,
**bevor** die Präferenz-Filter laufen:

```
   userIDs  ═══ „wen betrifft das?"  ─────────────────────► user_events  (vollständig)
                                     │
                                     ├─ push_enabled?  ──► Web-Push      (lückenhaft)
                                     └─ email_enabled? ──► E-Mail        (meist aus)
```

`notification_preferences` bedeutet damit ausschließlich **Zustellkanal**, nie
**Sichtbarkeit**. Wer alle Pushes abgeschaltet hat, bekommt trotzdem den vollständigen Log.

### Die Fassade bekommt zwei Optionen — und wird dadurch die einzige Schreibstelle

Sechs Sender rufen heute `push.SendToUsers` direkt auf. Keiner davon ist fehlerhaft: fünf
von sechs rufen `push.FilterByPushPref` selbst auf, direkt vor dem Senden. Sie bauen
`notify.Send` von Hand nach, weil die Fassade ihre Variante nicht ausdrücken kann:

| Stelle | Warum an der Fassade vorbei |
|---|---|
| `matchreports/notify.go:52` | will bewusst keine E-Mail |
| `carpooling/paarungen_handler.go` ×3 | dito |
| `scheduler.go:342` (Video-Löschwarnung) | Push ignoriert die Präferenz absichtlich (Datenverlust-Warnung) |
| `scheduler.go:553` (Dienst-Reminder) | eigener, reichhaltiger E-Mail-Body + eigener Idempotenz-Log |
| `videos/worker.go:304` | Test-Seam über ein Interface |

`notify.Send` bekommt daher zwei funktionale Optionen — `notify.NoEmail()` und
`notify.SkipPushPref()` — und alle sechs fallen auf die Fassade zurück:

```
   VORHER                              NACHHER

   20× notify.Send ──┐                 26× notify.Send(…, opts) ──► ein Fan-out
                     ├─► Push/Mail                                   │
    6× Handbau    ───┘                                               ├─► user_events
    (je 3–8 Zeilen)                                                  ├─► Push
                                                                     └─► E-Mail
   ⇒ 7 Schreibstellen für den Log      ⇒ genau eine
```

Ein neuer Architektur-Test (`internal/arch/pushfanout_test.go`, analog zum bestehenden
`broadcast_test.go`) erzwingt das mechanisch: außerhalb von `internal/notify` und
`internal/push` darf `push.SendToUsers` nicht mehr aufgerufen werden. Ausnahmen nur über
eine begründete Allowlist. Ohne diesen Test driftet der Log zwangsläufig — die nächste
Domäne baut wieder von Hand nach, und die Lücke merkt niemand.

### Retention: drei Tage nach Ansicht, gestempelt bei Auslieferung

`GET /api/dashboard` liefert die Einträge und stempelt dabei `seen_at` auf **genau den
zurückgegebenen Zeilen**. Danach laufen drei Tage.

```
  App geöffnet, Liste geladen
           │
    ●──────●───────── 3 Tage ─────────►╳
    │      seen_at
  Event   (Server stempelt beim Ausliefern)

  Nie eingeloggt → Uhr startet nie → bleibt liegen
```

Kein `IntersectionObserver`, kein zusätzlicher Request. Wer nicht in der App war, verliert
nichts.

### Der Absagegrund wird künftig persistiert

Bisher galt (`docs/agent/06-gotchas.md`): *„Der Grund wird bewusst nirgends persistiert."*
Das war der akzeptierte Preis dafür, dass Löschen wirklich löscht — mit der ausdrücklich
benannten Folge, dass wer die Push wegwischt, den Grund nirgends wiederfindet. Genau dieses
Problem löst der Log. Die Entscheidung wird umgekehrt: der Grund steht als Teil des
Meldungstexts in `user_events.body` und verschwindet mit der Retention. **Die drei Tage sind
damit die neue Datenschutz-Aussage** und ersetzen „gar nicht gespeichert".

### Chat bleibt draußen

Chat und Mitteilungen laufen über `chat.Handler.pushFn` → `push.SendToUserWithBadge`, einen
Pfad, der `notify.Send` nie berührt hat. Sie haben eigene Ungelesen-Zähler, den App-Badge
und eine eigene Dashboard-Section „Nachrichten". Der Ausschluss kostet keine Filterlogik;
er wird durch den `CHECK`-Constraint auf `user_events.category` (ohne `chat`) und einen
Allowlist-Eintrag im Architektur-Test dokumentiert.

Der App-Badge (`navigator.setAppBadge`) bleibt unverändert Chat-only — der Event-Log zahlt
nicht ein.

### Dashboard-Section „Ereignisse"

Neue `Accordion`-Section, gleiche Optik wie die bestehenden. Bewusst **nicht**
„Benachrichtigungen" genannt: „Nachrichten" (jemand spricht mich an, ungelesen-basiert) und
„Ereignisse" (die Terminlage bewegt sich, 3-Tage-Retention) müssen unterscheidbar bleiben.

Live-Aktualisierung entsteht ohne neue Hub-Verdrahtung: die Dashboard-Seite lädt bereits
bei `games`/`trainings`/`duties`/`mitfahrgelegenheiten` neu, und das Broadcast-Gate
garantiert, dass jede Mutations-Route diese Events sendet.

## Nicht Teil dieses Changes

- **Zweite Uhr für Erinnerungen.** Sieben der ~26 Meldungstypen sind zeitbezogen („Spiel in
  3 Stunden") und stehen nach dem Termin noch bis zu drei Tage im Log. Bewusst akzeptiert;
  ein optionales `relevant_until` wäre ein rein additiver Nachtrag.
- **Eigene Seite mit Verlauf.** Der Log lebt nur in der Dashboard-Section (max. 30 Einträge).
- **Log-spezifische Präferenzen.** Es gibt keine Kategorie-Abschaltung für den Log — er ist
  per Definition vollständig.

## Test-Anforderungen

| Route / Einheit | Testname | Erwartung |
|---|---|---|
| `notify.Send` | `TestSend_SchreibtLogFuerAlleEmpfaenger` | Fan-out über die **ungefilterte** `userIDs`; je Empfänger genau eine Zeile |
| `notify.Send` | `TestSend_LogUnabhaengigVonPushPraeferenz` | `push_enabled=0` → keine Push, **trotzdem** Log-Zeile |
| `notify.Send` | `TestSend_LogUnabhaengigVonPushSubscription` | Nutzer ohne `push_subscriptions`-Zeile bekommt Log-Zeile |
| `notify.Send` | `TestSend_NoEmailUnterdruecktNurEmail` | `NoEmail()` → keine Mail, Push und Log unverändert |
| `notify.Send` | `TestSend_SkipPushPrefIgnoriertPraeferenz` | `SkipPushPref()` → Push trotz `push_enabled=0`, E-Mail weiter nach Präferenz |
| `notify.Send` | `TestSend_LeereEmpfaengerlisteSchreibtNichts` | keine Zeile, kein Fehler |
| `GET /api/dashboard` | `TestDashboard_LiefertEventsNeuesteZuerst` | 200, `events` absteigend nach `createdAt`, max. 30 |
| `GET /api/dashboard` | `TestDashboard_StempeltSeenAtNurAufGelieferten` | 200; Zeilen jenseits des Caps behalten `seen_at IS NULL` |
| `GET /api/dashboard` | `TestDashboard_SeenAtWirdNichtUeberschrieben` | zweiter Abruf lässt den ersten Stempel unverändert |
| `GET /api/dashboard` | `TestDashboard_FremdeEventsUnsichtbar` | Events anderer Nutzer erscheinen nie |
| `GET /api/dashboard` | `TestDashboard_OhneAuth401` | 401 |
| Retention | `TestPurge_LoeschtGeseheneNachDreiTagen` | `seen_at` älter als 3 Tage → weg |
| Retention | `TestPurge_LaesstUngeseheneLiegen` | `seen_at IS NULL`, 30 Tage alt → bleibt |
| Retention | `TestPurge_SicherheitskappeGreiftBeiNeunzigTagen` | `seen_at IS NULL`, 91 Tage alt → weg |
| Architektur | `TestArchitecture_KeinDirekterPushFanout` | `push.SendToUsers` außerhalb `notify`/`push` nur laut Allowlist |
| Architektur | `TestArchitecture_PushAllowlistOhneWaisen` | Allowlist-Eintrag ohne realen Aufrufer lässt den Test fehlschlagen |
| Absage | `TestDeleteGame_GrundStehtImEventLog` | ersetzt `TestDeleteGame_GrundWirdNichtPersistiert`; Grund im `body` |

**Garantierte Invarianten:**

1. Die Empfängermenge wird **beim Schreiben eingefroren** und beim Lesen nie neu
   ausgewertet. Ein Nutzer, der das Team verlässt, behält seine Zeilen bis zur Retention;
   ein neu hinzugekommener bekommt keine rückwirkend.
2. Der Log ist eine **Obermenge** der zugestellten Pushes für jede Kategorie außer `chat`.
3. `seen_at` wird ausschließlich auf Zeilen gesetzt, die im selben Response ausgeliefert
   wurden.
