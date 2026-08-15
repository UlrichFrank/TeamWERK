# Design — Absage-Benachrichtigungen mit Inhalt

Alle Code-Stellen in diesem Dokument wurden vor dem Entwurf gelesen; Aussagen über das
heutige Verhalten sind belegt, nicht erinnert.

## 1. Warum kein `status='cancelled'` bei Spielen

Die naheliegende Lösung wäre, Spiele wie Trainings zu modellieren: `training_sessions` hat
seit Migration 001 `status IN ('active','cancelled')` plus `cancel_reason`, und die
Oberfläche zeigt das bereits als durchgestrichenen Termin mit Begründung
(`TerminePage.tsx:502`, `TermineDetailPage.tsx:427`). Damit wären alle vier Beschwerden auf
einen Schlag erledigt, inklusive eines funktionierenden Direktlinks.

Diese Variante wurde **bewusst verworfen**. Ausschlaggebend war die Antwort auf die Frage,
was passiert, wenn ein abgesagtes Spiel doch stattfindet: es wird neu angelegt. Damit
entfällt der Hauptnutzen eines Zustandsfelds (Reaktivierung per Status-Flip), und übrig
bliebe nur der Preis:

| | Kosten bei `games.status` |
|---|---|
| Alle Spiel-Queries | müssen lernen, `cancelled` zu filtern oder darzustellen — `ScopeGamesQuery`, Kalender, Dashboard, iCal-Feed, Anwesenheit, RSVP |
| Auto-Duty-Regen | `runAutoRegen` müsste entscheiden, ob ein abgesagtes Spiel Dienste behält, verliert oder Kontext für `same_day_behavior` bleibt |
| H4A-Import | `external_id`-Abgleich müsste definieren, was ein Reimport mit einem abgesagten Spiel tut |
| Migration | Schemaänderung an der meistgelesenen Tabelle |

Für eine Textverbesserung ist das die falsche Größenordnung. Der Change bleibt deshalb
rein verhaltensbezogen.

**Zwischenvariante, ebenfalls verworfen:** ein separates Absage-Log (kleine Tabelle, im
Kalender als Geist-Eintrag für ~14 Tage sichtbar). Sie hätte den Grund dauerhaft auffindbar
gemacht, ohne das Spielmodell anzufassen. Verworfen, weil sie ein neues Konzept einführt und
der Nutzen erst dann zählt, wenn jemand den Grund später sucht — ein Fall, der bisher nicht
aufgetreten ist. Falls er auftritt, ist diese Variante der nächste Schritt, nicht §1.

## 2. Der Grund hat keine Heimat — bewusst

Weil die Zeile gelöscht wird, existiert der Grund ausschließlich in der zugestellten
Nachricht. Das ist schwächer, als es aussieht:

```
   Wer erfährt "warum"?
   ═══════════════════════════════════════════════════
   Push       ──▶  kommt an, einmal lesbar
                   weggewischt = für immer weg
                   push_enabled Default TRUE   ✓

   E-Mail     ──▶  bleibt im Postfach
                   email_enabled Default FALSE ✗
                   (notify.go: "Users without a row default to false")

   In der App ──▶  nichts. Termin weg, kein Hinweis, kein Verlauf.
```

Konkret: Wer die Push wegwischt und am nächsten Tag nachsehen will, findet im Kalender
keine Lücke, keinen Hinweis, keinen Grund. Das ist die akzeptierte Konsequenz aus §1 und
wird hier festgehalten, damit es später nicht als Bug wiederentdeckt wird.

Daraus folgt auch: **kein Audit.** Wer wann mit welcher Begründung gelöscht hat, ist
nachträglich nicht rekonstruierbar. `carpooling_events.actor_name` (Migration 001) zeigt,
dass es im Projekt ein Muster für „Aktor beim Ereignis festhalten" gibt — hier wird es
bewusst **nicht** angewandt.

## 3. Der leere Link ist im Service Worker nicht neutral

Die Anforderung „kein Direktlink" lässt sich nicht allein durch `url: ""` erfüllen:

```
   web/src/sw.ts:147-160
   const url = (event.notification.data as { url: string })?.url ?? '/'
                                                              ▲
                                          ?? greift bei null/undefined —
                                          NICHT beim leeren String
   …
   existing.navigate(url)      ──▶ navigate("") löst relativ gegen die
                                   Client-URL auf ⇒ Reload der Seite,
                                   auf der der Nutzer gerade stand
   self.clients.openWindow(url)    ohne offenes Fenster: undefiniert
```

Ein Klick auf die Absage-Push würde also die gerade offene Seite neu laden — schlimmer als
der heutige Sprung in den Kalender, weil es wie ein Fehler wirkt.

**Lösung:** der Handler trennt „kein Ziel" von „Ziel unbekannt".

```
   data.url
     │
     ├── leer / fehlt  ──▶  Fenster fokussieren (bzw. "/" öffnen,
     │                      wenn keines existiert) — NICHT navigieren
     │
     └── gesetzt       ──▶  wie bisher: focus + navigate(url)
```

