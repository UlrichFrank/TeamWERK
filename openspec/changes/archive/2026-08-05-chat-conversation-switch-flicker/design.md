# Design — Konversationswechsel ohne Fremdinhalt

## Messung: wie der Befund entstanden ist

Zwei Instrumente gegen die Prod-Binary (`bin/teamwerk-e2e`, Seed-DB, Port 18081), weil
jsdom/Vitest diese Klasse strukturell nicht sieht und Chrome sie maskiert:

1. **rAF-Sampling** — pro Frame `scrollTop`/`scrollHeight`/`clientHeight` des
   Scroll-Containers `[data-windowed-scroll]`, dazu Anzahl `img` / `complete` /
   `aria-busy`. Sieht den Pre-Paint-Zustand, also Scroll-Mathematik.
2. **Video (25 fps) + Frame-Differenz** (`ffmpeg tblend=difference,signalstats`) — sieht,
   was **tatsächlich gemalt** wird. Nötig, weil rAF-Sampling einen Paint zwischen zwei
   Ticks nicht beweist.

Für einen realistischen Fall wurde eine **reine Text-Konversation mit 130 Nachrichten** in
die Seed-DB gelegt: alle vorhandenen langen Seed-Threads sind bildlastig, die gemeldeten
Chats sind Direktnachrichten.

### Ergebnis 1 — der Mischzustand ist real und reproduzierbar

WebKit, Wechsel zurück zu einer bereits gelesenen Konversation, Frame 218/291:
Header „E2E Chat lang gelesen", Liste zeigt „E2E Nachricht 1–8" aus der vorherigen
Konversation. Reproduziert in **3 von 3** WebKit-Aufnahmen (Frames 217–219, 218–220,
219–221), also jeweils ~120 ms. Die Chromium-Aufnahme zeigte den Zustand nicht — der erste
neue Frame war bereits vollständig korrekt.

Derselbe WebKit-Lauf mit der **Text-Konversation** (kleinerer, schnellerer Fetch) zeigte den
Mischzustand ebenfalls nicht: eine einzige saubere Änderung. Das ist der Beleg, dass es sich
um ein **Rennen** handelt und nicht um einen Engine-Unterschied im Rendering.

### Ergebnis 2 — der Scroll-Anker ist entlastet

| Phase | `scrollTop` | `scrollHeight` | Deutung |
|---|---|---|---|
| Container leer | 0 | 492 | vor dem Commit |
| Nachrichten + Bild-Platzhalter | 6388 | 6848 | bereits unten — **kein** Frame bei `scrollTop 0` |
| nach Bild-Decode | 12688 | 13148 | `Δtop = +6300`, `Δheight = +6300` |

Der Anker folgt dem Höhenwachstum exakt; der untere Rand bleibt stehen. Damit sind drei
naheliegende Hypothesen widerlegt:

- **kein** unkompensiertes `scrollHeight`-Wachstum (Blink-Scroll-Anchoring-These),
- **kein** Flattern durch mehrfache `scrollTop`-Writes,
- **keine** verspätet aufgelöste `clientHeight` durch die Flex-Verschachtelung.

Nebenbefund ohne Handlungsbedarf: `clientHeight` fällt beim Wechsel 492 → 460, weil der
Knopf „Ältere Nachrichten laden" über der Liste erscheint.

## Entscheidung 1: Liste beim Wechsel leeren

`openConversation` setzt `setMessages([])` vor dem Fetch. Damit ist der Mischzustand
strukturell unmöglich, unabhängig von Netzwerklaufzeit, Engine und Gerät.

**Verworfene Alternativen:**

