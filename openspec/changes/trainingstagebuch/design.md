## Context

Vier Bestandsteile bestimmen dieses Design — drei tragen es, einer ist der Grund für eine
Abweichung.

1. **`internal/trainings` ist bereits belegt** und meint etwas anderes: terminierte
   Mannschaftseinheiten (`training_series`, `training_sessions`) mit RSVP und Trainer-erfasster
   Anwesenheit. Das Tagebuch ist strukturell das Gegenteil — selbstberichtet, ohne Termin, ohne
   Zu-/Absage, ohne Fremderfassung. Eigenes Package, eigene Tabelle, kein gemeinsames Modell.
2. **`attendance.canSeeMemberStats`** (`internal/attendance/handler.go:342`) beantwortet exakt die
   Frage „wer darf die Zahlen dieses Spielers sehen" und löst dabei Trainer über
   `trainer_memberships` × `kader` gegen die **aktive Saison** auf. Diese Logik wird gespiegelt,
   nicht neu erfunden.
3. **`scheduler.deleteRetainedVideos`** (`internal/scheduler/scheduler.go:357`) ist das
   Retention-Muster des Hauses: tägliche, von Natur aus idempotente Löschung entlang
   `seasons.end_date`, Inline-SQL, `NULL`-Saison bedeutet „nie löschen".
4. **`media.Serve`** (`internal/media/handler.go:134`) prüft ausschließlich, dass ein gültiges JWT
   vorliegt. Wer `/api/media/1..N` durchzählt, bekommt jedes Chat-Bild des Vereins. Für Chat ist
   das offenbar akzeptiert; für Trainingsnachweise ist es das nicht — daher ein eigener Store.

## Goals / Non-Goals

**Goals**
- Der Spieler erfasst in unter 30 Sekunden eine Einheit, auch ohne Nachweis.
- Der Trainer sieht auf einen Blick, wer in seiner Mannschaft etwas tut — und kann hineinzoomen.
- Nachweisbilder verschwinden ohne Zutun, sobald sie ihren Zweck erfüllt haben.
- Kein Spieler sieht die Daten eines anderen Spielers.

**Non-Goals**
- Keine Kopplung an `training_sessions` und keine Einrechnung in `attendance-statistics`.
- Keine Freigabe-/Ablehnungs-Workflows auf Nachweisen. Fehlt ein Nachweis, ist das kein Fehler.
- Keine sRPE-Trainingslast (`Dauer × RPE`) — sportwissenschaftlich naheliegend, aber ohne
  Leseanleitung nur eine hübsche Zahl. Später, wenn Daten da sind.
- Keine serverseitige Bildkonvertierung (siehe Entscheidung 4).

## Decisions

### 1. Saison-Anker: die aktive Saison bei Erfassung, nicht das Trainingsdatum

Die Retention hängt an `seasons.end_date`. Die naheliegende Zuordnung wäre
`trained_on BETWEEN start_date AND end_date` — sie hat aber ein Loch. `internal/config/handler.go`
legt Saisons als freie Zeiträume an, ohne Lücken- oder Überlappungsprüfung:

```
   Saison 25/26                      Saison 26/27
   ├──────────────────────┤          ├──────────────────────┤
   01.08.25        31.05.26          01.08.26        31.05.27
                          └────┬─────┘
                            SOMMERPAUSE
                          Juni + Juli 26
```

Einträge aus der Sommerpause fielen in **keine** Saison — ausgerechnet die Zeit, in der
Eigentraining am meisten zählt und die einen Hauptanlass für dieses Feature darstellt. Sie hätten
keinen Retention-Anker und blieben ewig liegen.

Deshalb: `season_id` = die Saison, die bei der Erfassung `is_active = 1` trägt. Eine Saison bleibt
aktiv, bis der Vorstand umschaltet; die Sommerpause gehört damit der auslaufenden Saison, und kein
Eintrag fällt durch. Das deckt sich mit `videos.season_id` und mit der Hausregel „ohne aktive
Saison geht nichts".

**Trade-off:** Schaltet der Vorstand mitten im Juli um, bekommen ab dann erfasste Einträge die neue
Saison — auch wenn `trained_on` davor liegt. Fachlich ist das sogar richtig (die Vorbereitung
gehört zur neuen Saison), und die Retention verschiebt sich dadurch nur nach hinten, nie nach vorn.

**Fallback:** Existiert keine aktive Saison, wird `season_id = NULL` gespeichert. Solche Einträge
werden **nie** automatisch bereinigt — dieselbe fail-safe-Richtung wie bei den Videos.

### 2. Eigener Store statt `internal/media`