Die E-Mail-Seite ist bereits korrekt: `sendCategoryEmail` (`internal/notify/notify.go`)
prüft `if url != ""` und lässt die „Direktlink"-Zeile dann weg. Es ist ausschließlich der
Service Worker, der nachzieht.

`sw.ts` hat heute keine Testabdeckung. Der `notificationclick`-Handler wird dafür so
umgebaut, dass die Zielauflösung in einer exportierten, reinen Funktion steckt
(`resolveClickTarget(data) → { navigate: boolean; url: string }`) — testbar ohne
Service-Worker-Laufzeit, analog zur bereits exportierten Factory in `VideoUploadPage.tsx`.

## 4. `silent` ist eine eigene Capability

Löschrecht und Stummschaltrecht fallen auseinander:

```
   LÖSCHEN DARF                          UNTERDRÜCKEN SOLL
   ─────────────                         ──────────────────
   admin                                 admin
   vorstand                              vorstand
   trainer               ┐
   sportliche_leitung    ┘ ← fallen raus

   policy.CanDeleteGame  = CanEditGame       (4 Personas, rules.go:79-87)
   trainings.hasTeamAccess                    (handler.go:133-145)
   policy.IsVorstandLike = admin | vorstand   (rules.go:40-42)
```

Eine echte Teilmenge, also eine neue Capability `CapSuppressEventNotification`
(`suppress_event_notification`), gegated auf `IsVorstandLike`. Vorhandene lassen sich nicht
recyceln:

- `manage_games` hat der Trainer ebenfalls (`Capabilities`, `rules.go:216`).
- `manage_trainings` ist `IsTrainerLike` und **schließt den reinen Vorstand aus**. Das
  Frontend muss die Checkbox deshalb an der neuen Capability festmachen, nicht an einer
  vorhandenen.

### 4a. Bei Trainings greift die Unterdrückung faktisch nur für Admins

Beim Umsetzen kam heraus, dass die Reichweite bei Trainings kleiner ist als hier zunächst
angenommen. Die drei Trainings-Routen hängen im Router an einem **engeren Tier** als die
Spiel- und Dienst-Routen:

```
   internal/app/router.go
   ────────────────────────────────────────────────────────────────
   :440  RequireClubFunction("vorstand", "trainer", "sportliche_leitung")
           └─ DELETE /api/games/{id}          ← reiner Vorstand kommt durch ✓
           └─ DELETE /api/duty-slots/{id}     ← reiner Vorstand kommt durch ✓

   :415  RequireClubFunction("trainer", "sportliche_leitung")
           └─ DELETE /api/training-sessions/{id}   ← 403 für reinen Vorstand ✗
           └─ DELETE /api/training-series/{id}     ← 403 für reinen Vorstand ✗
           └─ PUT    /api/training-sessions/{id}   ← 403 für reinen Vorstand ✗
```

`RequireClubFunction` hat einen Admin-Bypass (`auth/middleware.go:97`), aber keinen für
`vorstand`. Der Router weist also **vor** dem Handler ab — womit der `vorstand`-Zweig in
`trainings.hasTeamAccess` (`handler.go:134`) für diese Routen toter Code ist. Die frühere
Formulierung in §4, `hasTeamAccess` lasse den Vorstand „beim Löschen durch", stimmt für den
Handler, ist für den Request aber ohne Wirkung.

**Konsequenz:** Bei Trainings kann `silent` praktisch nur ein **Admin** nutzen — oder ein
Vorstand, der zusätzlich `trainer`/`sportliche_leitung` trägt. Bei Spielen und Dienst-Slots
gilt die Regel wie spezifiziert.

Das wird hier **bewusst nicht geheilt**. Den Trainings-Tier um `vorstand` zu erweitern,
gäbe dem Vorstand nebenbei das Recht, sämtliche Trainings anzulegen, zu ändern und zu
löschen — eine Berechtigungsausweitung, die mit Benachrichtigungstexten nichts zu tun hat
und eine eigene Entscheidung braucht.

**Fail-safe-Regel:** ein `silent: true` ohne Capability wird ignoriert, nicht mit 403
quittiert. Begründung: die Löschung selbst ist erlaubt; sie an einem Zusatzflag scheitern zu
lassen, würde einen Trainer bei einem legitimen Vorgang blockieren. Benachrichtigen ist der
sichere Default — im Zweifel erfährt es das Team.

## 5. Wo der Grund im Request steckt

`DELETE` mit JSON-Body. Go und Chi verarbeiten das ohne Sonderbehandlung, axios über
`api.delete(url, { data })`.

Die Alternative — Query-Parameter — wurde geprüft und verworfen. Sie wäre nicht einmal ein
Logging-Risiko (`internal/health/recover.go:38` protokolliert `r.URL.Path`, **nicht**
`RawQuery`), aber ein Freitext mit Umlauten und Satzzeichen gehört nicht in eine URL, und
`silent` als zweites Feld würde die Signatur ohnehin zu einem Objekt machen.

