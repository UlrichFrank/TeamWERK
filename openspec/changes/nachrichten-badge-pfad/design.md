# Design — Nachrichten-Badge auf dem ganzen Pfad

## Kontext

Rein Frontend. Kein neuer Endpunkt, keine Migration, kein Backend-Diff. Alle drei Badges
speisen sich aus den zwei bereits abgerufenen Listen `GET /api/chat/conversations` und
`GET /api/chat/broadcasts`.

## Decision 1 — Keine Auto-Navigation beim App-Start

**Verworfen:** beim Öffnen der App automatisch nach `/chat` springen, wenn Ungelesenes
existiert.

Drei Gründe:

1. **Welche Nachricht?** Bei zwei Konversationen plus einer Mitteilung gibt es keine
   richtige Antwort. Ein Sprung nach `/chat` ohne `?conv=` ist kein Sprung „zur Nachricht",
   sondern nur eine erzwungene Seitenauswahl.
2. **Wiederholung.** Wer eine Nachricht bewusst liegen lässt, würde bei *jedem* App-Start
   entführt. Es gibt keinen Zustand „gesehen, aber nicht gelesen", an dem das abschaltbar
   wäre — `message_reads` kennt nur gelesen/ungelesen.
3. **Existiert bereits, an der richtigen Stelle.** `/chat?conv=42` ist ein funktionierender
   Deep-Link (`ChatPage.tsx:705`) und wird vom Push-Pfad genutzt
   (`internal/chat/handler.go:891`). Auto-Navigation würde diesen Mechanismus an einer
   zweiten, schlechteren Stelle nachbauen.

**Konsequenz:** Der Nutzer behält die Navigationshoheit. Die Änderung macht den Zustand
sichtbar, nicht zwingend.

## Decision 2 — Verschachtelte Anzeige statt „nur wenn eingeklappt"

Der Modul-Header behält seinen Badge auch dann, wenn das Modul offen ist und das Nav-Item
seinen eigenen Badge zeigt. Gleiches auf dem Dashboard.

Die Alternative (Badge am Container nur zeigen, solange das Kind unsichtbar ist) ist in
anderen Produkten verbreitet, hätte hier aber zwei Nachteile: sie führt einen
Zustandsübergang ein, der beim Aufklappen wie ein Verschwinden der Meldung aussieht, und sie
widerspricht dem, was die App an anderer Stelle bereits tut.

`/chat` zeigt heute drei Ebenen gleichzeitig:

```
   <h1> Nachrichten ⟨5⟩          ← Gesamtsumme        ChatPage.tsx:1189
   ├── Tab „Chats"      ⟨3⟩      ← Anteil             (dieser Change)
   ├── Tab „Mitteilungen" ⟨2⟩    ← Anteil             ChatPage.tsx:1217
   └── Konversationszeile ⟨3⟩    ← Einzelwert         ChatPage.tsx:1269
```

Das ist Hausstil. Die neuen Badges folgen ihm.

## Decision 3 — `navBadges`-Map statt zweiter Sonderfall

`AppShell.tsx:310` prüft heute `item.to === '/chat'`. Der Modul-Header bräuchte dieselbe
Prüfung eine Ebene höher — mit dem Unterschied, dass er über *mehrere* Items summieren muss.

Gewählt:

```ts
const navBadges: Record<string, number> = { '/chat': chatUnread }
```

- **Modul-Header:** `visibleItems.reduce((s, i) => s + (navBadges[i.to] ?? 0), 0)`
- **Item:** `navBadges[item.to] ?? 0` — ersetzt den bestehenden Sonderfall

Der Diff ist netto nicht größer als ein Copy-Paste, entfernt aber einen Sonderfall statt
einen zweiten hinzuzufügen. Zwei Eigenschaften ergeben sich daraus, ohne dass sie eigens
programmiert werden:

- **Sichtbarkeit gratis.** Die Summe läuft über `visibleItems`, das bereits gegen
  `navRoutes` gefiltert ist. Wer `/chat` serverseitig nicht sieht, bekommt auch am
  Modul-Header nichts — kein zweiter Check nötig.
- **Render-Unabhängigkeit.** Das eingeklappte Modul rendert seine Items gar nicht
  (`{isOpen && visibleItems.map(...)}`). Die Summe wird aus der Map berechnet, nicht aus dem
  Render-Baum, und ist deshalb im eingeklappten Zustand korrekt.

**Bewusst keine Verallgemeinerung darüber hinaus.** Die Map hat einen Eintrag. Kandidaten für
weitere (ungesehene Ereignisse, offene Spielberichte, offene Dienste) sind erkennbar, aber
nicht Teil dieses Changes — die Map ist die Naht, nicht die Ausbaustufe.

## Decision 4 — Hamburger trägt einen Punkt, keine Zahl

