# Nachrichten-Badge auf dem ganzen Pfad

## Why

Die Ungelesen-Zahl existiert im gesamten UI an **genau zwei** Stellen — und beide liegen am
tiefsten Punkt einer zweistufigen Einklapp-Hierarchie, in der keine einzige Zwischenebene
etwas anzeigt.

```
   MOBIL, App gerade geöffnet (Landing = Dashboard)
   ═════════════════════════════════════════════════

   ☰  TeamWERK                      ← Gate 1: Hamburger, kein Indikator
   │
   │  (Sidebar ist Overlay — komplett unsichtbar)
   ▼
   NUTZER        ▸
   SPIELBETRIEB  ▸                  ← Gate 2: Akkordeon, genau EIN Modul
   VEREIN        ▸                    offen, folgt zwangsweise der Route,
   VERWALTUNG    ▸                    kein Indikator am Header
        │
        │ nur wenn aufgeklappt
        ▼
     Nachrichten ⟨3⟩                ← einzige Fundstelle (AppShell.tsx:310)

   Dashboard-Inhalt darunter:
     ▸ Meine Termine                  (offen — Default 'termine')
     ▸ Nachrichten     ▸            ← Gate 3: eingeklappt, kein Indikator
     ▸ Ereignisse      ▸              (DashboardPage.tsx:598, 666)
```

Drei Gates hintereinander, alle drei stumm. Auf Mobil, direkt nach dem Öffnen, ist die Zahl
sichtbarer Hinweise auf eine neue Nachricht **null**.

**Es gibt keinen Workaround.** `AppShell.tsx:252-259` klappt bei *jeder* Navigation
zwangsweise das Modul der aktiven Route auf. Das Modul „Verein" lässt sich also nicht
offenhalten, während man im Kalender arbeitet — die Route schließt es wieder zu. Auf einer
Verein-Route zu stehen ist die **einzige** Bedingung, unter der der Badge je sichtbar wird.

|  | Dashboard | andere Seite |
|---|---|---|
| **Desktop** | sichtbar (Sections default offen) | blind, außer auf Verein-Route |
| **Mobil** | blind | blind |

### Das Signal ist in Ordnung — nur das Surfacing nicht

Die naheliegende Lesart wäre „Benachrichtigung geht verloren". Die trifft hier nicht zu:

```
   SIGNAL                          SURFACING
   ══════                          ═════════

   unreadCount (DB, exakt)  ────▶  1 Badge, 3 Gates tief
   SSE chat:new-message     ────▶  löst nur reload() aus
   Push /chat?conv=42       ────▶  funktioniert — nur außerhalb der App
   setAppBadge()            ────▶  nur Homescreen-Icon, nicht in der App

   ▲ alles korrekt                 ▲ hier ist das Loch
```

Anders als bei den Meldungen, die den Event-Log nötig machten (`user_events`, sechs
verlustbehaftete Stellen im Push-Pfad), ist `unreadCount` ein **exakter, unbegrenzt
haltbarer** Zustand. Er braucht keinen neuen Träger — er braucht Sichtbarkeit. Diese
Änderung ist deshalb rein Frontend: keine Migration, keine Route, kein Backend-Diff.

## What Changes

Der bestehende Chat-Badge wird **zusätzlich** an vier Stellen gezeigt. Bewusst nicht Teil
dieses Changes: Auto-Navigation beim App-Start und In-App-Toast bei Ankunft (siehe
`design.md`, Verworfene Alternativen).

### 1. „Chats"-Tab auf `/chat` — Symmetrie-Lücke

```
   ChatPage.tsx:1203
   ┌──────────────────────┬──────────────────────┐
   │  💬  Chats           │  📢  Mitteilungen ⟨2⟩│
   └──────────────────────┴──────────────────────┘
           ▲                          ▲
      kein Badge              Zeile 1217, existiert bereits
```

Der Chats-Tab bekommt den **Konversations-Anteil** (nicht `totalUnread` — sonst zählen die
Mitteilungen doppelt, einmal im eigenen Tab und einmal nebenan).

### 2. Modul-Header „Verein" im Menü — Sonderfall wird Mechanik

Heute steht in `AppShell.tsx:310` ein hartkodierter Sonderfall (`item.to === '/chat'`). Statt
ihn ein zweites Mal für den Modul-Header hinzuschreiben, wird er einmal aufgelöst:

