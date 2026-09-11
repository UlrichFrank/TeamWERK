## Context

`internal/chat/handler.go` hat bereits zwei getrennte Listen-Endpunkte: `ListMessages` (Cursor-Pagination `after`/`before`, `messagePageSize = 100`, liefert nur `preview`/`truncated` statt Volltext) und `ListBroadcasts` (ungepaginiert, max. 100, liefert vollen `body`). Sichtbarkeit läuft für Konversationen über `conversation_members` (Suche nutzt `isActiveMember`-Semantik, s.u.), für Mitteilungen über eine `broadcast_reads`-Zeile pro Empfänger (`hidden_at IS NULL` = sichtbar). Wichtiger Fund beim Erkunden: `DeleteMessage` setzt nur `deleted_at`, der `body` bleibt in der DB unverändert — die Maskierung passiert erst in der SELECT-Projektion von `ListMessages`. Eine Suche, die direkt gegen `messages.body` matcht, muss `deleted_at IS NOT NULL` deshalb **explizit** ausschließen, sonst tritt eine Nachricht, die als gelöscht gilt, über die Suchergebnisse wieder zutage.

Es gibt aktuell **keinen** "Jump-to-Message"-Mechanismus im Frontend (auch nicht beim Klick auf ein Reply-Zitat — das ist nur ein statisches Label). Das ist der riskanteste Teil dieses Changes: `docs/agent/06-gotchas.md` dokumentiert bereits einen echten Scroll-Bug (`chat-open-at-unread`) in genau diesem Screen — Vitest/jsdom sah ihn nicht, nur echtes Chrome-Layout. Der Such-Jump führt einen neuen Scroll-Einstiegspunkt ein und muss deshalb bewusst **neben** der bestehenden Anchor-Logik (`applyAnchor`/`releaseAnchor`, Unread-Divider) laufen, nicht in sie verschmolzen werden.

## Goals / Non-Goals

**Goals:**
- Volltextsuche über `messages.body` (aktive Konversationen) und `broadcasts.body` (sichtbare Mitteilungen) in einem Request.
- Sprung zu einem Chat-Treffer zeigt die Nachricht ohne manuelles Scrollen.
- Bestehende Scroll-/Anchor-Logik beim normalen Öffnen einer Konversation bleibt unverändert.

**Non-Goals:**
- Keine Volltext-Indizierung (FTS5, Trigram) — die Datenmenge eines einzelnen Vereins-Chats (Handball-Abteilung, keine Tausenden Mitglieder) rechtfertigt den Aufwand nicht; `LIKE '%term%'` auf einer Handvoll Tausend Zeilen ist auf der VPS unproblematisch.
- Kein "weiter nach unten laden" (Cursor `after`) über das initiale Around-Fenster hinaus in der Such-Trefferansicht — wer weiterlesen will, scrollt (bestehendes `before` funktioniert weiter nach oben) oder schließt die Suche und öffnet die Konversation regulär.
- Keine Suche über Bild-Beschreibungen/Dateinamen, keine Umfragen-Fragen/-Optionen als eigener Treffertyp (eine Umfrage ist eine `messages`-Zeile — ihr `body` trägt die Frage und wird mitdurchsucht, das reicht für den Use-Case).
- Keine serverseitige Umlaut-/Akzent-Normalisierung — `LIKE` faltet in SQLite nur ASCII-Groß-/Kleinschreibung (gleiche bekannte Einschränkung wie die bestehende Mitgliedersuche in `internal/members/handler.go`), wird nicht neu gelöst.

## Decisions

**1. Endpoint-Form folgt der bestehenden Paginierungskonvention.** `GET /api/chat/search?q=&limit=&offset=0` → `{ items: [...], total: N }`, analog zu `GET /api/members`/`GET /api/users` (`docs/agent/04-api-db.md`, Abschnitt Paginierung). `q` ist Pflicht (HTTP 400 bei leer), `limit` default 50/max 200, `offset` default 0 — identische Grenzen wie `internal/members/handler.go`.

**2. Zwei UNION-artige Teilqueries statt einer gemeinsamen Tabelle.** `messages` und `broadcasts` haben unterschiedliche Sichtbarkeits-Joins (Konversationsmitgliedschaft vs. `broadcast_reads`) und unterschiedliche Spalten. Der Handler fragt beide getrennt ab (je mit `LIKE ? LIMIT/OFFSET`-Fenster auf einer kombinierten, nach `sent_at` sortierten Menge — einfachste Umsetzung: beide Teilmengen laden, in Go mergen/sortieren/slicen, da die Gesamttrefferzahl pro Suche in diesem Kontext klein bleibt), statt einen SQL-`UNION` mit tabellenübergreifenden Spalten-Aliasen zu bauen. Das hält beide Zweige lesbar und testbar, kostet aber eine zusätzliche Sortierung in Go statt in SQL — akzeptabel bei der erwarteten Ergebnisgröße (kein Fall, in dem `total` in die Tausende geht).

**3. Sichtbarkeitsscope = aktive Mitgliedschaft, nicht `isMember`.** `ListMessages` erlaubt Lesen auch für Nutzer, die eine Konversation verlassen haben (`isMember`, ohne `left_at`-Filter) — das ist dort bewusst so, weil ein Deep-Link (`?conv=`) auf eine bereits geöffnete Konversation zeigen kann. Für die Suche ist das nicht wünschenswert: Ergebnisse aus einer verlassenen Konversation wären UI-technisch eine Sackgasse (die Konversation steht nicht mehr in `conversations` im Frontend-State, ein Klick liefe ins Leere). Die Suche filtert deshalb auf `conversation_members.left_at IS NULL` (wie `isActiveMember`), konsistent mit dem, was der Nutzer in der Sidebar sieht.

