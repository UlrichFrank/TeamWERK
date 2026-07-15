## Context

Der `chat-image-dimensions`-Change (b9e472c) hat den Auto-Scroll ans Ende
robust gemacht, indem `openConversation` `forceScrollToEndRef = true` setzt
und der Auto-Scroll-Effekt darauf reagiert. Kollateral: **jedes** Öffnen
landet am Ende, auch wenn 40 ungelesene Nachrichten darüber liegen. Nutzer
müssen von Hand hochscrollen und blind raten, wo der neue Verlauf anfängt.

Der `unreadCount` in der Konversation ist server-authoritativ (`SELECT COUNT`
aus `message_reads`, siehe `handler.go:194-210`) und wird schon bei jedem
`GET /api/chat/conversations` mitgeliefert. Wir müssen also nur beim Öffnen
die Positionierungs-Logik verzweigen.

Die Messages-Liste ist chronologisch aufsteigend sortiert. Da beim
`loadMessages` sofort `POST /api/chat/conversations/{id}/read` folgt
(handler.go:947), sind alle ungelesenen Nachrichten am Tail. Die
Berechnung reduziert sich auf ein Index:

```
firstUnreadIndex = messages.length - conv.unreadCount
```

Beispiel: 100 geladen, 3 ungelesen → Divider vor Index 97, ans Index 97
scrollen.

Randbedingung: `unreadCount` reflektiert die *ganze* Konversation, nicht nur
die geladene 100er-Seite. Ist `unreadCount > messages.length`, liegt der
Divider vor der Seite → wir zeigen einen Chip statt Divider und positionieren
am obersten geladenen Eintrag.

## Goals / Non-Goals

**Goals:**
- Öffnen einer Konversation mit Ungelesenem positioniert am ersten
  ungelesenen Eintrag, mit sichtbarer „N ungelesene Nachrichten"-Grenze.
- Öffnen einer Konversation ohne Ungelesenes: Verhalten unverändert (ans
  Ende).
- Nach dem Positionieren funktioniert die vorhandene Sticky-Scroll-Logik
  weiter — wer nach unten scrollt und neue Nachrichten ankommen sieht,
  bleibt am Ende; wer hochscrollt, bleibt oben.
- Fall „Ungelesenes älter als geladene Seite" ist explizit adressiert (Chip,
  Position am Top).

**Non-Goals:**
- **Kein serverseitiges Nachladen** der Seite mit dem ersten ungelesenen
  (Option b aus der Explore-Diskussion). Der Chip macht den Zustand
  transparent; der Nutzer klickt „Ältere laden" im Bedarfsfall.
- **Kein persistenter Scroll-Position-Speicher** (LocalStorage, Session).
  „Letzter Stand" ist definiert als „erster ungelesener" — nicht als
  Pixel-Offset.
- **Kein „als ungelesen markieren"-Feature.** Der Divider steht immer an der
  serverseitig eindeutigen Grenze (kleinste ungelesene ID).
- **Keine Divider-Aktualisierung während man in der Konversation ist.** Der
  Divider wird beim Öffnen fixiert und ändert sich nicht mit eingehenden
  SSE-Nachrichten — das würde die Position unter dem Nutzer verschieben.
  Neu eingehende Nachrichten landen einfach unten, wie heute.

## Decisions

### Decision 1: `firstUnreadIndex` client-seitig aus `unreadCount` berechnen

**Was**: Kein neues Response-Feld. Frontend rechnet
`messages.length - conv.unreadCount`.

**Warum**: Der Wert ist trivial und schon aus der bestehenden
`Conversation`-Response ableitbar. Ein neues `firstUnreadMessageId`-Feld
wäre auch möglich, aber:
- macht `handler.go` breiter, ohne dass es die Berechnung vereinfacht
- das Ergebnis wäre stale, sobald der `POST /read` läuft (sofort nach
  `loadMessages`), also müssten wir es „ab-open"-cachen — Client-Berechnung
  passiert an derselben Stelle und ist expliziter.

**Alternativen**:
- Server liefert `firstUnreadId` in Message-Response → mehr Query-Komplexität
  ohne Nutzen.
- Client speichert `lastReadId` pro Konversation in LocalStorage → verliert
  bei Session-Reset; widerspricht dem „Server ist authoritativ"-Prinzip.

### Decision 2: `UnreadDivider` als eigene Komponente, Muster wie `DaySeparator`

**Was**: Neuer Komponent `UnreadDivider` (in `ChatPage.tsx` inline, wie
`DaySeparator` bei Zeile ~2210), rendert eine horizontale Trennlinie mit
zentriertem Text „N ungelesene Nachrichten". Wird als Pseudo-Zeile in der
`WindowedRows`-`renderRow`-Callback eingefügt — genauso wie `DaySeparator`
mit dem `contents`-Wrapper (`ChatPage.tsx:986`).

**Warum**: Konsistent mit bestehendem Divider-Pattern; keine neue
Layout-Kategorie. Nutzt bereits deaktiviertes Windowing (`threshold=Infinity`
seit `b01a657`), also keine Interaktion mit Höhen-Schätzung.

**Alternativen**:
- Divider als sticky-header — bricht das lineare Scroll-Muster, komplexer.
- Divider als eigene DOM-Ebene über der Liste — Layer-Overhead ohne Nutzen.

### Decision 3: `unreadTargetRef` statt Umbau von `forceScrollToEndRef`

