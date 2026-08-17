# Tasks — Nachrichten-Badge auf dem ganzen Pfad

Rein Frontend: keine Migration, keine Route, kein Backend-Diff. Die Reihenfolge ist
aufsteigend nach Risiko — der gemeinsame Helfer zuerst, der Dashboard-Umbau zuletzt.

## 1. Gemeinsamer Zähl-Helfer

- [x] 1.1 `web/src/lib/chatUnread.ts`: reine Funktion `chatUnreadCounts(conversations, broadcasts)` → `{ conversations, broadcasts, total }`. `conversations` = Σ `unreadCount`, `broadcasts` = Anzahl `!isRead && !isSent`, `total` = Summe. Kein React, kein Netzwerk (unit-testbar ohne Render, analog `createThrottledProgress` in `VideoUploadPage.tsx`). Typen möglichst schmal halten (nur die gelesenen Felder), damit alle drei Aufrufstellen ihre eigenen Interfaces behalten können.
- [x] 1.2 `web/src/lib/chatUnread.test.ts` (kolokiert statt in `__tests__/` — dominante Konvention in `web/src/lib/`): getrennte Anteile, eigene Mitteilung (`isSent: true`) zählt nicht, leere Listen → alles 0.

## 2. Tab „Chats" auf `/chat`

- [x] 2.1 `web/src/pages/ChatPage.tsx`: `totalUnread` (Z. 1177) auf `chatUnreadCounts` umstellen; die beiden Anteile lokal verfügbar machen.
- [x] 2.2 Badge am Tab „Chats" (Z. 1204-1210) ergänzen — **nur** der Konversations-Anteil, nicht `total` (sonst zählen die Mitteilungen doppelt, s. `design.md` Decision 5). Klassen-String 1:1 vom Mitteilungen-Badge (Z. 1218) übernehmen: `bg-brand-yellow text-brand-black text-xs font-bold rounded-full px-1.5`.
- [x] 2.3 Bei der Gelegenheit den doppelt ausgewerteten Ausdruck am Mitteilungen-Badge (Z. 1217 + 1219 berechnen dasselbe `filter(...).length`) durch den Wert aus 2.1 ersetzen.
- [x] 2.4 `web/src/pages/__tests__/ChatPage.unreadTabs.test.tsx`: Chats-Tab zeigt nur Konversations-Unread; Summe beider Tab-Badges = Zahl an der `<h1>`; ohne Unread kein Badge (kein `0`).

## 3. Navigation: `navBadges`-Map + Modul-Header

- [x] 3.1 `web/src/components/AppShell.tsx`: `loadChatUnread` (Z. 145-156) auf `chatUnreadCounts` umstellen — Verhalten unverändert, nur die Formel wandert in den Helfer.
- [x] 3.2 `const navBadges: Record<string, number> = { '/chat': chatUnread }` im Render-Scope anlegen.
- [x] 3.3 Den Sonderfall am Nav-Item (Z. 310, `item.to === '/chat' && chatUnread > 0`) auf `navBadges[item.to] ?? 0` umstellen. Der Badge selbst (Klassen, Position im `justify-between`-Flex) bleibt unverändert.
- [x] 3.4 Modul-Header (Z. 289-298) um den Badge ergänzen: `visibleItems.reduce((s, i) => s + (navBadges[i.to] ?? 0), 0)`, nur rendern wenn `> 0`. **Über `visibleItems`, nicht über `mod.items`** — sonst zeigt der Header eine Zahl für einen Eintrag, den der Nutzer gar nicht sehen darf. Der Header ist ein `<button>` mit `justify-between` und trägt bereits das Chevron rechts; Badge davor einsetzen.
- [x] 3.5 Die Modul-Summen aus 3.4 so berechnen, dass sie **außerhalb** der `navModules.map()`-Renderfunktion verfügbar sind (z. B. als abgeleitete Liste `{ label, badge }[]`, über die anschließend gerendert wird). Task 3.6 braucht sie; eine zweite, parallel gerechnete Variante wäre genau die Doppelung, die Decision 4 vermeiden soll.
- [x] 3.6 Hinweis-Punkt am Menü-Button (`AppShell.tsx:388-395`, nur der `sm:hidden`-Header): Bedingung „irgendeine Modul-Summe > 0" aus 3.5 — **nicht** `Object.values(navBadges).some(...)`, das umgeht die `navRoutes`-Filterung. Punkt in `brand-danger`, ohne Zahl; Button braucht dafür `relative`, der Punkt `absolute` oben rechts. `aria-label` dynamisch: bei Unread `Menü öffnen, {n} ungelesene Nachrichten`, sonst unverändert `Menü öffnen`. Punkt in `brand-yellow` mit `ring-1 ring-brand-black/30` (der Ring grenzt ihn gegen den weißen Header ab; `brand-danger` wäre die falsche Bedeutungsebene).
- [x] 3.7 `web/src/components/__tests__/AppShell.navBadges.test.tsx`: Badge am eingeklappten Modul „Verein" sichtbar (obwohl `/chat` nicht gerendert ist); Item- und Modul-Badge gleichzeitig bei offenem Modul; kein Badge wenn `/chat` nicht in `navRoutes`; kein `0`-Badge. Für den Hamburger: Punkt bei geschlossener Sidebar sichtbar, `aria-label` nennt die Zahl, ohne Unread kein Punkt und `aria-label` fällt zurück, kein Punkt ohne `/chat` in `navRoutes`. Bestehende `AppShell.permissions.test.tsx` als Vorlage für das Mocking von `navRoutes`/`useAuth`, `useMediaQuery` für den Mobil-Fall mocken.