**4. `%`/`_` im Suchbegriff werden für `LIKE` escaped.** SQLite `LIKE` behandelt `%`/`_` als Wildcards; ein Suchbegriff mit Literal-`%` (kommt vor, z. B. "50% Rabatt") würde sonst überraschend viele Treffer liefern. Escapen via `ESCAPE '\'` und Ersetzen von `\`, `%`, `_` im Suchbegriff vor dem Binding (gleiche Technik wie SQL-Injection-frei über Parameter-Bindung, hier zusätzlich Wildcard-Bindung).

**5. Snippet statt Volltext im Suchergebnis.** Konsistent mit dem Preview/Volltext-Split in `ListMessages` (`messagePreviewLen = 280`) liefert ein Suchtreffer nur einen ca. 140 Zeichen breiten, rune-genau um die erste Fundstelle zentrierten Ausschnitt (mit `…`-Ellipsen an abgeschnittenen Enden), keinen vollen `body`. Der volle Text kommt beim Öffnen der Konversation/Mitteilung ohnehin mit.

**6. Jump-to-Message: neuer `around`-Cursor auf `ListMessages`, kein neuer Endpunkt.** `GET /api/chat/conversations/{id}/messages?around={messageId}` liefert bis zu `messagePageSize/2` Nachrichten mit `id <= around` (absteigend, dann umgedreht) plus bis zu `messagePageSize/2` mit `id > around` (aufsteigend) — zwei Queries im selben Muster wie die bestehenden `before`/`after`-Zweige, in Go zusammengeführt. Response bekommt zusätzlich `hasOlder`/`hasNewer`-Flags (aktuell implizit über Seitengröße erkennbar, hier explizit, weil die Trefferposition nicht am Rand liegt). `around` ist mit `after`/`before` mutually exclusive (wie bereits `after`/`before` es zueinander sind). Alternative verworfen: eigener `GET /api/chat/search/context/{messageId}`-Endpunkt — hätte Sichtbarkeits- und Scan-Logik dupliziert, die `ListMessages` schon hat.

**7. Frontend: eigener Such-Overlay-Zustand, kein Reuse der Anchor-Logik.** Klick auf einen Chat-Treffer setzt `activeConv` wie gewohnt, lädt aber über den neuen `around`-Cursor statt über den Default-Pfad (`loadMessages`/Unread-Anchor). Die getroffene Nachricht bekommt einen `data-message-id`-Marker und wird per `scrollIntoView({block: "center"})` + kurzzeitiger CSS-Highlight-Klasse (Pulse, 2s) hervorgehoben — ein isolierter, einmaliger Effekt, der nach dem ersten Rendern der Zielnachricht feuert und **nicht** mit `keepPosition`/`applyAnchor` interferiert, weil er nur beim Betreten über einen Suchtreffer aktiv ist (eigenes `searchJumpTarget`-Stück State, das nach Verbrauch genullt wird). Mitteilungs-Treffer nutzen unverändert `openBroadcast`.

**8. Header-Control-Button, kein globales Tastenkürzel.** Lupe (`Search`-Icon aus `lucide-react`, bereits importiert) als `HEADER_CTRL_ICON` neben der `<h1>Nachrichten</h1>` öffnet ein Modal (Card-Look wie bestehende Modals: `bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow`). Kein `Cmd/Ctrl+K`-Shortcut in dieser Iteration — kein bestehendes Shortcut-System in der App, würde Scope unnötig vergrößern.

## Risks / Trade-offs

- **[Risiko] Gelöschter `body` bleibt in der DB, ein Bug im `deleted_at`-Filter würde gelöschte Inhalte durchsuchbar machen.** → Mitigation: dedizierter Test (`TestSearch_GeloeschteNachrichtNichtGefunden`), der eine gelöschte Nachricht mit eindeutigem Suchbegriff anlegt und die Abwesenheit in den Ergebnissen prüft — nicht nur den Happy-Path.
- **[Risiko] `LIKE '%term%'` kann keinen Index nutzen — Full Scan über `messages`/`broadcasts` bei jeder Suche.** → Mitigation: akzeptiert (Non-Goal FTS5), aber `limit`/`offset`-Obergrenzen (analog Mitglieder-Suche) verhindern unbegrenzte Ergebnismengen; bei spürbarem Wachstum der Tabellen ist FTS5 ein sauberer Folge-Change, keine Korrektur dieses Designs.
- **[Risiko] Scroll-Jump interferiert mit bestehender Anchor-/Unread-Logik in `ChatPage.tsx` (dokumentierter Gotcha-Bug-Typ).** → Mitigation: eigener, isolierter State-Pfad (Entscheidung 7) statt Wiederverwendung; `make test-e2e`/Playwright-Fall für "Suchtreffer öffnen scrollt zur Nachricht" ergänzen (Kriterium aus `docs/agent/07-testing.md`: Scroll-Verhalten gehört auf die Playwright-Ebene, nicht Vitest).
- **[Trade-off] Zwei getrennte Teilqueries + Go-seitiges Merge statt SQL-`UNION`.** Mehr Code als ein einzelnes `UNION`-Statement, aber vermeidet fragile Spalten-Alias-Kopplung zwischen zwei strukturell verschiedenen Tabellen und bleibt einfacher zu testen.

## Migration Plan

Keine DB-Migration. Reiner additiver Code-Change (neue Route, neuer Query-Parameter auf bestehender Route, neue Frontend-Komponente). Rollback = Revert des Commits, keine Datenmigration rückabzuwickeln.