`media` liefert Bytes an jeden Eingeloggten. Trainingsnachweise sind persönliche Belege, oft von
Minderjährigen und potenziell mit Gesundheitsbezug (Reha-Übungen, Physio-Pläne,
Fitness-App-Screenshots mit Herzfrequenz). Sie brauchen eine Prüfung pro Objekt.

Konsequenz: eigenes Verzeichnis `TRAINING_DIARY_DIR`, Dateiname `<uuid>.<ext>`, Metadaten als
Spalten am Eintrag statt als Zeile in `media`. Ausgeliefert wird über
`GET /api/training-diary/{id}/proof`, das dieselbe ACL wie die Einträge selbst anwendet.

**Ein Nachweis pro Eintrag → Spalten, keine Kindtabelle.** Eine `1:n`-Tabelle wäre allgemeiner,
aber die Anforderung ist ausdrücklich „ein Nachweis je Training". Spalten sparen einen Join in
jeder Liste und einen Lebenszyklus.

Im Frontend liefert `AuthImage` das Bild (axios → Blob → Object-URL), weil `<img src>` keinen
`Authorization`-Header sendet — dasselbe Muster wie im Chat, nur hinter einer echten Prüfung.

### 3. Retention markiert, statt zu löschen

Der Job entfernt die **Datei** und setzt `proof_purged_at`; `proof_disk_name` wird auf `NULL`
gesetzt. Der Eintrag mit Datum, Art, Dauer und RPE bleibt dauerhaft bestehen.

Das ist der Unterschied zwischen „das Bild ist weg" und „hier war nie eines": Die UI kann
„Nachweis gelöscht (Retention)" anzeigen statt eines defekten Bildes, und die Jahres-Historie eines
Spielers bleibt vollständig. Der Serve-Endpoint antwortet auf einen gelöschten Nachweis mit
**HTTP 410 Gone** statt 404 — der Unterschied ist für den Client bedeutungstragend.

Frist 90 Tage nach Saisonende, gleichgezogen mit `video-retention`. Bewusst **ohne** T-7-Vorwarnung:
bei Videos ist die Warnung sinnvoll (unwiederbringlicher, großer Inhalt), bei einem
Nachweis-Screenshot wäre ein Push nur Lärm — das Original liegt ohnehin auf dem Handy des Spielers.
Stattdessen ein statischer Hinweistext im UI. Das spart zugleich die aufwendigere Hälfte des
Video-Jobs (`sendVideoRetentionWarnings` samt `notification_log`-Idempotenz).

Der Scheduler ist Foundation und darf laut Architektur-Test kein Domain-Package importieren →
Inline-SQL und nachgebauter Pfad, exakt wie bei `deleteRetainedVideos`.

### 4. Kompression bleibt clientseitig — enger parametrisiert

Der bestehende Helfer `web/src/lib/imageCompress.ts` wird unverändert wiederverwendet, nur mit
anderen Optionen:

| | Chat / Mitteilungen | Trainingstagebuch |
|---|---|---|
| `targetBytes` | 1 MB | **150 KB** |
| `maxEdge` | 1920 px | **1280 px** |
| Formate | WebP → JPEG | WebP → JPEG (unverändert) |

WebP bleibt erste Wahl: bei gleichem Zielwert deutlich bessere Qualität als JPEG, Browser-Support
seit Safari 14 unkritisch. Das dient direkt dem Ziel „stärker komprimieren als im Chat".

