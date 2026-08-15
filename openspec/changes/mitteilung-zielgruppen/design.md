# Design — Mitteilungs-Zielgruppen

## 1. Warum die Ordner-ACL nicht wiederverwendet wird

Der ursprüngliche Gedanke war, die Zielgruppen als „Teilauswahl des
Berechtigungssystems von `/dokumente`" zu bauen. Das trägt nicht, aus zwei Gründen.

**Die Auflösung läuft in die Gegenrichtung.**

```
   DOKUMENTE (pull)                     MITTEILUNGEN (push)
   ════════════════                     ═══════════════════
   Nutzer öffnet Ordner                 Absender drückt "Senden"
          │                                      │
          ▼                                      ▼
   FolderAccess(db, principal, id)      resolveRecipients(target)
          │                                      │
   "Darf DIESER Nutzer?"                "WER sind alle Nutzer?"
          │                                      │
          ▼                                      ▼
   Antwort gilt JETZT                   N Zeilen broadcast_reads,
   (Kaderwechsel wirkt sofort)          eingefroren beim Senden
```

`policy.FolderAccess` beantwortet ein Prädikat für ein bekanntes Subjekt. Die
Kernqueries (`playerTeamsQuery`, `parentTeamsQuery`) sind auf `WHERE m.user_id = ?`
verdrahtet und beantworten die Mengenfrage nicht. Eine Wiederverwendung hieße, einen
zweiten Satz Queries daneben zu legen — mit dem Risiko, dass zwei Definitionen von
„Spieler" auseinanderdriften, ohne dass ein Test das merkt.

**Und die Vokabulare überschneiden sich nach diesem Change kaum noch.** Von den sechs
Ordner-Principals bleibt nichts Gemeinsames übrig: `role` (admin/standard) ist für
Mitteilungen sinnlos, `user` ist ein Direktchat, `team` und `team_parents` entfallen
bewusst (§2), `club_function` wird für `spieler` zwar inhaltlich gespiegelt, aber ohne
die Eltern-Erbung (§3). Es bleibt `everyone` ≈ `users`.

**Entscheidung:** eigenes, kleines Zielgruppen-Vokabular in `internal/chat`, vier feste
Werte, reines SQL über `users` / `members` / `member_club_functions` / `family_links`.
Kein Import aus `policy`, kein gemeinsamer Resolver. Der Preis ist eine bewusste
Doppelung des Begriffs „Spieler" an zwei Stellen; der Gewinn ist, dass keine der beiden
Seiten die andere versehentlich mitverändert.

**Verworfen:** einen globalen Principal `parents` zusätzlich in `folder_permissions`
aufzunehmen, um die Vokabulare synchron zu halten. Das hätte `/dokumente` für einen
Nutzen geändert, den dort niemand angefragt hat.

## 2. Warum „Team" ersatzlos entfällt

`internal/chat/team_groups.go` liefert bereits pro Team der aktiven Saison drei
Standardgruppen mit den Kinds `trainer` / `spieler` / `eltern`. Nach dem Change stehen
die beiden Kanäle sauber orthogonal:

```
                 pro Team                    vereinsweit
              ┌──────────────────┐      ┌──────────────────┐
   Chat       │ Gruppe: Trainer  │      │        —         │
   (Dialog)   │ Gruppe: Spieler  │      │                  │
              │ Gruppe: Eltern   │      │                  │
              └──────────────────┘      └──────────────────┘
              ┌──────────────────┐      ┌──────────────────┐
   Mitteilung │        —         │      │ Nutzer           │
   (Ansage)   │  (entfällt)      │      │ Mitglieder       │
              │                  │      │ Spieler / Eltern │
              └──────────────────┘      └──────────────────┘
```

Heute überlappen die linken Kästen; danach nicht mehr. Der Trainer verliert das
Broadcast-Recht, bekommt aber keinen Kanal weniger — „Gruppe: Spieler" seines Teams ist
derselbe Empfängerkreis, nur mit Rückkanal.