Auf Mobil ist die Sidebar ein Overlay (`AppShell.tsx:374-384`) und damit vor dem Antippen
unsichtbar. Der Modul-Header aus Decision 3 sitzt *innerhalb* dieses Overlays — er löst den
zweiten Gate, nicht den ersten. Der Hamburger (`AppShell.tsx:389`) ist die einzige Fläche,
die dauerhaft sichtbar ist.

Drei Festlegungen, alle drei gegen die naheliegende Variante:

**Punkt statt Zahl.** Der Hamburger repräsentiert das *gesamte* Menü. Eine `3` dort ist heute
zufällig richtig, weil `navBadges` einen Eintrag hat. Mit einem zweiten Eintrag (offene
Spielberichte, ungesehene Ereignisse) gäbe es nur schlechte Optionen: über semantisch
Unverwandtes summieren, oder einen Wert anzeigen, der stillschweigend nur noch eine Teilmenge
abbildet. Ein Punkt sagt „im Menü ist etwas" — eine Aussage, die auch mit fünf Einträgen noch
stimmt.

```
   ●  =  irgendein Modul-Header-Badge > 0
```

Wichtig ist dabei, **worüber** summiert wird. Die naheliegende Formulierung
`Object.values(navBadges).some(n => n > 0)` wäre falsch: sie umgeht die
`navRoutes`-Filterung, die den Modul-Headern ihre Sichtbarkeitsprüfung gratis liefert
(Decision 3). Ein Nutzer ohne Chat-Zugriff bekäme einen Punkt für etwas, das er im
geöffneten Menü nirgends findet. Der Punkt hängt deshalb an denselben, bereits gefilterten
Modul-Summen — kein zweiter Zustand, keine zweite Filterlogik.

**`brand-yellow` mit Ring, nicht `brand-danger`.** Der erste Entwurf nahm `brand-danger`, weil
alle anderen Badges auf `bg-brand-gray` (Sidebar) bzw. `bg-brand-surface-card` (Dashboard)
sitzen, der mobile Header aber `bg-brand-white` ist und `#FDE400` dort als 2.5-Einheiten-Punkt
zerfließt. Das löst zwar das Sichtbarkeitsproblem, verschiebt aber die Bedeutungsebene: Rot ist
im übrigen System das Fehler-/Gefahren-Signal (`brand-danger` in Alerts, destruktiven Buttons),
und eine ungelesene Nachricht ist beides nicht. Der Punkt bleibt deshalb gelb wie alle anderen
Badges und bekommt einen dünnen `ring-1 ring-brand-black/30`, der ihn gegen Weiß abgrenzt —
farblich konsistent, ohne die Sichtbarkeit zu opfern.

**`aria-label` wächst mit.** Ein rein visueller Punkt ist für Screenreader nicht existent. Das
statische `aria-label="Menü öffnen"` wird bei `total > 0` zu
`"Menü öffnen, 3 ungelesene Nachrichten"`. Bewusst mit der konkreten Zahl, obwohl visuell nur
ein Punkt steht: die Einschränkung auf einen Punkt ist eine Platz- und
Semantik-Entscheidung für das Auge, kein Grund, der assistiven Ausgabe Information
vorzuenthalten.

**Nur Mobil.** Auf Desktop ist die Sidebar dauerhaft sichtbar (`hidden sm:flex`), der Punkt
wäre dort redundant zum Modul-Header.

## Decision 5 — Chats-Tab zählt nur Konversationen

Naheliegend wäre `totalUnread` (steht in `ChatPage.tsx:1177` fertig da). Falsch: die
Mitteilungen erschienen dann zweimal, einmal im eigenen Tab und einmal nebenan, und die
Summe der beiden Tabs überstiege die Zahl an der `<h1>`.

```
   <h1> ⟨5⟩  =  Chats ⟨3⟩  +  Mitteilungen ⟨2⟩      ✓ gewählt
   <h1> ⟨5⟩  ≠  Chats ⟨5⟩  +  Mitteilungen ⟨2⟩      ✗
```

Die Tab-Badges partitionieren die Gesamtsumme. Der Modul-Header im Menü und der
Dashboard-Header zeigen dagegen die Summe — sie stehen für „Nachrichten" als Ganzes, nicht
für eine Hälfte.

## Decision 6 — Fetch wandert nach `DashboardPage`

Das ist der einzige nicht-triviale Teil des Changes.

`Accordion` rendert seine Kinder konditional:

```tsx
// DashboardPage.tsx:151
{isOpen && (
  <div id={`section-${id}`}>{children}</div>
)}
```

React *erzeugt* das Element `<MeineNachrichtenSection />` zwar, **mountet** es aber nicht,
solange `isOpen` false ist. Ohne Mount läuft der `useEffect` mit `load()` nie. Damit sind
zwei naheliegende Lösungen ausgeschlossen:

| Ansatz | Warum nicht |
|---|---|
| Callback-Prop `onUnread={setBadge}` | Section mountet nicht → Callback feuert nie |
| `rows.length` als Zähler nach oben | Section mountet nicht; *und* der Wert wäre falsch (s.u.) |