**Verworfene Alternative — serverseitige Normalisierung.** Erwogen war, den Server zur Autorität zu
machen (Invariante „gespeichert wird immer JPEG ≤ 150 KB") und nicht dekodierbare Formate per
`ffmpeg` oder `libvips` zu wandeln. Das wäre die einzige Konstruktion, die eine echte *Garantie*
gibt: `imageCompress.ts` reicht in zwei Pfaden stillschweigend das Original durch — bei Dateien
unter der Zielgröße (Zeile 94) und bei gescheitertem Decode (Zeile 101), und `createImageBitmap`
scheitert bei HEIC auf Chrome und Firefox.

Dagegen sprach die Praxis: im Chat funktioniert der bestehende Weg. iOS wandelt Fotos beim
Datei-Picker praktisch immer nach JPEG, sodass HEIC den Server kaum je erreicht. Der Preis der
Server-Variante wäre hoch gewesen — eine neue Systemabhängigkeit auf dem VPS, ein
Konvertierungs-Semaphore gegen OOM auf der 1-GB-Kiste, ein Roh-Upload-Limit von 20 MB statt 1 MB
und die Abkehr von einer Konvention, die an vier anderen Stellen trägt. Das ist zu viel Maschinerie
für einen Randfall.

**Bewusst akzeptierte Restlücke:** Erreicht doch einmal ein HEIC den Server, greift die
MIME-Whitelist und antwortet mit HTTP 400 („Format nicht unterstützt"). Der Spieler sieht eine
klare Fehlermeldung statt eines stillen Fehlschlags — dasselbe Verhalten wie heute im Chat.

### 5. Ein einziges, payload-freies Broadcast-Event

Alle Mutationen senden `h.hub.Broadcast("training-diary-changed")`. Das erfüllt die Hard-Rule und
das Broadcast-Gate ohne Allowlist-Eintrag.

Die naheliegende Sorge — ein globaler Broadcast auf privaten Daten — greift nicht: das Event ist
ein nackter String ohne Nutzlast, und jeder Client lädt beim Reload ausschließlich das, was seine
ACL hergibt. Die Alternative (`BroadcastToUsers` an Eigentümer plus betroffene Trainer) verlangte
bei **jeder** Mutation eine Auflösung „welche Trainer gehören zu diesem Spieler" und brächte keinen
Sicherheitsgewinn. Der Preis sind ein paar überflüssige Refetches auf offenen Tagebuch-Seiten.

### 6. Artenkatalog als CHECK-Constraint, nicht als Config-Tabelle

`kind` ist auf sieben Werte festgelegt (`kraft`, `ausdauer`, `athletik`, `technik`,
`beweglichkeit`, `reha`, `sonstiges`); `kind_custom` trägt den Freitext und ist **genau dann**
gesetzt, wenn `kind = 'sonstiges'`.

Eine pflegbare Tabelle nach Muster `training_group_categories` wäre die generellere Lösung, kostet
aber eine Verwaltungsoberfläche, eine weitere Route und einen Rechte-Tier. Der Freitext-Zweig fängt
jede Lücke bereits ab, und die Hausregel für Wertelisten sind `CHECK`-Constraints. Eine achte Art
ist damit eine Migration — vertretbar, weil der Katalog nach Sportart stabil ist.

## Risks / Trade-offs

- **Selbstberichtete Daten sind nicht verifiziert.** Der Nachweis ist weich, also lässt sich das
  Tagebuch schönen. Das ist Absicht: Motivation und Sichtbarkeit sind das Ziel, nicht Kontrolle.
  Wer daraus Konsequenzen ableiten will (Aufstellung, Sanktionen), stützt sich auf wackligen Grund
  — gehört als Hinweis in die Trainer-Sicht.
- **Leerstand wirkt wie Faulheit.** Ein Spieler ohne Einträge ist in der Team-Übersicht nicht von
  einem Spieler zu unterscheiden, der trainiert und nur nichts einträgt. Die Kennzahl misst
  Erfassungsdisziplin, nicht Trainingsfleiß. Die Trainer-Sicht sollte das benennen statt zu
  suggerieren, sie messe die Realität.
- **1280 px kostet Lesbarkeit bei Screenshots.** Ein hochkantes Strava-Bild (1170×2532) landet bei
  ~592×1280; große Kennzahlen bleiben lesbar, Kleingedrucktes nicht. Bewusst gewählt zugunsten des
  Speicherbudgets — 1600 px / 200 KB wäre die entspanntere Variante, falls es in der Praxis stört.
- **Eltern sehen das Tagebuch.** Bei jüngeren Jahrgängen selbstverständlich (sie laden den Nachweis
  oft selbst hoch), bei einer A-Jugend womöglich unerwünscht. Bewusst gleichgezogen mit
  `attendance-statistics`, wo Eltern denselben Zugriff bereits haben — eine altersabhängige
  Abstufung wäre eine neue Rechte-Dimension und hier nicht gerechtfertigt.
- **Retention ist unwiederbringlich.** Nach 90 Tagen ist die Datei weg; nur ein älteres Backup holt
  sie zurück. Genau so gewollt — das ist der Zweck einer Retention-Policy.

## Open Questions

- Sollen `vorstand`/`vorstand_beisitzer` lesen dürfen? Aktuell **nein** (gleichgezogen mit
  `attendance.canSeeMemberStats`, das `vorstand` ebenfalls nicht führt). Falls der Vorstand die
  Übersicht braucht, ist das eine Zeile in der ACL — aber es widerspricht „privat".
- Braucht der Trainer eine Zeitraum-Auswahl feiner als „Saison" (z. B. „letzte 4 Wochen")? In der
  Sommerpause ist die Saisonsumme wenig aussagekräftig. Nicht in diesem Change; leicht nachrüstbar,
  da die Aggregation ohnehin über einen Datumsbereich läuft.
