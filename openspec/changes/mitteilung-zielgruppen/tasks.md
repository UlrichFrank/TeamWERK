> **Reihenfolge:** Dieser Change geht `mitteilung-lesebestaetigung` voraus. Die
> Lesebestätigung braucht eine vertrauenswürdige Empfängermenge als Nenner — solange eine
> Zielgruppe still auf null auflöst, wäre `47 / 183` falsch, ohne dass es auffiele.

## 1. Datenbank

- [x] 1.1 Migrationsnummer bestimmen: `ls internal/db/migrations/ | sort -V | tail -1`. Erwartet `049`, aber **prüfen** — zwischen Design und Umsetzung kann eine Nummer dazugekommen sein. Nie eine Nummer ≤ aktueller DB-Version vergeben (golang-migrate überspringt sie lautlos).
- [x] 1.2 `049_broadcast_zielgruppen.up.sql`: `broadcasts` nach dem Muster aus `028_chat_broadcast_media.up.sql` neu aufbauen — `target_type TEXT NOT NULL CHECK(target_type IN ('users','members','spieler','eltern','legacy'))`, **ohne** `target_id` und `target_role`, alles Übrige unverändert (inkl. des Body/Media-CHECKs).
- [x] 1.3 Im `INSERT … SELECT` die Bestandswerte abbilden: `CASE target_type WHEN 'all' THEN 'users' ELSE 'legacy' END`. `broadcast_reads` **nicht** anfassen — daran hängt die Zustellung.
- [x] 1.4 `049_*.down.sql`: Rückbau auf `CHECK(target_type IN ('all','team','role'))` mit wiederhergestellten `target_id`/`target_role` (NULL) und `CASE target_type WHEN 'users' THEN 'all' ELSE 'all' END` — die ursprüngliche Team-Zuordnung ist nicht rekonstruierbar, das Down ist bewusst verlustbehaftet und wird im Kommentar so benannt.
- [x] 1.5 Up/Down gegen eine **isolierte Kopie** verifizieren, nicht gegen die echte `teamwerk.db`. Bekannte Repo-Einschränkung: `make migrate-down` ist ein No-Op (`db.Migrate()` ruft nur `m.Up()`), das Down also direkt per `sqlite3 < …down.sql` prüfen.
- [x] 1.6 `internal/db/migrations_test.go` ergänzen: neuer CHECK vorhanden, `target_id`/`target_role` weg, eine `'all'`-Bestandszeile landet auf `'users'`, eine `'team'`-Zeile auf `'legacy'`, `broadcast_reads` unverändert in Zeilenzahl.

## 2. Backend: Zielgruppen-Auflösung

