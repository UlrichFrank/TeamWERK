# Design — spielbericht-typo3-publisher

## Zweck des Dokuments

Fixiert die Kern-Entscheidungen des Publisher-Ausbaus mit Begründungen
und listet die bewusst offen bleibenden Punkte. Kontrakt zwischen
TeamWERK und der TYPO3-Extension steht in
`../team-stuttgart-org/openspec/changes/spike-match-report-import/specs/match-report-import/spec.md`.

## Kern-Entscheidungen

### D-1 · `presseteam` als hierarchische `users.role`

`users.role` wird von `admin | standard` auf `admin | standard |
presseteam` erweitert. Semantik: `admin ⊇ presseteam ⊇ standard`.

- **Warum keine Vereinsfunktion:** Eltern schreiben Berichte, sind aber
  keine Members (`family_links`-Pfad, keine `member_club_functions`-Einträge
  möglich).
- **Warum keine vierte orthogonale Achse:** Nutzer-Feedback in Explore —
  eine dritte Rolle im `users.role`-Enum ist der explizite Wunsch, nicht
  eine separate Whitelist-Struktur.
- **Konsequenz Guard:** `auth.RequireRole("presseteam","admin")` (admin
  fällt hierarchisch mit rein). Kein separater `presseteam`-Guard-Helper —
  die aufzählenden Rollen sind Standardmuster im Repo.

### D-2 · Fire-and-forget Publish, kein Rollback aus TeamWERK

Nach erfolgreichem POST an TYPO3 (`202/200`) ist der Bericht in TeamWERK
`published` und **read-only**. Änderungen und Löschungen laufen direkt in
TYPO3 durch die Vorstands-Redaktion.

- **Warum:** matcht die Realität — nach Publish ist die öffentliche
  Homepage die Quelle der Wahrheit, TeamWERK ist der Autoren-Workflow-Tool.
  Umgekehrter Weg (TeamWERK überschreibt TYPO3) bräuchte Idempotenz
  (`external_report_id` auf `pages` — im Spike bewusst nicht gebaut) und
  ein komplexeres State-Modell.
- **Konsequenz Kontrakt:** TeamWERK sendet nie ein Update. Auch nicht
  „update if exists". Doppel-Publish wird durch State-Machine im
  TeamWERK-Server verhindert (siehe D-4), nicht auf Typo3-Seite.

### D-3 · Ergebnis strukturiert erfassen, für TYPO3 zusammensetzen

TYPO3 hat `match_score VARCHAR(50)` — Freitext. TeamWERK hält
`home_goals`, `away_goals`, optional `home_goals_ht`, `away_goals_ht`
(alles nullable INTEGER, Turnier-Flag = alles NULL erlaubt). Beim Publish
setzt der Publisher `match_score = "24:22 (12:9)"` zusammen.

- **Warum:** Tippfehler-Vermeidung, spätere Auswertung ohne
  String-Parsing, `match_score`-Format bleibt kanonisch.
- **Konsequenz Frontend:** Zwei Zahlenfelder statt eines Strings; bei
  Turnier-Häkchen ausgeblendet.

### D-4 · State-Machine mit expliziter `publishing`-Zwischenstufe

```
   draft ─publish─▶ publishing ─(2xx)─▶ published
                        │
                        └─(4xx/5xx)─▶ publish_failed
                                          │
                                          └─retry (nur manuell)─▶ publishing
```

- **Warum `publishing`?** Verhindert Doppel-POST bei ungeduldigem
  Klick oder Netz-Retry. State-Wechsel `draft→publishing` per
  `UPDATE … WHERE state='draft' RETURNING id` (SQLite: prüfen ob 1 Row
  geändert). Zweiter Klick sieht `state='publishing'` und bekommt 409.
- **Warum kein Auto-Retry?** Publisher kann viele Fehlerursachen haben
  (falscher Team-Kategorie-ID, Duplikat, Netzausfall). Mensch soll
  entscheiden — nicht Cron.

### D-5 · Bilder auf VPS zwischenspeichern, nach `published` löschen

Draft-Bilder liegen unter `./storage/match-report-images/{report_id}/`.
Nach `published` synchron im gleichen Request Dateien + DB-Zeilen der
Bilder-Referenzen löschen (Reihenfolge: erst DB, dann Filesystem).

- **Warum synchron?** Vermeidet Zombie-Bilder bei Server-Neustart
  zwischen State-Wechsel und Cleanup. Kosten: 10 × Datei-Löschen im
  Publish-Request — irrelevant.
- **Bei `publish_failed`:** Bilder bleiben liegen für den manuellen
  Retry. Cleanup nur, wenn User den Draft explizit löscht (was ohne
  Slot-Rückgabe nicht direkt möglich sein soll — separater Delete-Weg
  löscht auch den Slot-Bezug).

### D-6 · Duty-Slot als Autoren-Anker

Wer den Slot „Spielbericht" für ein Spiel zieht, ist der einzige
Autor für diesen Bericht. Kein Second-Author, kein Übergabe-Mechanismus.

- **Warum:** Nutzt die bestehende Deadline/Reminder/Konto-Infrastruktur
  1:1. Kein neues Zuweisungs-Modell.