**Das ist trotzdem eine wegnehmende Änderung an einer produktiv genutzten Fähigkeit.**
Trainer, die heute `targetType: team` verwenden, finden den Button nach dem Deploy nicht
mehr vor. Die Ankündigung an die Trainer ist Teil der Tasks, nicht optional.

## 3. Warum `spieler` keine Eltern erbt

`policy.FolderAccess` lässt Eltern die Vereinsfunktionen ihrer Kinder erben — eine
`club_function: spieler`-ACL-Zeile matcht auch auf den Elternaccount
(`folders.go`, `case "club_function"` über `ctx.family()`). Für Ordner ist das richtig:
ein 9-jähriger hat oft keinen eigenen Zugang, und die Vererbung ist der einzige Weg, wie
das Trainingslager-PDF zuhause ankommt.

Für Mitteilungen ist es falsch, weil `eltern` als eigene Zielgruppe daneben steht. Mit
Vererbung wäre „Alle Spieler" eine Obermenge von „Alle Eltern" und die Auswahl damit
irreführend:

```
   MIT Vererbung (verworfen)          OHNE Vererbung (gewählt)
   ┌───────────────────────┐         ┌──────────┐  ┌──────────┐
   │  spieler              │         │ spieler  │  │  eltern  │
   │  ┌──────────┐         │         └──────────┘  └──────────┘
   │  │  eltern  │         │           disjunkt, beide Auswahlen
   │  └──────────┘         │           bedeuten, was sie sagen
   └───────────────────────┘
     "Eltern" wäre sinnlos
```

**Bekannte Konsequenz, bewusst getragen:** ein Spieler ohne eigenen Account wird von
„Alle Spieler" nicht erreicht, und seine Eltern auch nicht. Wer beide Gruppen meint,
sendet zweimal oder wählt „Alle Mitglieder". Der `recipients`-Zähler (§5) macht die
Lücke wenigstens sichtbar, statt sie zu verschweigen.

## 4. Warum genau eine Zielgruppe

