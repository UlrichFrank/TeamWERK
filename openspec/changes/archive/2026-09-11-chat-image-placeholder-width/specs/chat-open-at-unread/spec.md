## MODIFIED Requirements

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

Für Bilder, deren Dimensionen dem Client beim Rendern bekannt sind, MUST der Platzhalter
bereits vor dem Laden des Bildes **dieselbe Layout-Höhe und -Breite** einnehmen wie das
fertige Bild. Der Übergang vom Platzhalter zum dekodierten Bild MUST die Höhe des
Scroll-Inhalts unverändert lassen; die Re-Verankerung darf für solche Bilder nicht benötigt
werden. Das gilt in jedem Container-Kontext, in dem Chat-Bilder gerendert werden,
insbesondere in einer Sprechblase, deren Breite sich an ihrem Inhalt bemisst
(shrink-to-fit) — ein Platzhalter, der dort keine eigene Breite einbringt, erfüllt diese
Anforderung nicht. Für Bilder **ohne** bekannte Dimensionen bleibt ein einmaliger
Höhenwechsel zulässig; dort trägt die Re-Verankerung die Position.

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

#### Scenario: Platzhalter mit bekannten Dimensionen verändert die Inhaltshöhe beim Decode nicht

- **GIVEN** eine Konversation, deren geladene Seite ausschließlich Bilder **mit** bekannten Dimensionen enthält, und die Bild-Downloads sind noch nicht abgeschlossen (alle Platzhalter sichtbar)
- **WHEN** die Bilder anschließend geladen und vollständig dekodiert werden
- **THEN** ist die Gesamthöhe des Scroll-Inhalts nach dem Decode gleich der Höhe vor dem Decode (Toleranz wenige Pixel für Rundung)
- **AND** kein Bild-Platzhalter hatte vor dem Decode eine kleinere Breite als das fertige Bild

#### Scenario: Reines Bild in einer inhaltsbreiten Sprechblase

- **GIVEN** eine Bild-Nachricht ohne Text mit bekannten Dimensionen in einer Sprechblase, deren Breite sich am Inhalt bemisst
- **WHEN** die Nachricht gerendert wird, bevor das Bild geladen ist
- **THEN** hat die Sprechblase bereits die Breite, die sie mit dem fertigen Bild haben wird (natürliche Bildbreite, gedeckelt auf die maximale Blasenbreite)
- **AND** der Platzhalter hat die aus Breite und Seitenverhältnis folgende Höhe (nicht 0 px, nicht nur das Blasen-Padding)

#### Scenario: Bild ohne bekannte Dimensionen bleibt tolerierter Einzelfall

- **GIVEN** eine Bild-Nachricht, für die keine Dimensionen bekannt sind (Altbestand ohne Backfill, unlesbarer Header)
- **WHEN** das Bild geladen wird
- **THEN** darf sich die Inhaltshöhe einmalig ändern
- **AND** die Öffnungs-Positionierung wird durch die fortlaufende Re-Verankerung gehalten (bestehendes Verhalten)