- **Konsequenz:** Bei Slot-Rückgabe wird der Draft nicht gelöscht,
  sondern nur wieder autorenfrei — nächster Zieher übernimmt. Bei
  `published`-Bericht ist der Slot ohnehin erledigt.
- **Presseteam-Filter im Slot-Ziehen:** `duty-board` muss beim Anzeigen
  filtern (nur Presseteam sieht den Slot) und beim Ziehen prüfen
  (Backend-Check auf `role IN ('presseteam','admin')`).

### D-7 · Publisher-Auth per Bearer, Konfig in `.env`

Token in `.env` (`TYPO3_IMPORT_TOKEN`), Endpoint-URL ebenso
(`TYPO3_IMPORT_URL`). Kein Rotationsmechanismus in Version 1 — Rotation
= Deploy mit neuer `.env` + neuer TYPO3-`additional.php` in kurzer
Abfolge.

- **Warum simpel:** matcht dem Spike-Ansatz. Rotation ist selten (~1×/Jahr).
- **Konsequenz Deploy-Runbook:** Reihenfolge dokumentieren — TYPO3 muss
  neuen Token akzeptieren bevor TeamWERK ihn sendet (überlappender Zeitraum
  mit beiden Tokens gibt's nicht — kurzes Ausfall-Fenster in Kauf).

## Season-Segment im Publisher (Detail-Regel)

Contract-Version nach AC-8: TeamWERK sendet zwei getrennte Felder — den
title-Slug (`spike-test-tws-ma-vs-vfl-kirchheim`) und das Season-Segment
(`2026-2027`). Die TYPO3-Extension baut daraus den vollständigen Pfad
`/spielberichte/{season}/{slug}` selbst und **legt den Season-Ordner
(doktype=1) an**, falls er noch nicht existiert. Damit entfällt die
frühere `TYPO3_SEASON_FOLDER_PID`-Konfiguration ersatzlos.

Regel für Season-Segment:

1. Über `game_id → games.season_id → seasons.start_date/end_date` das
   Saison-Range holen und als `{start.year}-{end.year}` formatieren.
2. Wenn kein Season-Match (defensive Sicherung), Fallback auf die
   Sommer-zu-Sommer-Heuristik: Spieldatum `>= Juli` →
   `{year}-{year+1}`, sonst `{year-1}-{year}`. Nur Warnung loggen, nicht
   abbrechen.

Fällt der TYPO3-Endpoint zurück (z. B. weil die Season-Ordner-Anlage in
der Extension einmal versagt), bekommt der Publisher HTTP 4xx/5xx und
setzt state=`publish_failed` mit dem Extension-Error im Detail — der
Autor kann manuell erneut publishen.

## Bewusst offene / offene-Punkt-Liste

Aus dem Spike-Abschluss und dieser Diskussion:

1. **AC-8 (Mittwald-Roundtrip) auf Typo3-Seite noch offen.** Vor
   Publisher-Live-Gang muss der Nachbar-Spike auf Mittwald grün laufen.
   Der TeamWERK-Ausbau kann in Entwicklung beginnen (gegen DDEV), aber
   Deployment ist blockiert.
2. **AC-5-Rendering-Gap:** `MatchReport.html` im TYPO3-Template rendert
   das `media`-Feld aktuell nicht. Ist Template-Bug, nicht Publisher-Bug —
   Nachbar-Repo muss die Fluid-Section ergänzen (`<f:for each="{media}">…</f:for>`).
   Publisher bleibt davon unberührt (schickt korrekte Referenzen).
3. **Kein Idempotenz-Feld auf TYPO3-Seite.** Doppel-Publish würde zwei
   pages-Zeilen anlegen. Schutz: State-Machine im TeamWERK (D-4). Falls
   die im Extremfall versagt (paralleler DB-Zugriff mit Race), müsste
   TYPO3 als Härtung `external_report_id` bekommen — als eigener
   Follow-up-Change im Nachbar-Repo notiert.
4. **HTML-Sanitizing:** Publisher liefert Markdown-generiertes HTML,
   das durch allowlist-basierten Renderer läuft. Konkrete Allowlist:
   `<p>`, `<h2>`, `<h3>`, `<strong>`, `<em>`, `<ul>`, `<ol>`, `<li>`,
   `<a href>`, `<br>`. Alles andere strippen. Keine `<img>` inline —
   Bilder sind separate FAL-Referenzen.
5. **Season-Pfad Härtung:** Regel steht oben, aber untypische Fälle
   (Spiel über Jahreswechsel, verspätet nachgetragene Saison) müssen im
   ersten Betriebsmonat beobachtet werden. Fallback greift, aber
   Logging muss klar sein.

## Fallback-Wege

Nicht geplant. Der Spike zeigt: der Weg funktioniert. Falls in
Produktion `POST /api/match-report/import` reproduzierbar Fehler wirft,
die kein einfacher Bugfix im Extension-Code sind, ist der Fallback der
CLI-Command auf Extension-Seite (Nachbar-Change), nicht ein
Architekturwechsel in TeamWERK.
