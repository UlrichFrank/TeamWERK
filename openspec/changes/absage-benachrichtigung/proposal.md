## Why

Wird ein Termin gelöscht, bekommt das Team eine Benachrichtigung ohne Inhalt:

> **Spiel abgesagt** — Ein Spiel wurde abgesagt.
> Direktlink: https://teamwerk.team-stuttgart.org/termine

Drei Informationen fehlen, und die vierte ist falsch:

1. **Welches Spiel?** Der Text nennt weder Gegner noch Datum. Wer in zwei Mannschaften
   Kinder hat, weiß nicht, welcher Termin gemeint ist.
2. **Durch wen?** Nirgends steht, wer gelöscht hat — also auch nicht, wen man fragen kann.
3. **Warum?** Es gibt kein Feld dafür. Halle gesperrt, Gegner tritt nicht an, Schnee — alles
   dieselbe Meldung.
4. **Der Direktlink ist sinnlos.** Er zeigt auf den Kalender, in dem der Termin gerade
   verschwunden ist. Wer klickt, sucht etwas, das es nicht mehr gibt.

Bemerkenswert ist, dass die Daten schon bereitliegen. Zwei Zeilen über der nichtssagenden
Meldung baut derselbe Handler für die Dienst-Zuständigen einen brauchbaren Text
(`internal/games/handler.go:1521-1528`):

```go
body := fmt.Sprintf("Dein Dienst zum %s am %s wurde gelöscht.", eventName, formatDateDMY(eventDate))
notify.Send(…, "duties", "Dienst entfällt", body, "/dienste")

notify.Send(…, "games", "Spiel abgesagt", "Ein Spiel wurde abgesagt", "/termine")
//                                        ^^^ opponent + eventDate sind hier in scope
```