Gewählt: `load` + `useChatEvents` wandern nach `DashboardPage`. Die Section wird
presentational und bekommt `rows` als Prop.

```
   ┌─ DashboardPage ──────────────────────────────────────┐
   │                                                       │
   │  load()  ──▶ convs, broadcasts                        │
   │               │                                       │
   │               ├─ chatUnreadCounts(...).total          │
   │               │        └──▶ <Accordion badge={…}>     │  immer gerendert
   │               │                                       │
   │               └─ rows = [...].sort().slice(0,5)       │
   │                        └──▶ <MeineNachrichtenSection  │  nur wenn offen
   │                               rows={rows} />          │
   └───────────────────────────────────────────────────────┘
```

**Nebeneffekt (erwünscht):** `useChatEvents` öffnet pro Aufrufstelle eine eigene
`EventSource` (`hooks/useChatEvents.ts:12`). Heute laufen auf dem Dashboard zwei parallel
(AppShell + Section). Nach dem Umzug bleiben es zwei statt drei — der Hook wandert mit, er
wird nicht zusätzlich aufgerufen.

**Bewusst nicht gemacht:** AppShell und DashboardPage auf *eine* gemeinsame Quelle
zusammenlegen. Das würde einen geteilten Context oder einen Cache erfordern; der Gewinn wäre
ein gesparter Doppel-Fetch auf genau einer Route. Nicht das Problem dieses Changes.

## Decision 7 — `rows.length` ist nicht der Zähler

Zwei unabhängige Verfälschungen in derselben Pipeline (`DashboardPage.tsx:492-512`):

```
   convs
     .filter(c => c.unreadCount > 0)     ← 1 Konversation mit 3 ungelesenen = 1 row
     ...
   [...convRows, ...bcRows]
     .sort(...)
     .slice(0, 5)                        ← bei 7 Quellen bleiben 5

   Beispiel: 7 Konversationen à 2 ungelesen
             rows.length = 5     echt = 14
```

Der Badge nutzt deshalb `chatUnreadCounts(convs, broadcasts).total` auf den **ungefilterten**
Listen. `rows` bleibt für die Anzeige bei fünf Einträgen gedeckelt — die Deckelung ist
Absicht der Capability `dashboard-nachrichten` und bleibt unangetastet.

## Decision 8 — Gemeinsamer Zähl-Helfer

Die Summenformel steht heute zweimal:

| Stelle | Code |
|---|---|
| `AppShell.tsx:152-154` | `convUnread + bcUnread` |
| `ChatPage.tsx:1177-1179` | `totalUnread` |

Mit dem Dashboard würde sie auf drei wachsen. Die driftgefährdete Hälfte ist die
Broadcast-Bedingung `!b.isRead && !b.isSent` — dass **eigene** Mitteilungen nicht mitzählen,
ist keine Selbstverständlichkeit, die ein dritter Aufschreiber zuverlässig reproduziert.

Neu: `web/src/lib/chatUnread.ts`

```ts
export function chatUnreadCounts(convs, broadcasts): {
  conversations: number   // Σ unreadCount
  broadcasts: number      // Anzahl !isRead && !isSent
  total: number           // Summe beider
}
```

Reine Funktion ohne Netzwerk und ohne React — unit-testbar ohne Component-Render, analog zu
`createThrottledProgress` in `VideoUploadPage.tsx`. Alle drei Aufrufstellen konsumieren sie;
`ChatPage` behält seine lokalen Listen und ruft nur die Funktion.

## Verworfene Alternativen (Gesamtbild)

Aus der Vorüberlegung stammen drei weitere Schichten. Sie sind hier bewusst **nicht** drin,
damit dieser Change klein und prüfbar bleibt:

| Idee | Status | Kommentar |
|---|---|---|
| In-App-Toast bei `chat:new-message` | offen | Adressiert den *Ankunfts*-Fall, den Badges nur passiv abdecken. Muster existiert (`maintenanceToast`, `AppShell.tsx:484`). Braucht Coalescing bei lebhaften Gruppenchats. |
| Dashboard-Section auf Mobil initial öffnen bei `unread > 0` | offen | Weichere Variante der Auto-Navigation. Nach diesem Change weniger dringend, weil der Header-Badge dann sichtbar ist. |
| Auto-Navigation nach `/chat` beim App-Start | verworfen | Decision 1. |
| Zahl statt Punkt am Hamburger | verworfen | Decision 4 — skaliert nicht über einen zweiten `navBadges`-Eintrag hinaus. |

## Kleiner Nebenbefund (nicht in diesem Change)

`DashboardPage.tsx:539` verlinkt Konversations-Zeilen auf `/chat`, obwohl `row.id` direkt
daneben verfügbar ist und `/chat?conv=<id>` funktioniert. Der Push-Pfad springt damit
präziser in die App als das eigene Dashboard. Eigenständiger Ein-Zeilen-Fix, gehört nicht in
den Badge-Change.
