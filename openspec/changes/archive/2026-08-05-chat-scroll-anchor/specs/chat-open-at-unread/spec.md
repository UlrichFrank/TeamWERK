## MODIFIED Requirements

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

## ADDED Requirements

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