| Alternative | Warum verworfen |
|---|---|
| `activeConv` erst nach dem Fetch setzen | Der Header bliebe bis zu einer Sekunde beim alten Chat stehen — der Klick fühlt sich tot an. Verschiebt das Problem, statt es zu lösen. |
| Rendering des Panes an `activeConv.id === messagesConvId` koppeln | Gleiche Wirkung, aber ein zusätzlicher Ableitungs-State, der mit jedem SSE-Pfad (`appendNewMessages`, `loadOlderMessages`) synchron gehalten werden muss. Mehr Fläche für Folgefehler. |
| Nachrichten pro Konversation cachen und sofort anzeigen | Beseitigt das Flackern und wäre schneller — bringt aber Invalidierung (Edit/Delete/Reaktion/SSE) und Speicher pro Konversation mit. Eigenständiges Feature, nicht Teil eines Bugfixes. |
| Nur `useLayoutEffect` (ohne Leeren) | Wirkt erst, wenn die neuen Nachrichten da sind — das Fenster davor bleibt unverändert. Löst das gemessene Problem **nicht**. |

## Entscheidung 2: Ladezustand statt leerer Fläche

Eine leere Fläche ist von „Konversation ohne Nachrichten" nicht unterscheidbar. Ein
`loadingMessages`-State rendert stattdessen einen Hinweis mit `role="status"` (zugleich der
Anker für die Testassertion). `loadMessages` setzt ihn im `finally` zurück, damit ein
Fehlschlag nicht in einem Dauer-Ladezustand endet — der bestehende `catch {}`-Block
verschluckt Fehler sonst still.

## Entscheidung 3: Anker im `useLayoutEffect`

Der `[messages]`-Effekt (`ChatPage.tsx:705`) läuft heute als `useEffect`, also **nach** dem
Paint. Nach dem Leeren der Liste bliebe damit ein Zwei-Schritt: erst der leere/positionslose
Zustand mit neuem Inhalt, dann der Anker-Sprung. `useLayoutEffect` legt beides in denselben
Frame.

Die Messung zeigt, dass dieser Frame in der Praxis meist nicht sichtbar wurde (React flusht
passive Effekte oft vor dem Paint) — die Umstellung ist deshalb **Absicherung, nicht
Hauptursache**. Sie kostet nichts: der Effekt macht ausschließlich DOM-Lesen/-Schreiben ohne
State-Updates, genau der Anwendungsfall für einen Layout-Effekt.

**Risiko:** `useLayoutEffect` blockiert den Paint. Der Rumpf ruft `applyAnchor` bzw.
`smoothScrollToBottom` — beides ist O(1) an Scroll-Arithmetik plus ein
`scrollIntoView`. Kein Layout-Thrashing über die Liste. Der Watcher-Effekt (Zeile 751)
bleibt bewusst ein `useEffect`: er hängt nur Listener an und darf den Paint nicht blockieren.

## Entscheidung 4: Testbarkeit durch künstliche Verzögerung

Der Bug ist ein Rennen und damit nicht verlässlich zu treffen — weder per Frame-Diff noch
per Wiederholung. Er wird **deterministisch**, sobald die Antwort künstlich verzögert wird:

- **Vitest:** der `api.get`-Mock für die Zielkonversation liefert ein Promise, das der Test
  selbst auflöst. Zwischen Klick und Auflösung ist der Mischzustand stabil beobachtbar.
- **Playwright:** `page.route` verzögert `GET /api/chat/conversations/*/messages` um 800 ms.

Damit braucht der Regressionstest **kein WebKit und keine Video-Analyse** — er läuft in der
bestehenden Chromium-Suite und schlägt ohne den Fix zuverlässig fehl. Das ist der Grund,
warum das WebKit-Projekt in diesem Change entfallen kann.

Die Video-/rAF-Instrumente waren für die **Diagnose** nötig, nicht für die Absicherung. Sie
werden nicht ins Repo übernommen.

## Entscheidung 5: Anker-Guard für die Leerphase

*(Nachträglich aufgenommen. Der Konflikt wurde erst bei der Implementierung von Entscheidung 1
sichtbar und ist im Code verifiziert — nicht vermutet.)*