**Rückwärtskompatibilität ist Pflicht:** ein fehlender oder leerer Body muss weiter zu einem
Erfolg führen. Alle vier Handler dekodieren deshalb tolerant — Decode-Fehler (inklusive
`io.EOF` bei leerem Body) führen zu „kein Grund, nicht stumm", nicht zu HTTP 400. Das hält
alte PWA-Installationen im Service Worker-Cache funktionsfähig.

**Kürzung statt Ablehnung:** `reason` wird getrimmt und auf 200 Zeichen (Runen, nicht Bytes
— Umlaute) gekürzt, ohne Fehler. Ein 400 wäre hier feindlich: der Nutzer hat gerade auf
„Löschen" geklickt, die Aktion ist unwiderruflich gedacht, und ein zu langer Grund darf sie
nicht abbrechen. 200 orientiert sich am bestehenden `CHECK (length(note) <= 200)` auf
`games.note` (Migration 011) und daran, dass Android/iOS Push-Bodies ohnehin abschneiden.

## 6. Textbausteine

Ein gemeinsamer Helfer setzt die Sätze zusammen, damit die fünf Aufrufstellen nicht
divergieren. Er lebt in `internal/notify` (Foundation, importiert keine Domäne) und nimmt
nur Strings entgegen:

```go
// notify.CancellationBody("HSG Ostfildern", "14.09.2026", "Tim Meier", "Halle gesperrt")
//   → "HSG Ostfildern am 14.09.2026 entfällt. Abgesagt von Tim Meier: Halle gesperrt."
//
// ohne Grund:
//   → "HSG Ostfildern am 14.09.2026 entfällt. Abgesagt von Tim Meier."
```

Damit ist Invariante 1 („keine Meldung ohne Substantiv") an einer Stelle prüfbar statt an
fünf.

Aktor-Name: `SELECT first_name, last_name FROM users WHERE id = ?` (`users`, Migration 001,
Zeilen 500-501). Ist beides leer, fällt der Satz auf „Abgesagt von einem Trainer." zurück —
`users.email` wäre hier eine PII-Preisgabe an das ganze Team und wird nicht verwendet.

**Pro Route:**

| Stelle | Titel | Body | `url` |
|---|---|---|---|
| `DeleteGame` → Team | `Spiel abgesagt` | `CancellationBody(opponent, date, actor, reason)` | `""` |
| `DeleteGame` → Dienste | `Dienst entfällt` | bisheriger Satz + ` Abgesagt von {actor}: {reason}` | `/dienste` |
| `DeleteSession` | `Training abgesagt` | `CancellationBody(title|"Training", date, actor, reason)` | `""` |
| `DeleteSeries` | `Trainingsserie beendet` | `CancellationBody(name, "ab {from}", actor, reason)` | `""` |
| `DeleteSlot` | `Dienst abgesagt` | `CancellationBody("{duty_type} zum {event_name}", event_date, actor, reason)` | `/dienste` |
| `UpdateSession` bei `active→cancelled` | `Training abgesagt` | `CancellationBody(title, date, actor, cancel_reason)` | `/termine?focus=training-{id}` |

Die beiden `/dienste`-Links bleiben, weil die Dienstbörse nach der Löschung existiert — nur
der eine Slot ist weg. Der Nutzer landet dort, wo er sich neu eintragen kann. Das ist der
Unterschied zu `/termine`, wo er nichts findet.

## 7. Was jede Aufrufstelle zusätzlich lesen muss

| Stelle | liegt schon vor | muss neu geladen werden |
|---|---|---|
| `DeleteGame` | `opponent`, `eventDate` (`handler.go:1414-1421`) | Aktor-Name |
| `DeleteSession` | `teamID` | `title`, `date` (heute wird nur `team_id` gelesen, `:625`) |
| `DeleteSeries` | `teamID` | `name`, `valid_from`/`valid_until` (heute nur `team_id`, `:542`) |
| `DeleteSlot` | `assigned`, `teamIDs` | `event_name`, `event_date`, Name der Dienstart |
| `UpdateSession` | `teamID`, `req.CancelReason` | **alter** `status` vor dem `UPDATE` |

Der letzte Punkt ist die einzige Stelle mit einer Reihenfolge-Falle: `UpdateSession` liest
den Status heute gar nicht, und `req.Status` fällt bei fehlendem Feld auf `"active"` zurück
(`handler.go:828-831`) — ein PUT ohne `status` reaktiviert also eine abgesagte Einheit.
Dieses Verhalten bleibt unverändert; für die Meldung zählt allein der Vergleich
`alt != neu && neu == "cancelled"`. Ohne diesen Vergleich würde jede spätere Korrektur an
einer abgesagten Einheit erneut „Training abgesagt" an das Team senden.

## 8. Was sich am Broadcast-Verhalten nicht ändert

Alle fünf Routen behalten ihren `Broadcast`-Aufruf unverändert. `silent: true` betrifft
ausschließlich `notify.Send` (Push/E-Mail an Betroffene), **nie** das SSE-Live-Update —
sonst würden offene Sessions einen gelöschten Termin weiter anzeigen. Das Broadcast-Gate
(`internal/arch/broadcast_test.go`) bleibt damit ohne neuen Allowlist-Eintrag grün.
