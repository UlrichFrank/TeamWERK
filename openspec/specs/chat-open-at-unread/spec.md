# chat-open-at-unread Specification

## Purpose
TBD - created by archiving change chat-open-at-unread. Update Purpose after archive.
## Requirements
### Requirement: Öffnen positioniert am ersten Ungelesenen

Beim Öffnen einer Konversation SHALL das Frontend die Scroll-Position abhängig vom
`unreadCount` der Konversation bestimmen:

- `unreadCount === 0` → ans Ende scrollen (letzte Nachricht sichtbar).
- `0 < unreadCount ≤ Anzahl geladener Nachrichten` → an den `UnreadDivider` scrollen, der
  unmittelbar vor der ersten ungelesenen Nachricht liegt (`scrollIntoView({ block: 'start' })`).
- `unreadCount > Anzahl geladener Nachrichten` → an den obersten geladenen Eintrag scrollen
  und einen sichtbaren Hinweis-Chip anzeigen (siehe Requirement „Chip bei älteren Ungelesenen").

Die gewählte Zielposition SHALL **fortlaufend gehalten** werden, nicht nur einmalig
angewandt: Solange nach dem Öffnen noch Bilder laden oder decoden bzw. sich das Layout im
Container ändert, MUST die Position bei jeder Layout-Änderung erneut angewandt werden — für
den `UnreadDivider` ebenso wie fürs Ende. Die Haltephase endet, sobald alle Medien fertig
sind ODER der Nutzer selbst scrollt (Wheel/Touch/Tastatur oder Scrollbar-Drag).

Die Divider-Position wird beim Öffnen einmal fixiert und ändert sich während der Session
nicht mehr (auch nicht durch später eintreffende SSE-Nachrichten).

#### Scenario: Konversation ohne Ungelesenes

- **WHEN** ein User eine Konversation mit `unreadCount === 0` öffnet
- **THEN** wird an das Ende des Chatverlaufs gescrollt (letzte Nachricht im sichtbaren Bereich)

#### Scenario: Konversation mit Ungelesenem in der geladenen Seite

- **WHEN** ein User eine Konversation mit `unreadCount = 5` und 100 geladenen Nachrichten öffnet
- **THEN** wird an den `UnreadDivider` gescrollt, der zwischen der 95. und 96. Nachricht liegt
- **AND** der Divider ist im sichtbaren Bereich (`block: 'start'`)

#### Scenario: Ungelesenes älter als geladene Seite

- **WHEN** ein User eine Konversation mit `unreadCount = 150` und 100 geladenen Nachrichten öffnet
- **THEN** wird an die oberste geladene Nachricht gescrollt
- **AND** ein Hinweis-Chip „50 weitere ungelesene Nachrichten älter" wird sichtbar
- **AND** kein `UnreadDivider` wird gerendert

#### Scenario: Divider bleibt statisch bei neuer eingehender Nachricht

- **GIVEN** ein User hat eine Konversation mit 3 ungelesenen Nachrichten geöffnet und der Divider steht an Position X
- **WHEN** über SSE eine weitere Nachricht in dieser Konversation eintrifft
- **THEN** bleibt der `UnreadDivider` an Position X
- **AND** die neue Nachricht wird am Ende der Liste angehängt

### Requirement: Positionierung überlebt asynchrones Nachladen von Bildern

Die Öffnungs-Positionierung SHALL auch dann korrekt bleiben, wenn Bilder erst **nach** dem
initialen Scroll laden und decoden und dadurch das Layout verschieben — **unabhängig davon,
ob der Browser CSS scroll-anchoring (`overflow-anchor`) unterstützt** (iOS Safari tut das
nicht). Der Client MUST die Zielposition selbst tragen (fortlaufende Re-Verankerung), statt
sich auf Browser-scroll-anchoring zu verlassen. Die Re-Verankerung MUST enden, sobald keine
Medien mehr ausstehen (alle `AuthImage`-Platzhalter aufgelöst und alle `img` dekodiert) oder
ein absolutes Zeitlimit erreicht ist, und MUST durch echte Nutzer-Eingabe jederzeit sofort
freigegeben werden. Bei Freigabe MUST der Sticky-Zustand aus der tatsächlichen Scroll-Position
abgeleitet werden.

#### Scenario: Divider bleibt oben, nachdem Bilder darüber decoden (ohne Browser-scroll-anchoring)

- **GIVEN** eine lange Konversation mit `unreadCount = 40`, deren geladene Seite ein Bild ohne Server-Dimensionen ÜBER dem Divider enthält, in einem Umfeld ohne CSS scroll-anchoring (z. B. iOS Safari)
- **WHEN** der User die Konversation öffnet und die Bilder anschließend vollständig decoden
- **THEN** bleibt der `UnreadDivider` am oberen Rand des Scroll-Containers (er driftet NICHT aus dem Viewport)

#### Scenario: Zuverlässig am Ende trotz spät decodender Bilder

- **WHEN** der User eine komplett gelesene, bildlastige Konversation öffnet und die Bilder erst nach dem initialen End-Scroll ihre Höhe annehmen
- **THEN** steht der Container nach dem Decode weiterhin am Ende (letzte Nachricht sichtbar)

#### Scenario: Nutzer-Scroll gibt den Anker sofort frei

- **GIVEN** eine gerade geöffnete Konversation, deren Bilder noch laden (Anker aktiv)
- **WHEN** der User selbst scrollt (Mausrad, Touch, Tastatur oder Scrollbar-Drag)
- **THEN** wird der Anker freigegeben und die Position folgt fortan dem User; nachfolgende Bild-Loads reißen ihn nicht zurück

#### Scenario: „Ältere laden" behält die Position über decodende voran-gestellte Bilder

- **GIVEN** der Chip-Fall (erste Ungelesene älter als die geladene Seite), Container oben
- **WHEN** der User „Ältere Nachrichten laden" klickt und die voran-gestellten älteren Nachrichten inkl. Bilder decoden
- **THEN** bleibt der zuvor sichtbare Inhalt an derselben Stelle (die Ansicht springt nicht ans Ende oder nach oben weg)

### Requirement: UnreadDivider als visuelle Trennlinie

Der `UnreadDivider` SHALL eine horizontale visuelle Trennlinie zwischen
der letzten gelesenen und der ersten ungelesenen Nachricht rendern. Text:
„N ungelesene Nachrichten" (mit N = `unreadCount` beim Öffnen). Layout
konsistent mit dem bestehenden `DaySeparator` (zentrierter Text, dünne
Linie, `brand-text-muted`-Ton).

#### Scenario: Divider zeigt korrekten Zähler

- **WHEN** eine Konversation mit `unreadCount = 7` geöffnet wird
- **THEN** rendert der Divider den Text „7 ungelesene Nachrichten"

#### Scenario: Divider verschwindet nach Konversationswechsel

- **GIVEN** Konversation A wurde mit Divider geöffnet
- **WHEN** der User zu Konversation B wechselt und wieder zurück zu A
- **THEN** hat Konversation A jetzt `unreadCount === 0` (durch den
  zwischenzeitlichen `POST /read`) und wird ohne Divider ans Ende gescrollt

### Requirement: Chip bei älteren Ungelesenen

Wenn `unreadCount > Anzahl geladener Nachrichten` gilt, SHALL ein
Hinweis-Chip oberhalb der Nachrichtenliste (unter dem existierenden
„Ältere Nachrichten laden"-Button, wenn vorhanden) gerendert werden. Text:
„M weitere ungelesene Nachrichten älter — 'Ältere laden' klicken" (mit
M = `unreadCount - Anzahl geladener Nachrichten`). Der Chip verschwindet
automatisch, sobald durch Laden älterer Nachrichten die Bedingung nicht
mehr erfüllt ist.

#### Scenario: Chip wird angezeigt

- **WHEN** eine Konversation mit `unreadCount = 120` und 100 geladenen
  Nachrichten geöffnet wird
- **THEN** ist der Chip mit dem Text „20 weitere ungelesene Nachrichten
  älter" sichtbar

#### Scenario: Chip verschwindet nach Nachladen

- **GIVEN** der Chip zeigt „20 weitere ungelesene Nachrichten älter"
- **WHEN** der User „Ältere laden" klickt und 100 weitere Nachrichten
  geladen werden (nun 200 im Frontend, `unreadCount = 120` unverändert)
- **THEN** verschwindet der Chip (weil `unreadCount ≤ 200`)

#### Scenario: Chip erscheint nicht bei geringem Ungelesenem

- **WHEN** eine Konversation mit `unreadCount = 5` und 100 geladenen
  Nachrichten geöffnet wird
- **THEN** ist kein Chip sichtbar (Divider übernimmt die visuelle Rolle)

### Requirement: Konversationswechsel zeigt niemals Fremdinhalt

Beim Öffnen oder Wechseln einer Konversation SHALL der Chat-Bereich zu jedem Zeitpunkt
konsistent sein: Header und Nachrichtenliste MUST zur **selben** Konversation gehören. Es
darf keinen darstellbaren Zwischenzustand geben, in dem der Header die neu gewählte
Konversation zeigt, während die Nachrichtenliste noch Nachrichten der zuvor geöffneten
Konversation enthält.

Da die Nachrichten der Zielkonversation erst nach einem Netzwerk-Roundtrip vorliegen, MUST
die Nachrichtenliste beim Wechsel **geleert** werden, bevor der Abruf startet. Während des
Abrufs SHALL ein Ladezustand angezeigt werden, der von „Konversation ohne Nachrichten"
unterscheidbar ist. Der Ladezustand MUST auch dann enden, wenn der Abruf fehlschlägt.

Diese Invariante MUST unabhängig von Browser-Engine, Gerät und Netzwerklaufzeit gelten; sie
darf sich nicht darauf verlassen, dass der Abruf schneller ist als der nächste Frame.

Das Leeren betrifft ausschließlich den **Wechsel** der Konversation. Aktualisierungen einer
bereits geöffneten Konversation (eintreffende SSE-Nachricht, Reaktion, Austritt eines
Mitglieds, Nachladen älterer Nachrichten) MUST die Liste NICHT leeren und keinen Ladezustand
auslösen.

#### Scenario: Wechsel zwischen zwei Konversationen

- **GIVEN** ein User hat Konversation A geöffnet und deren Nachrichten sind sichtbar
- **WHEN** er in der Konversationsliste Konversation B auswählt und der Abruf der Nachrichten von B noch nicht abgeschlossen ist
- **THEN** zeigt der Header Konversation B
- **AND** es ist KEINE Nachricht aus Konversation A sichtbar
- **AND** ein Ladezustand ist sichtbar

#### Scenario: Nachrichten treffen ein

- **GIVEN** der Ladezustand aus dem vorherigen Szenario
- **WHEN** der Abruf der Nachrichten von B auflöst
- **THEN** verschwindet der Ladezustand
- **AND** die Nachrichten von B sind sichtbar und gemäß den Positionierungs-Anforderungen dieser Capability positioniert

#### Scenario: Abruf schlägt fehl

- **GIVEN** ein User wechselt zu Konversation B
- **WHEN** der Abruf der Nachrichten fehlschlägt
- **THEN** endet der Ladezustand
- **AND** es ist KEINE Nachricht aus der zuvor geöffneten Konversation sichtbar

#### Scenario: Eingehende Nachricht in der offenen Konversation leert nicht

- **GIVEN** ein User hat Konversation A geöffnet und liest im Verlauf
- **WHEN** über SSE eine neue Nachricht in A eintrifft
- **THEN** bleibt die bestehende Nachrichtenliste sichtbar
- **AND** es wird KEIN Ladezustand angezeigt