Das Leeren der Liste committet eine leere Nachrichtenliste. Das löst den `[messages]`-Effekt
aus → `applyAnchor()` → `scheduleAnchorSettle()` armiert einen 600-ms-Timer. Die leere Box
enthält weder `aria-busy` noch `<img>`, also liefert `anchorMediaPending()` `false`, und
`check()` ruft `releaseAnchor()` — **obwohl der Fetch noch läuft**. Der Anker ist dann weg,
bevor die Nachrichten überhaupt da sind.

Beim normalen Wechsel A→B ist das entschärft: `activeConv.id` ändert sich, die Cleanup-Funktion
des Watcher-Effekts läuft und räumt `anchorSettleTimerRef` ab. **Nicht** entschärft sind zwei
reale Fälle:

- **Erstes Öffnen** nach dem Laden der Seite — der Watcher-Effekt war zuvor per
  `if (!activeConv) return` (`ChatPage.tsx:795`) ausgestiegen und hatte gar kein Cleanup
  registriert.
- **Erneutes Öffnen der bereits aktiven Konversation** — `activeConv?.id` ändert sich nicht,
  also läuft kein Cleanup.

In beiden Fällen gilt: Fetch > 600 ms ⇒ Anker verloren. Für `anchorRef = "bottom"` fängt der
MutationObserver das per Sticky wieder ans Ende. Für `"divider"`/`"divider-chip"` geht die
Öffnungsposition verloren — der Nutzer landet am Ende statt am UnreadDivider, also genau die
Garantie, die `chat-open-at-unread` zusichert. Lokal (~120 ms) und in den E2E-Tests tritt das
nicht auf; produktiv über Mobilfunk mit 100 Nachrichten ist > 600 ms plausibel.

**Entscheidung:** ein `awaitingMessagesRef`, gesetzt vor dem Leeren, zurückgesetzt sobald die
Nachrichten committet sind (auch im Fehlerfall). `applyAnchor` steigt bei gesetztem Ref sofort
aus — ohne Scroll und ohne Settle-Arming. Der Guard sitzt bewusst **in `applyAnchor`** und
nicht am Aufrufer: so deckt er neben dem `[messages]`-Effekt auch den Pfad über `reposition()`
im Watcher ab (der MutationObserver feuert auf das Entfernen aller Zeilen).

`anchorRef` selbst bleibt gesetzt und überlebt die Leerphase — die Anker-Semantik ändert sich
also nicht, nur der Zeitpunkt der ersten Anwendung.

**Verworfene Alternativen:**

| Alternative | Warum verworfen |
|---|---|
| Guard nur im `[messages]`-Effekt | Deckt den `reposition()`-Pfad des MutationObservers nicht ab; der armiert den Settle-Timer ebenso. |
| Settle-Timeout großzügiger bemessen | Verschiebt die Grenze, statt sie zu beseitigen — bei langsamer Verbindung weiterhin verlierbar. Und es greift in die durch die Messung entlastete Anker-Logik ein, mit Risiko für die drei bestehenden E2E-Scroll-Tests. |
| `messages.length` direkt in `applyAnchor` prüfen | `applyAnchor` ist ein `useCallback`; eine Abhängigkeit von `messages` ließe es bei jeder Nachrichtenänderung neu entstehen und damit den Watcher-Effekt (Deps enthalten `applyAnchor`) neu aufsetzen — MutationObserver und Listener würden ständig ab- und wieder angehängt. Ein Ref hat diese Kopplung nicht. |
| Befund nur dokumentieren, Fix vertagen | Wäre eine bewusst eingebaute Regression gegen eine zugesicherte Capability-Garantie. Der Guard ist additiv und klein. |

## Warum jsdom das diesmal sieht

Anders als der ursprüngliche `chat-scroll-anchor`-Bug (Scroll-Position nach Bild-Decode —
Layout-Physik, für jsdom unsichtbar) ist dies eine reine **DOM-Inhalts**-Frage: welche
Nachrichten stehen zu welchem Zeitpunkt im Baum. Das kann Vitest vollständig prüfen. Der
Playwright-Test ergänzt nur die echte Navigation über die Konversationsliste.