Dasselbe Muster zieht sich durch: `trainings/handler.go:591` („Eine Trainingsserie wurde
beendet"), `:631` („Eine Trainingseinheit wurde abgesagt"), `duties/handler.go:533` („Ein
Dienst, für den du eingetragen warst, wurde abgesagt"). Vier Meldungen ohne Substantiv.

Dazu kommt ein vertauschtes Paar bei den Trainings: Wer eine Einheit in der UI **absagt**
(`status='cancelled'` + `cancel_reason`, `trainings/handler.go:885`), löst „Training
**geändert** — Eine Trainingseinheit wurde aktualisiert" aus. Wer sie **löscht** (`:631`),
löst „Training **abgesagt**" aus. Die semantisch reichere Aktion sendet die schwächere
Nachricht — und der bereits erfasste `cancel_reason` wird nicht mitgeschickt, obwohl die
Oberfläche ihn längst anzeigt (`TermineDetailPage.tsx:427`).

## What Changes

- **Optionales Grundfeld** in allen Lösch-Modals (Spiel, Trainingseinheit, Trainingsserie,
  Dienst-Slot). Der Grund reist im Body des `DELETE`-Requests mit, wird auf 200 Zeichen
  **still gekürzt** und landet ausschließlich im Benachrichtigungstext.
- **Konkreter Meldungstext** statt Platzhalter: Gegner/Titel + Datum + Aktor-Name + Grund.
- **Kein Direktlink bei Löschungen.** Die Team-Meldung geht mit `url: ""` raus. Der
  Dienst-Meldung bleibt `/dienste` erhalten — die Dienstbörse existiert weiter.
- **Service Worker behandelt den leeren Link.** `web/src/sw.ts:149` fängt heute nur
  `null`/`undefined` ab (`?? '/'`), nicht den leeren String; `navigate("")` würde die
  gerade offene Seite neu laden. Künftig: bei leerem `url` nur fokussieren bzw. öffnen,
  nicht navigieren.
- **Benachrichtigung unterdrückbar** für Admin und Vorstand über ein Häkchen im Modal
  (`silent: true`). Deckt beide Meldungen einer Löschung ab (Team **und** Dienste). Gedacht
  für Korrekturen — Import-Dubletten, Tippfehler-Termine —, bei denen eine Absage-Push an
  Spieler und Eltern reines Rauschen wäre.
- **Neue Capability `suppress_event_notification`** (`policy.IsVorstandLike` = Vorstand +
  Admin). Bewusst enger als das Löschrecht selbst: `CanDeleteGame` schließt auch `trainer`
  und `sportliche_leitung` ein, `hasTeamAccess` bei Trainings ebenso. Ein Trainer darf
  löschen, aber nicht stumm löschen.
- **Fail-safe statt 403:** Schickt jemand ohne die Capability `silent: true`, wird das Flag
  **ignoriert und trotzdem benachrichtigt**. Die Löschung selbst ist erlaubt und soll nicht
  an einem Zusatzflag scheitern; Benachrichtigen ist der sichere Default.
- **Trainings-Absage sendet die Absage-Meldung.** Beim Übergang `active → cancelled` in
  `PUT /api/training-sessions/{id}` geht „Training abgesagt" samt `cancel_reason` raus, mit
  dem **funktionierenden** Link `/termine?focus=training-{id}`. Nur beim Übergang — ein
  weiteres PUT auf eine bereits abgesagte Einheit meldet wie bisher „Training geändert".
- **Keine Migration, kein Schemawechsel.** Der Löschgrund wird bewusst nirgends persistiert.

## Nicht Teil dieses Changes

- **Kein `games.status = 'cancelled'`.** Löschen bleibt Löschen; ein abgesagtes Spiel, das
  doch stattfindet, wird neu angelegt. Begründung und die verworfene Alternative stehen in
  `design.md` §1.
- **Kein Audit-Trail.** Weil der Datensatz verschwindet, existiert der Grund nur in der
  zugestellten Nachricht. Die Konsequenz ist in `design.md` §2 offen benannt.
- **Der Vorrang-Konflikt zwischen `CanManageTrainings` und `hasTeamAccess`** (reiner
  Vorstand hat die Capability nicht, kommt über den Route-Guard aber durch) wird hier nur
  umgangen, nicht behoben — siehe `design.md` §4.

## Capabilities

### Added Capabilities

- **`absage-benachrichtigung`** — Das Querschnittsverhalten: Absage-Meldungen benennen
  Termin und Auslöser, nehmen einen optionalen Löschgrund entgegen (still auf 200 Zeichen
  gekürzt, nirgends persistiert), behandeln eine leere Ziel-URL als „kein Ziel" statt als
  kaputtes Ziel, und Vorstand/Admin können den Versand für Korrekturen unterdrücken.

### Modified Capabilities

Die drei bestehenden Push-Capabilities schreiben das heutige Verhalten wörtlich fest und
werden von diesem Change direkt widerlegt — sie müssen mitziehen, sonst steht die Spec
gegen den Code:

- **`push-games`** — sagt heute: „Für gelöschte Spiele (kein navigierbarer Termin mehr)
  zeigt die `url` auf `/termine`." Künftig: leerer String, plus Gegner/Datum/Aktor/Grund im
  Body und die Unterdrückbarkeit.
- **`push-trainings`** — sagt heute dasselbe für gelöschte Einheiten und Serien, und ordnet
  jedem `PUT` pauschal „Training geändert" zu. Künftig: leerer String bei Löschungen,
  „Training abgesagt" beim Statuswechsel `active → cancelled` — mit **erhaltenem**
  `focus`-Link, weil die abgesagte Einheit weiter existiert.
- **`push-duties`** — behält beide `/dienste`-Links (die Dienstbörse überlebt die
  Löschung), bekommt aber Dienstart/Event/Datum/Aktor/Grund in den Body und die
  Unterdrückbarkeit.

## Test-Anforderungen

**Routen** (alle bestehen bereits; kein neuer Endpoint):

| Route | Fall | Erwartung |
|---|---|---|
| `DELETE /api/games/{id}` | Vorstand, `{"reason":"Halle gesperrt"}` | 204/200; Team-Meldung enthält Gegner, Datum, Aktor-Name, Grund; `url == ""` |
| | Vorstand, ohne `reason` | wie oben ohne Grund-Satz, kein leeres „Grund: " |
| | Vorstand, `{"silent":true}` | 0 Benachrichtigungen (Team **und** Dienste) |
| | Trainer, `{"silent":true}` | Flag ignoriert, Benachrichtigungen gehen raus |
| | `reason` mit 500 Zeichen | 200, Text enthält genau 200 Zeichen Grund, **kein** 400 |
| | leerer Body / kein Body | 200 wie bisher (Rückwärtskompatibilität) |
| | fremdes Team, reiner Trainer | 403, keine Benachrichtigung |
| | unbekannte ID | 404, keine Benachrichtigung |
| `DELETE /api/training-sessions/{id}` | Vorstand mit `reason` | Titel/Datum + Aktor + Grund, `url == ""` |
| | Trainer mit `silent:true` | Flag ignoriert |
| | unbekannte ID | 404 |
| `DELETE /api/training-series/{id}` | Vorstand mit `reason` | Serienname + Zeitraum + Aktor + Grund, `url == ""` |
| | unbekannte ID | 404 |
| `DELETE /api/duty-slots/{id}` | Vorstand mit `reason` | Dienstname + Event + Datum + Aktor + Grund, Link `/dienste` **bleibt** |
| | `silent:true` von Vorstand | 0 Benachrichtigungen |
| | unbekannte ID | 404 |
| `PUT /api/training-sessions/{id}` | `status: active → cancelled` | „Training abgesagt" + `cancel_reason`, Link `/termine?focus=training-{id}` |
| | `status: cancelled → cancelled` | „Training geändert" (keine zweite Absage-Meldung) |
| | `status: cancelled → active` | „Training geändert" |
| | ungültiger `status` | 400 wie bisher |

**Garantierte Invarianten** (jede bekommt einen eigenen Test):

1. **Keine Absage-Meldung ohne Substantiv.** Für jede der vier Löschrouten gilt: der
   Benachrichtigungs-Body enthält den Namen des Termins **und** ein Datum im Format
   `TT.MM.JJJJ`. Der Test prüft den Text, nicht nur den Aufruf.
2. **Löschmeldungen verlinken nicht ins Leere.** Für alle drei Termin-Löschrouten ist der
   an `notify.Send` übergebene `url` der leere String. Ein Test über alle drei Aufrufe
   zugleich, damit eine neue Route nicht stillschweigend `/termine` erbt.
3. **`silent` ist ein Recht, kein Wunsch.** Derselbe Request einmal als Vorstand
   (0 Benachrichtigungen) und einmal als Trainer (alle Benachrichtigungen) — bei
   identischem Body und in beiden Fällen HTTP-Erfolg.
4. **Stumm heißt vollständig stumm.** `silent: true` auf ein Spiel mit eingetragenen
   Dienst-Zuständigen unterdrückt **beide** Meldungen (`games` und `duties`), nicht nur
   die Team-Meldung.
5. **Der Grund wird nicht persistiert.** Vollständiger DB-Scan nach einer Löschung mit
   einem eindeutigen Marker-Grund findet den String in **keiner** Tabelle und **keiner**
   Spalte — inklusive Poison-Sanity, die beweist, dass der Scanner scharf ist (Vorbild:
   `TestPreviewH4A_CredentialsWerdenNichtPersistiertOderGeloggt`).
6. **Kürzung statt Ablehnung.** 500-Zeichen-Grund → HTTP-Erfolg, exakt 200 Zeichen im Text.
7. **Genau ein Absage-Signal pro Statuswechsel.** Zweimal hintereinander
   `status: 'cancelled'` per PUT → genau eine „Training abgesagt"-Meldung.
8. **Leerer Link navigiert nicht.** Vitest auf den `notificationclick`-Handler: bei
   `data.url === ""` wird `focus()` aufgerufen und `navigate()` **nicht**.
