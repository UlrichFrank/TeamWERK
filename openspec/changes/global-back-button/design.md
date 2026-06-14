## Context

AppShell ist der äußere Layout-Wrapper für alle eingeloggten Seiten. Es rendert Sidebar (Desktop), Mobile-Header und `<Outlet>` (Page-Content). Aktuell haben einzelne Seiten (`TermineDetailPage`, `MeinTeamPage`, `SpieltagDetailPage`, `MembersPage`) eigene Zurück-Buttons mit unterschiedlicher Platzierung und Logik (`navigate(-1)`, `<Link to="…">`, Text-Links).

React Router v6 speichert seinen internen History-Stack-Index in `window.history.state.idx`. Dieser Wert ist `0` beim ersten App-Aufruf und steigt mit jeder Navigation — er ist die zuverlässige Quelle für „Gibt es etwas, wohin zurücknavigiert werden kann?".

## Goals / Non-Goals

**Goals:**
- Ein einziger `← Zurück`-Button, global in AppShell, sichtbar auf allen Seiten wenn History vorhanden
- Konsistentes Verhalten: immer `navigate(-1)`, kein State-Tracking, kein Label-Engineering
- Mobile: Button im Mobile-Header zwischen Hamburger und Titel
- Desktop: Button als schmale Leiste über dem Page-Content
- Bestehende lokale Zurück-Elemente konsolidieren (entfernen)

**Non-Goals:**
- Dynamisches Label (z.B. „Zurück zum Dashboard") — zu viel Komplexität für zu wenig Gewinn
- Breadcrumb-Navigation
- Deaktivierter (statt versteckter) Button wenn kein History

## Decisions

**`window.history.state?.idx` als canGoBack-Signal**
React Router v6 setzt diesen Wert intern bei jeder Navigation. `idx === 0` bedeutet: erste Seite im Stack. Kein eigenes Tracking nötig. Wird per `useEffect([location])` neu gelesen, damit Browser-Zurück-Taste den State synchronisiert.

```tsx
const [canGoBack, setCanGoBack] = useState(() => (window.history.state?.idx ?? 0) > 0)
useEffect(() => {
  setCanGoBack((window.history.state?.idx ?? 0) > 0)
}, [location])
```

**Platzierung: Zwei Rendering-Orte, eine Logik**
- Mobile: In den bestehenden `<header>` (sm:hidden), zwischen `<button>☰</button>` und `<span>TeamWERK</span>`
- Desktop: Als `{canGoBack && <div>…</div>}` direkt vor `<Outlet />` im Content-Wrapper

**Kein neuer Hook, keine neue Komponente**
Die Logik ist minimal — sie lebt direkt in `AppShell`. Eine eigene Komponente oder ein Hook wäre Overengineering für 5 Zeilen.

**Bestehende Buttons entfernen**
`TermineDetailPage` hat zwei Buttons (oben und unten auf der Seite), `MeinTeamPage` einen, `SpieltagDetailPage` einen Link, `MembersPage` einen. Alle fliegen raus — AppShell übernimmt.

## Risks / Trade-offs

**`window.history.state?.idx` ist React Router-internes API** → Es ist nicht öffentlich dokumentiert, aber seit React Router v6.0 stabil. Sollte eine zukünftige RR-Version das ändern, bricht `canGoBack` (Button verschwindet immer oder erscheint immer). Mitigation: Unit-Test im Kontext des Routers würde das sofort zeigen; Fallback ist `window.history.length > 1` (weniger präzise, da zählt auch externe History).

**Spacing-Shift beim Einblenden** → Der Button-Block hat eine feste Höhe (~36px), die den Page-Content leicht nach unten schiebt sobald er erscheint. Mitigation: kompakte Darstellung, `py-2` max.; Pages mit eigenem `<h1>` merken das kaum.

## Migration Plan

1. AppShell ändern (Button-Logik + Rendering)
2. Lokale Zurück-Elemente aus den vier Pages entfernen
3. Kein Deploy-Risiko: rein frontend-seitig, kein Backend berührt