Eine Mehrfachauswahl („Spieler ☑ Eltern ☑") würde §3 entschärfen, macht `broadcasts`
aber zu einer 1:n-Beziehung: `target_type` müsste einer Zeilentabelle
`broadcast_targets` weichen, mit Deduplikation über die Vereinigungsmenge und einer
Anzeige, die mehrere Zielgruppen darstellen kann.

Der Nutzen wäre marginal — „Alle Mitglieder" deckt den einzig häufigen Kombinationsfall
(Spieler + Eltern) bereits ab, mit dem Unterschied, dass auch Vorstand und Kassierer
mitlesen. Das ist bei einer Vereinsansage kein Schaden.

**Entscheidung:** ein Wert in einer Spalte. Sollte sich Mehrfachauswahl später als nötig
erweisen, ist der Weg dorthin additiv (Zeilentabelle daneben, `target_type` als
Ein-Zeilen-Fall migrieren) und nicht durch diesen Change verbaut.

## 5. Der Empfängerzähler statt einer Vorschau

Der Bug, der diesen Change ausgelöst hat, war nicht „falsche Query" — den gibt es
überall. Er war **stumm**: ein Fan-out auf null Empfänger sieht für den Absender exakt
aus wie ein erfolgreicher. Eine korrigierte Query allein stellt sicher, dass *dieser*
Fall behoben ist; sie stellt nicht sicher, dass der nächste auffällt.

Die Antwort auf `POST /api/chat/broadcasts` trägt deshalb `recipients`. Das Frontend
meldet „An 183 Empfänger gesendet" — und bei 0 entsprechend deutlich.

**Verworfen: ein Preview-Endpoint** (`GET /api/chat/broadcasts/recipient-count?target=…`),
der die Zahl schon bei der Auswahl im Dropdown zeigt. Schöner, aber eine zusätzliche
Route mit eigener Autorisierung für einen Nutzen, den die Antwort nach dem Senden
größtenteils auch liefert. Bleibt eine mögliche Erweiterung.

**Verworfen: HTTP 409 bei null Empfängern.** Eine leere Zielgruppe kann legitim sein
(Verein ohne erfasste Eltern), und eine Mitteilung an niemanden richtet keinen Schaden
an. Sichtbarkeit genügt, Blockade wäre übergriffig.

## 6. Migration: was mit den Bestandszeilen passiert

Entscheidend für die Risikoeinschätzung: **`broadcasts.target_type` wird nirgends
gelesen.** `ListBroadcasts` selektiert die Spalte nicht, `EditBroadcast` fasst sie nicht
an, `DeleteBroadcast` auch nicht. Sie ist reine Schreibspur, ausgewertet einmal beim
Fan-out. `target_id` trägt keinen Foreign Key (`028_chat_broadcast_media.up.sql:43`).

Die Zustellung selbst hängt ausschließlich an `broadcast_reads` — und die Tabelle wird
nicht angefasst. Alte Mitteilungen bleiben für ihre Empfänger unverändert sichtbar,
lesbar und löschbar.

```
   vorher                          nachher
   ──────                          ───────
   target_type 'all'    ────────▶  'users'     (semantisch identisch)
   target_type 'team'   ────────▶  'legacy'    (target_id fällt weg)
   target_type 'role'   ────────▶  'legacy'    (hat nie zugestellt)
   target_id            ────────▶  entfällt
   target_role          ────────▶  entfällt
```

`'legacy'` ist im CHECK erlaubt, wird aber vom Handler beim Schreiben abgelehnt — die
Spalte bleibt für Bestandszeilen ehrlich, ohne einen fünften Zielwert einzuführen, den
jemand auswählen könnte.

**Bewusster Verlust:** für die auf `'legacy'` gesetzten Zeilen ist nachträglich nicht
mehr erkennbar, an welches Team sie gingen. Akzeptabel, weil die Information nie
angezeigt wurde und die Empfängermenge in `broadcast_reads` vollständig erhalten bleibt.

SQLite kann Spalten nicht ohne Weiteres mit geändertem CHECK entfernen; die Migration
folgt dem Muster aus `028` (Tabelle neu anlegen, `INSERT … SELECT` mit `CASE`, alte
Tabelle droppen, umbenennen).

## 7. Capability-Kollaps

Mit sportlicher Leitung auf demselben Niveau wie Vorstand und Admin unterscheidet
`broadcast_all` keine zwei Mengen mehr:

```
   broadcast_messages  →  admin | vorstand | sportliche_leitung
   broadcast_all       →  admin | vorstand                       ← trennt nichts mehr,
                                                                   jeder Absender ist
                                                                   ohnehin vorstand-like
```

`broadcast_all` wird deshalb gestrichen, nicht bloß im Composer ignoriert. Eine
Capability, die im `GET /api/me`-Vokabular steht und nichts mehr steuert, ist eine
Falle für den nächsten Leser.

Der Grep-Test (Invariante 7 im Proposal) ist die Absicherung gegen den typischen
Restfehler: ein `hasCapability("broadcast_all")` bleibt im Frontend stehen, liefert nach
der Backend-Änderung dauerhaft `false`, und ein Button ist für alle unsichtbar, ohne dass
irgendetwas fehlschlägt — wieder ein stiller Fehler derselben Familie.

## 8. Was diese Änderung für den Folge-Change vorbereitet

`mitteilung-lesebestaetigung` (Anzeige `47 / 183 gelesen`) braucht eine vertrauenswürdige
Empfängermenge als Nenner. Solange eine Zielgruppe still auf null auflöst, wäre der Nenner
falsch, ohne dass es auffiele. Deshalb ist die Reihenfolge festgelegt: erst dieser Change,
dann die Lesebestätigung.