```
   const navBadges = { '/chat': chatUnread }          ← eine Zeile

   Modul-Header:  visibleItems.reduce((s,i) => s + (navBadges[i.to] ?? 0), 0)
   Item:          navBadges[item.to] ?? 0             ← ersetzt Zeile 310
```

Netto keine Zeile mehr als ein Copy-Paste, aber ein Sonderfall **weniger** als heute. Zwei
Eigenschaften fallen gratis ab: die Summe läuft über `visibleItems`, wer `/chat` serverseitig
nicht in `navRoutes` hat, bekommt also auch am Header nichts; und der Header rendert
unabhängig vom Aufklapp-Zustand.

### 3. Hamburger-Punkt auf Mobil — der äußerste Gate

Auf Mobil ist die Sidebar ein Overlay (`AppShell.tsx:374-384`); sie existiert visuell nicht,
bis der Hamburger angetippt wird. Der Modul-Badge aus (2) hilft dort erst **nach** einer
Interaktion, die man nur macht, wenn man ohnehin schon etwas vermutet.

```
   ┌────────────────────────────────────┐
   │  ☰ ●    ‹ Zurück    TeamWERK       │   ← AppShell.tsx:389
   └────────────────────────────────────┘
```

**Punkt, keine Zahl.** Der Hamburger steht für das *gesamte* Menü, nicht für Nachrichten. Eine
Zahl dort behauptet mehr, als sie weiß: sobald ein zweiter Eintrag in `navBadges` landet,
müsste sie entweder über semantisch Unverwandtes summieren oder still falsch werden. Ein
Punkt („da drin ist etwas") skaliert mit der Map, ohne eine Aussage zu treffen, die er nicht
belegen kann.

**Farbe ist nicht `brand-yellow`.** Der mobile Header ist `bg-brand-white` — der Gelbton der
übrigen Badges funktioniert nur auf dem `bg-brand-gray` der Sidebar und wäre hier praktisch
unsichtbar. Der Punkt nutzt `brand-danger`.

**Das `aria-label` wächst mit.** Ein rein visueller Punkt trägt für Screenreader keine
Information; `"Menü öffnen"` wird bei Ungelesenem zu `"Menü öffnen, 3 ungelesene Nachrichten"`.

Nur Mobil. Auf Desktop ist die Sidebar dauerhaft sichtbar (`hidden sm:flex`), der
Modul-Header genügt.

### 4. Dashboard-Akkordeon „Nachrichten" — hier liegt der Haken

```tsx
// DashboardPage.tsx:151
{isOpen && (
  <div id={`section-${id}`}>{children}</div>
)}
```

`MeineNachrichtenSection` **mountet nicht**, solange das Akkordeon zu ist. Kein Mount → kein
`load()` → keine Daten. Der Zähler kann unmöglich aus der Section nach oben gemeldet werden;
eine Callback-Prop ist by construction kaputt. Derselbe Schnitt wie in der Sidebar, eine
Etage tiefer.

Dazu eine zweite Falle: `rows.length` ist **nicht** die Ungelesen-Zahl.

```
   .filter(unreadCount > 0)   ← 1 Konversation mit 3 ungelesenen = 1 row
   .slice(0, 5)               ← bei 7 Quellen bleiben 5

   7 Konversationen à 2 ungelesen  →  rows.length = 5,  echt = 14
```

Beides zusammen: der Fetch wandert aus der Section nach `DashboardPage`. Die Section wird
presentational (`rows` als Prop), `DashboardPage` hält `load` + `useChatEvents` + die echte
Summe und reicht sie an einen neuen `badge?: number`-Prop von `Accordion`.

```
   ┌─ DashboardPage ──────────────────────────────┐
   │  load() ─┬─ unreadTotal ──▶ <Accordion badge> │  ← immer gerendert
   │          └─ rows(5) ──────▶ <Section rows>    │  ← nur wenn offen
   └──────────────────────────────────────────────┘
```

Nebeneffekt: `useChatEvents` öffnet **pro Aufrufstelle eine eigene EventSource**. Heute laufen
auf dem Dashboard zwei parallel (AppShell + Section); nach dem Hochziehen bleiben es zwei
statt drei.

### 5. Gemeinsamer Zähl-Helfer

Die Summenformel steht heute zweimal im Code (`AppShell.tsx:152-154`,
`ChatPage.tsx:1177-1179`) und würde mit dem Dashboard auf drei wachsen. Die subtile Hälfte
ist `!isRead && !isSent` — **eigene** Mitteilungen zählen nicht mit; das driftet leicht
auseinander. Neu: `web/src/lib/chatUnread.ts` mit einer reinen Funktion, die beide Anteile
und die Summe liefert.

### Anzeige-Regel: verschachtelt, immer sichtbar

Der Badge bleibt sichtbar, auch wenn das Kind-Element sichtbar ist. Das ist keine
Meinungsfrage, sondern bereits Hausstil: `/chat` zeigt heute die Gesamtzahl an der `<h1>`,
den Anteil am Tab und die Einzelzahl an der Konversationszeile — drei Ebenen gleichzeitig.

| Ort | Zählt |
|---|---|
| Hamburger `<Menu>` (nur Mobil) | Punkt ohne Zahl — „irgendein Badge im Menü ist > 0" |
| Modul-Header „Verein" | Summe (Konversationen + Mitteilungen) |
| Nav-Item „Nachrichten" | Summe — unverändert zu heute |
| Tab „Chats" | nur Konversationen |
| Tab „Mitteilungen" | nur Mitteilungen — unverändert zu heute |
| Dashboard-Header „Nachrichten" | Summe |

## Impact

- **Betroffene Capabilities:** `chat-unread-badge-pfad` (neu), `dashboard-nachrichten`
  (modifiziert — Section lädt nicht mehr selbst)
- **Betroffener Code:** `web/src/lib/chatUnread.ts` (neu), `web/src/components/AppShell.tsx`,
  `web/src/pages/ChatPage.tsx`, `web/src/pages/DashboardPage.tsx`
- **Nicht betroffen:** Backend, DB, Routen, Service Worker, App-Icon-Badge
  (`chat-unread-app-badge` bleibt unverändert)
- **Risiko:** gering. Der einzige nicht-triviale Teil ist der Fetch-Umzug im Dashboard; er
  ist mechanisch und durch den bestehenden Test `DashboardPage.nachrichten.test.tsx`
  abgesichert.

## Test-Anforderungen

Frontend-only, keine neuen Routen — die Ebene ist Vitest + Testing Library. Testheimat
existiert überall.

| Datei | Test | Garantierte Invariante |
|---|---|---|
| `web/src/lib/__tests__/chatUnread.test.ts` | `zaehlt Konversationen und Mitteilungen getrennt` | `conversations` + `broadcasts` = `total` |
| ″ | `eigene Mitteilung zaehlt nicht` | `isSent: true` bleibt außen vor, auch bei `isRead: false` |
| `web/src/pages/__tests__/ChatPage.unreadTabs.test.tsx` | `Chats-Tab zeigt nur Konversations-Unread` | Tab-Badge ≠ `totalUnread`, keine Doppelzählung der Mitteilungen |
| ″ | `Chats-Tab ohne Unread zeigt keinen Badge` | kein `⟨0⟩` |
| `web/src/components/__tests__/AppShell.navBadges.test.tsx` | `Modul-Header Verein zeigt Summe bei eingeklapptem Modul` | Badge ist sichtbar, **ohne** dass `/chat` gerendert ist |
| ″ | `ohne /chat in navRoutes kein Badge am Modul-Header` | Sichtbarkeits-Filter wirkt auch auf den Header |
| ″ | `Item- und Modul-Badge gleichzeitig sichtbar bei offenem Modul` | verschachtelte Anzeige, kein Verstecken |
| ″ | `Hamburger traegt Punkt bei Unread` | Punkt sichtbar, **ohne** dass die Sidebar geöffnet ist |
| ″ | `Hamburger aria-label nennt die Zahl` | Screenreader bekommt die Information, die der Punkt visuell trägt |
| ″ | `kein Punkt am Hamburger ohne Unread` | `aria-label` fällt auf „Menü öffnen" zurück |
| `web/src/pages/__tests__/DashboardPage.nachrichten.test.tsx` | `Badge am eingeklappten Nachrichten-Header` | **Killer-Case** — Daten werden geladen, obwohl die Section nicht mountet |
| ″ | `Badge zaehlt Nachrichten, nicht Zeilen` | 7 Konversationen à 2 ungelesen → `14`, nicht `5` |
| ″ | `Live-Update aktualisiert den Badge bei eingeklappter Section` | `chat:new-message` wirkt ohne Mount der Section |

Der Test `Badge am eingeklappten Nachrichten-Header` mit `unreadCount: 3` fängt beide
Dashboard-Fallen (Mount-Abhängigkeit + `slice(0,5)`) in einem Fall ab.