- [x] 2.1 `internal/chat/audiences.go` anlegen: Konstanten `TargetUsers`/`TargetMembers`/`TargetSpieler`/`TargetEltern`, `ValidTarget(string) bool` (lehnt `legacy` und alles Unbekannte ab) und `resolveAudience(ctx, db, target string) ([]int, error)`. Bewusst **kein** Import aus `internal/policy` — die Ordner-ACL beantwortet ein Prädikat pro Subjekt, nicht die Mengenfrage (design.md §1).
- [x] 2.2 Die vier Queries schreiben. `members`/`spieler` über `JOIN members m ON m.user_id = u.id` (schließt `user_id IS NULL` strukturell aus), `spieler` zusätzlich über `member_club_functions`, `eltern` als `SELECT DISTINCT parent_user_id FROM family_links`. Jede Query liefert distinkte User-IDs — bei `spieler` sonst eine Zeile je Vereinsfunktion.
- [x] 2.3 `resolveBroadcastRecipients` in `handler.go` durch den Aufruf ersetzen; die alten `case "all"/"team"/"role"` und der `user_accessible_teams`-Zugriff entfallen.
- [x] 2.4 Tests `internal/chat/audiences_test.go`: je Zielgruppe die **exakte** erwartete ID-Menge (nicht nur „nicht leer"), Elternteil ohne eigene Funktion nicht in `spieler`, Elternteil zweier Kinder genau einmal in `eltern`, Mitglied ohne `user_id` in keiner Menge, Mitglied mit zwei Vereinsfunktionen genau einmal in `members`.

## 3. Backend: SendBroadcast

- [x] 3.1 Guard umstellen auf `admin | vorstand | sportliche_leitung`; der Trainer-Sonderpfad (`handler.go:1187-1206`, `kader_trainers`-Query) entfällt vollständig.
- [x] 3.2 Request-Struct: `TargetID`/`TargetRole` entfernen, `TargetType` gegen `ValidTarget` prüfen → 400 bei jedem anderen Wert (inkl. `all`, `team`, `role`, `legacy`, leer).
- [x] 3.3 `INSERT INTO broadcasts` ohne `target_id`/`target_role`.
- [x] 3.4 Antwort auf `201 {"id": <id>, "recipients": <n>}` umstellen — `n` zählt die tatsächlich benachrichtigten Empfänger **ohne** den Absender (dieselbe Menge, die SSE + Push bekommt).
- [x] 3.5 Route-Tests `internal/chat/broadcast_target_test.go`: Happy-Path je Zielgruppe mit Assertion auf `recipients` **und** auf die erzeugten `broadcast_reads`-Zeilen; 403 für Trainer/Spieler/Kassierer/Eltern; 201 für reine sportliche Leitung (heute 403); 400 für `all`/`team`/`role`/`legacy`/fehlend; Absender bekommt Zeile mit `read_at`, aber weder Push noch Zählung.
- [x] 3.6 Bestehende Broadcast-Tests auf das neue Vokabular ziehen: `media_message_test.go:221,246` und `push_fanout_test.go:93` senden `"all"` → `"users"`. `unread_test.go:52` schreibt direkt `target_type='all'` → `'users'`.

## 4. Backend: Capabilities

- [x] 4.1 `internal/policy/rules.go`: `CanBroadcast` auf `p.Role == "admin" || p.hasAnyFunction("vorstand", "sportliche_leitung")` ändern (`trainer` raus).
- [x] 4.2 `CapBroadcastAll` und `CanBroadcastAll` ersatzlos entfernen, inkl. des `if CanBroadcastAll(p)`-Blocks in `Capabilities`.
- [x] 4.3 `internal/policy/rules_test.go`: die vier `broadcast_all`-Assertionen (Z. 268, 281, 299, 304) ersetzen — Trainer hat **kein** `broadcast_messages` mehr, sportliche Leitung hat es, Vorstand hat es.
- [x] 4.4 Grep-Test in `internal/arch/`: der String `broadcast_all` kommt in `internal/` und `web/src/` nicht mehr vor. Fängt die Restfehler-Klasse ab, bei der eine Frontend-Abfrage auf eine nie gelieferte Capability zurückbleibt und einen Button stumm dauerhaft versteckt (design.md §7).

## 5. Frontend: Composer

- [x] 5.1 `web/src/pages/ChatPage.tsx`, `BroadcastComposer`: `targetType`-State auf `"users" | "members" | "spieler" | "eltern"` (Default `"spieler"`), `targetId`- und `targetRole`-State samt der beiden abhängigen `<select>`-Blöcke und dem `/teams`-Fetch entfernen.
- [x] 5.2 Ein Dropdown mit vier festen Optionen: Alle Nutzer · Alle Mitglieder · Alle Spieler · Alle Eltern. Kein Capability-abhängiges Ausblenden mehr — wer den Composer öffnen darf, darf alle vier.
- [x] 5.3 Prop `isAdmin` (transportierte in Wahrheit `broadcast_all`, also *vorstand-like*) und der Aufruf `isAdmin={hasCapability("broadcast_all")}` (Z. 1848) entfallen.
- [x] 5.4 Nach erfolgreichem Senden `recipients` aus der Antwort anzeigen: „An N Empfänger gesendet", bei `N === 0` deutlich als Hinweis (`Alert Info`-Klassenstring), nicht als Fehler.
- [x] 5.5 `web/src/test/renderAsPersona.tsx`: `broadcast_all` aus Z. 52 entfernen; `broadcast_messages` in Z. 49 auf vorstand-like + `sportliche_leitung` umstellen (`trainer` raus).
- [x] 5.6 `web/src/pages/__tests__/ChatPage.permissions.test.tsx`: Test „Trainer ohne broadcast_all sieht Team-Auswahl direkt nach Modal-Öffnung" (Z. 56) ersetzen durch: Trainer sieht den Button „Mitteilung senden" **nicht**; sportliche Leitung sieht ihn und findet alle vier Optionen.
- [x] 5.7 Vitest für den Composer: Auswahl je Zielgruppe schickt den erwarteten `targetType` und **kein** `targetId`/`targetRole`; Erfolgsmeldung zeigt die zurückgegebene Empfängerzahl; 0 Empfänger erzeugt einen sichtbaren Hinweis.

## 6. Verifikation und Übergabe

- [x] 6.1 `openspec validate mitteilung-zielgruppen --strict`.
- [x] 6.2 `/verify-change` — Build/Test/Lint plus Projekt-Invarianten (Route→Tests, Mutation→`Broadcast`, brand-Tokens, lucide-Icons, Migrationsnummer).
- [ ] 6.3 Auf Prod prüfen, wie viel Bestand betroffen ist: `SELECT target_type, COUNT(*) FROM broadcasts GROUP BY target_type;`. Zeilen mit `target_type='role'` sind nie zugestellt worden — falls vorhanden, die Absender informieren, dass diese Mitteilungen niemanden erreicht haben.
- [ ] 6.4 **Trainer vorab informieren.** Das Mitteilungsrecht verschwindet für sie mit dem Deploy. Ersatz benennen: die Team-Standardgruppe „Spieler" bzw. „Eltern" im Chat erreicht denselben Kreis, zusätzlich mit Rückkanal. Ohne diese Ankündigung ist der Button eines Tages einfach weg (design.md §2).
- [ ] 6.5 Ein Commit pro Task-Gruppe, Conventional Commits mit Scope `chat` (Migration mit Scope `db`, Capability-Änderung mit Scope `auth`).