**Was**: Neuer `unreadTargetRef` (Callback-Ref auf den `UnreadDivider`-Node);
in `openConversation` entscheidet die Logik nach `loadMessages`:

```
if (conv.unreadCount === 0) {
  forceScrollToEndRef.current = true;   // wie heute
} else {
  scrollToUnreadRef.current = true;     // neuer Pfad
}
```

Der bestehende Auto-Scroll-Effekt bekommt einen zweiten Zweig: wenn
`scrollToUnreadRef` gesetzt, `unreadDividerRef.current?.scrollIntoView({ block: 'start' })`;
sonst der bisherige `messagesEndRef`-Pfad.

**Warum**: Minimaler Diff, kein Refactor der Sticky-Scroll-Logik.
`forceScrollToEndRef` bleibt für Sende-Fall und `unreadCount === 0` intakt.

**Alternativen**:
- Ein einzelnes `scrollTargetRef` mit Union-Typ — kognitiv teurer, kein
  echter Gewinn.

### Decision 4: „Ältere-ungelesene"-Chip als Bedingung `unreadCount > messages.length`

**Was**: Über der `WindowedRows`-Liste (unterhalb des existierenden
„Ältere Nachrichten laden"-Buttons) wird ein Chip gerendert, wenn
`unreadCount > messages.length`. Text: „N weitere ungelesene Nachrichten
älter — 'Ältere laden' klicken". Chip verschwindet, sobald der User älter
geladen hat und die Bedingung nicht mehr gilt.

**Warum**: Ehrlichkeit ohne Backend-Umbau. Der 99-%-Fall (weniger als 100
Ungelesene) sieht den Chip nie. Für den Ausnahme-Fall bleibt es
transparent.

**Alternativen**:
- Server-Seitiges Fenster-Nachladen (siehe Non-Goals): sauber, aber teuer
  und für den seltenen Fall unangemessen.
- Automatisch mehrere Seiten laden bis der erste ungelesene drin ist: RAM/
  DOM-Explosion bei extremen Ausreißern.

### Decision 5: Divider bleibt statisch nach dem Öffnen

**Was**: Sobald `openConversation` den Divider positioniert hat, ändert sich
seine Position nicht mehr — auch nicht, wenn während der Session neue
Nachrichten via SSE reintropfen oder der Nutzer manuell hochscrollt.
Konkret: Divider-Index wird beim `loadMessages` einmal fixiert (in einem
State), nicht live neu berechnet.

**Warum**: Wenn der Divider mit neuen SSE-Messages nach unten wandert, würde
die Scroll-Position unter dem Nutzer springen (das war ja der Auslöser des
ganzen chat-image-dimensions-Fixes). Ein statischer Divider markiert den
Zustand *beim Öffnen* — das ist auch semantisch richtig („was war neu, als
ich reinkam").

**Alternativen**:
- Live-Neuberechnung → identische Bug-Klasse wie vorher.

## Risks / Trade-offs

- **Sonderfall „Konversation frisch beigetreten mit vielen alten
  Nachrichten"**: unreadCount könnte sehr groß sein (alle Nachrichten seit
  Beitritt, evtl. hunderte). Trigger vermutlich Option-c-Fall → landet am
  Top-Item + Chip. Nutzer sieht sofort, dass es was zu tun gibt. *Kein
  Regressionsrisiko*.

- **User war schon vorher drin und hat manuell hochgescrollt zum Recherchieren
  → verlässt und kommt zurück**: unreadCount hat sich zwischenzeitlich nicht
  erhöht (er hat ja gelesen), also unreadCount = 0 → landet am Ende. Verliert
  seine Scroll-Position vom Recherchieren. **Bewusst akzeptierter Trade-off**
  (nicht Non-Goal für nichts) — „letzter Stand" ist definiert als erster
  ungelesener, nicht als Scroll-Pixel.

- **`POST /read` läuft, während Divider gerendert wird**: die
  `messages`-Antwort ist von *vor* dem read-POST, `unreadCount` in der
  Konversations-Response ebenfalls. Nach `POST /read` läuft ein
  `loadConversations()` (handler.go:270), der frische `unreadCount = 0`
  bringt. Wenn wir das für den nächsten Divider verwenden würden → falsch.
  Mitigation: Divider-Index wird beim ersten `loadMessages` einmal fixiert
  (Decision 5).

- **Windowing-Zeilenhöhen**: Da Windowing im Chat seit `b01a657` deaktiviert
  ist (`threshold=Infinity`), ist `scrollIntoView` auf einen echten
  DOM-Knoten trivial korrekt. Wäre Windowing aktiv, müsste der Divider im
  gerenderten Fenster liegen — irrelevant für uns.

- **Race Divider ↔ AuthImage-Blob-Load**: Nach dem `chat-image-dimensions`-
  Fix haben Bilder ab dem ersten Frame ihre Aspect-Ratio → kein Layout-Shift
  → Divider-Position bleibt stabil. *Kein neuer Race*.

## Migration Plan

Reines Frontend-Feature, kein Migrations-, DB- oder API-Change. Deploy des
neuen Frontend-Bundles reicht. Rollback = altes Bundle → landet wieder
immer am Ende (aktuelles Verhalten). Kein Feature-Flag nötig — der Effekt
ist rein UI und additiv.

## Open Questions

- Keine.
