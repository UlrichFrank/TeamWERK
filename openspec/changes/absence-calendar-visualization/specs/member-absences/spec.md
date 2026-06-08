## ADDED Requirements

### Requirement: Überlappungsschutz gleicher Abwesenheitstypen
Das System SHALL verhindern, dass für denselben Member zwei Abwesenheiten desselben Typs angelegt werden, deren Zeiträume sich überschneiden.

#### Scenario: Gleicher Typ überschneidet sich
- **WHEN** ein Nutzer `POST /api/absences` aufruft und der Member bereits eine Abwesenheit desselben `type` hat, deren `[start_date, end_date]` den neuen Zeitraum überlappt
- **THEN** antwortet die API mit HTTP 409 und Body `{"error":"overlap"}`

#### Scenario: Verschiedene Typen im gleichen Zeitraum erlaubt
- **WHEN** ein Nutzer `POST /api/absences` mit `type=injury` aufruft und der Member bereits eine `vacation`-Abwesenheit im gleichen Zeitraum hat
- **THEN** wird die neue Abwesenheit angelegt und HTTP 201 zurückgegeben

#### Scenario: Angrenzende Zeiträume gleichen Typs erlaubt
- **WHEN** ein Nutzer `POST /api/absences` aufruft und der neue Zeitraum beginnt genau einen Tag nach dem Ende einer bestehenden Abwesenheit gleichen Typs
- **THEN** wird die neue Abwesenheit angelegt und HTTP 201 zurückgegeben

## MODIFIED Requirements

### Requirement: Kalender-Banner im Frontend
Die `KalenderPage` SHALL Abwesenheitszeiträume als farbige horizontale Fläche hinter dem Tag-Inhalt anzeigen. Die Fläche ist absolut positioniert (`absolute inset-x-0 top-0 h-5`) innerhalb der Kalenderzelle. Der Tag-Inhalt (Zahl, Events) liegt über dem Balken (`relative z-10`). Das bestehende Cell-Padding (`p-1.5`) bildet den gleichmäßigen Abstand zu allen Zell-Trennlinien. Abwesenheiten vom Typ `vacation` werden mit `bg-brand-yellow/20` dargestellt, `injury` mit `bg-red-400/20`. Sind beide Typen am gleichen Tag vorhanden, überlagern sich die transparenten Flächen. Der Balken erhält Radius nur am ersten Tag (linke Ecken: `rounded-l`) und letzten Tag (rechte Ecken: `rounded-r`); ein eintägiger Zeitraum erhält `rounded`; Mitteltage bleiben eckig.

#### Scenario: Eintägige Abwesenheit
- **WHEN** eine Abwesenheit genau einen Tag umfasst
- **THEN** erscheint ein Balken mit `rounded` (alle Ecken abgerundet) hinter der Tag-Zahl

#### Scenario: Mehrtägige Abwesenheit — erster Tag
- **WHEN** es sich um den ersten Tag einer mehrtägigen Abwesenheit handelt (oder den ersten Tag nach einer Wochengrenze)
- **THEN** hat der Balken `rounded-l` (linke Ecken abgerundet, rechte eckig)

#### Scenario: Mehrtägige Abwesenheit — Mitteltag
- **WHEN** es sich um einen mittleren Tag einer mehrtägigen Abwesenheit handelt
- **THEN** hat der Balken keine Rundung (eckiges Rechteck)

#### Scenario: Mehrtägige Abwesenheit — letzter Tag
- **WHEN** es sich um den letzten Tag einer mehrtägigen Abwesenheit handelt (oder den letzten Tag vor einer Wochengrenze)
- **THEN** hat der Balken `rounded-r` (rechte Ecken abgerundet, linke eckig)

#### Scenario: Urlaub und Verletzung am gleichen Tag
- **WHEN** ein Member am selben Tag sowohl eine `vacation`- als auch eine `injury`-Abwesenheit hat
- **THEN** erscheinen beide Balken übereinander; die Transparenz beider Farben mischt sich sichtbar

#### Scenario: Abwesenheit über Wochengrenze
- **WHEN** eine Abwesenheit Mo–So einer Woche und darüber hinaus geht
- **THEN** erscheinen separate Banner-Segmente für jede betroffene Woche im Kalender
