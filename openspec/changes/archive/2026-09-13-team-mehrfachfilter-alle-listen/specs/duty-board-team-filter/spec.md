## MODIFIED Requirements

### Requirement: Team-Filter matcht per Zugehörigkeit

Das Board `/dienste` SHALL bei aktivem Team-Filter eine Gruppe genau dann anzeigen, wenn **mindestens eine** der gewählten Mannschaften in `team_ids` der Gruppe enthalten ist. Ohne aktiven Team-Filter werden alle berechtigten Gruppen angezeigt.

Der Filter ist eine **Mehrfachauswahl** in derselben Form wie auf `/termine`: ein Dropdown mit Checkboxen, auf jeder Bildschirmbreite bedienbar, sichtbar ab zwei zugänglichen Mannschaften. Der Query-Parameter `team` trägt eine kommaseparierte ID-Liste (`team=3,7`); eine einzelne ID (`team=3`) bleibt gültig. Die leere und die vollständige Auswahl sind derselbe Zustand „kein Filter": `team` verschwindet dann aus der URL und alle Kästchen erscheinen wieder angehakt.

#### Scenario: Filter auf ein Team eines Mehr-Team-Spiels
- **WHEN** ein Nutzer auf `/dienste` nach Team B filtert und eine Gruppe ist Team A und Team B zugeordnet
- **THEN** bleibt die Gruppe sichtbar

#### Scenario: Filter auf nicht zugeordnetes Team
- **WHEN** ein Nutzer nach Team C filtert und eine Gruppe ist nur Team A und Team B zugeordnet
- **THEN** wird die Gruppe ausgeblendet

#### Scenario: Filter auf mehrere Teams
- **WHEN** ein Nutzer `/dienste?team=1,2` öffnet
- **THEN** sind Gruppen der Teams 1 und 2 sichtbar und Gruppen anderer Teams nicht

#### Scenario: Letzte Mannschaft abgewählt
- **WHEN** ein Nutzer im Team-Dropdown die letzte noch angehakte Mannschaft abwählt
- **THEN** verschwindet `team` aus der URL und alle berechtigten Gruppen sind wieder sichtbar

## ADDED Requirements

### Requirement: Fokus endet bei aktiver Filteränderung

Der Fokus-Marker (`?focus=slot-<id>` / `?focus=game-<id>`) lässt die betreffende Gruppe bewusst an Team- und Typ-Filter vorbei, damit ein Sprungziel — die Rückkehr von der Dienst-Anleitung, der Sprung aus dem Kalender-Modal (`/dienste?focus=game-<id>`) — nicht ins Leere zeigt. Dieser Durchlass SHALL enden, sobald der Nutzer den Team- oder Typ-Filter selbst ändert: `focus` verschwindet dabei aus der URL, und die zuvor fokussierte Gruppe unterliegt danach den Filtern wie jede andere.

Der Durchlass gilt dem Moment des Sprungziels, nicht der Sitzung danach — sonst bliebe eine Gruppe, die der gewählte Filter ausschließt, unbegrenzt im Board stehen. „Vergangene", „Meine" und der Textfilter beenden den Fokus NICHT.

#### Scenario: Team-Filteränderung beendet den Fokus
- **WHEN** ein Nutzer `/dienste?focus=game-2` öffnet (Spiel 2 gehört Team B) und anschließend im Team-Dropdown Team B abwählt
- **THEN** verschwindet `focus` aus der URL
- **THEN** ist die Gruppe von Spiel 2 nicht mehr sichtbar

#### Scenario: Fokus aus der URL überlebt den ersten Render
- **WHEN** ein Nutzer `/dienste?team=1&focus=game-2` öffnet und Spiel 2 gehört nicht zu Team 1
- **THEN** ist die Gruppe von Spiel 2 sichtbar und hervorgehoben
