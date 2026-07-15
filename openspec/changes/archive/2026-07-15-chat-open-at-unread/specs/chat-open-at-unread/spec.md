## ADDED Requirements

### Requirement: Öffnen positioniert am ersten Ungelesenen

Beim Öffnen einer Konversation SHALL das Frontend die Scroll-Position
abhängig vom `unreadCount` der Konversation bestimmen:

- `unreadCount === 0` → ans Ende scrollen (letzte Nachricht sichtbar).
- `0 < unreadCount ≤ Anzahl geladener Nachrichten` → an den `UnreadDivider`
  scrollen, der unmittelbar vor der ersten ungelesenen Nachricht liegt
  (`scrollIntoView({ block: 'start' })`).
- `unreadCount > Anzahl geladener Nachrichten` → an den obersten geladenen
  Eintrag scrollen und einen sichtbaren Hinweis-Chip anzeigen (siehe
  Requirement „Chip bei älteren Ungelesenen").

Die Divider-Position wird beim Öffnen einmal fixiert und ändert sich
während der Session nicht mehr (auch nicht durch später eintreffende
SSE-Nachrichten).

#### Scenario: Konversation ohne Ungelesenes

- **WHEN** ein User eine Konversation mit `unreadCount === 0` öffnet
- **THEN** wird an das Ende des Chatverlaufs gescrollt (letzte Nachricht
  im sichtbaren Bereich, wie im Verhalten vor diesem Change)

#### Scenario: Konversation mit Ungelesenem in der geladenen Seite

- **WHEN** ein User eine Konversation mit `unreadCount = 5` und 100
  geladenen Nachrichten öffnet
- **THEN** wird an den `UnreadDivider` gescrollt, der zwischen der 95.
  und 96. Nachricht liegt
- **AND** der Divider ist im sichtbaren Bereich (`block: 'start'`)

#### Scenario: Ungelesenes älter als geladene Seite

- **WHEN** ein User eine Konversation mit `unreadCount = 150` und 100
  geladenen Nachrichten öffnet
- **THEN** wird an die oberste geladene Nachricht gescrollt
- **AND** ein Hinweis-Chip „50 weitere ungelesene Nachrichten älter"
  wird sichtbar
- **AND** kein `UnreadDivider` wird gerendert (alle geladenen
  Nachrichten sind ungelesen)

#### Scenario: Divider bleibt statisch bei neuer eingehender Nachricht

- **GIVEN** ein User hat eine Konversation mit 3 ungelesenen Nachrichten
  geöffnet und der Divider steht an Position X
- **WHEN** über SSE eine weitere Nachricht in dieser Konversation
  eintrifft
- **THEN** bleibt der `UnreadDivider` an Position X
- **AND** die neue Nachricht wird am Ende der Liste angehängt

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