## 4. Dashboard: Fetch hochziehen + Header-Badge

Der einzige nicht-triviale Teil. Reihenfolge einhalten — 4.1 vor 4.2, sonst ist die Section
kurzzeitig ohne Datenquelle.

- [x] 4.1 `web/src/pages/DashboardPage.tsx`: `load` + `useChatEvents` aus `MeineNachrichtenSection` (Z. 483-520) nach `DashboardPage` hochziehen. Dort `convs`/`broadcasts` halten, daraus `rows` (Filter + Sort + `slice(0, 5)`, unverändert) und `unreadTotal = chatUnreadCounts(convs, broadcasts).total` ableiten. Grund: `Accordion` rendert `{isOpen && children}` (Z. 151) — die Section **mountet nicht**, solange sie eingeklappt ist, ein Callback-Prop feuert also nie (`design.md` Decision 6).
- [x] 4.2 `MeineNachrichtenSection` presentational machen: `rows: NachrichtRow[]` als Prop, kein eigener `useState`/`useEffect`/`useChatEvents` mehr. Leerzustand und Fußzeilen-Link bleiben in der Section.
- [x] 4.3 `Accordion` (Z. 129-158) um `badge?: number` erweitern: rendern nur bei `> 0`, im Header-`<span>` neben dem Titel, vor dem Chevron. Optionaler Prop — die anderen fünf Accordions bleiben unverändert.
- [x] 4.4 `<Accordion id="nachrichten" … badge={unreadTotal}>` (Z. 666) setzen. **Nicht `rows.length`** — eine Zeile kann mehrere ungelesene Nachrichten bündeln und die Liste ist auf 5 gedeckelt (`design.md` Decision 7).
- [x] 4.5 `DashboardPage.nachrichten.test.tsx` erweitern: Badge am **eingeklappten** Header (Killer-Case — deckt Mount-Abhängigkeit und `slice(0,5)` in einem Fall ab); `7 Konversationen à 2 ungelesen → 14, nicht 5`; `chat:new-message` erhöht den Badge bei eingeklappter Section. Die bestehenden Szenarien (Liste, Leerzustand, Links) müssen nach dem Prop-Umbau unverändert grün bleiben.

## 5. Verifikation

- [x] 5.1 `pnpm -C web test` und `pnpm -C web lint` grün.
- [x] 5.2 Sichtprüfung im Browser (Chrome DevTools MCP gegen die Prod-Binary auf :18080, Seed-DB), mobil (390×844) und Desktop, jeweils mit ungelesener Nachricht: (a) Dashboard eingeklappt → Badge am Section-Header; (b) mobil, Sidebar **geschlossen** → Punkt am Hamburger sichtbar; Sidebar öffnen → Badge am Modul „Verein"; (c) `/chat` → beide Tab-Badges tragen ihren Anteil, die `<h1>` bleibt zahlenfrei; (d) Konversation lesen → Modul-, Item-, Hamburger- und Dashboard-Badge fallen live, ohne Reload (223 → 220 → 180 verifiziert).
- [x] 5.3 Dabei gefunden und behoben: die Dashboard-Zeile verlinkte auf das nackte `/chat` statt auf `/chat?conv=<id>` — der Klick öffnete die Konversation nicht, nichts wurde gelesen, alle Badges blieben stehen. Regressionstest in `DashboardPage.nachrichten.test.tsx`. Offen für den Menschen: Gegenprobe mit einem Nutzer **ohne** `/chat` in `navRoutes` (Seed-Admin sieht alles) und Kontrast des gelben Punkts auf dem weißen Header am echten Gerät.
- [x] 5.4 `/verify-change` (brand-Tokens statt Raw-Tailwind, keine Unicode-Icons, `openspec validate`).
- [x] 5.5 Ein Commit pro Task-Gruppe, Conventional Commits mit Scope `chat` (bzw. `pwa` für den AppShell-Teil).
