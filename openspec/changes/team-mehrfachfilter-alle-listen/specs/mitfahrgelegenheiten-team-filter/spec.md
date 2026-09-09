## MODIFIED Requirements

### Requirement: Team-Dropdown filtert Mitfahrgelegenheiten

Die Mitfahrgelegenheiten-Seite SHALL einen Mannschafts-Filter anzeigen, wenn der Nutzer Zugang zu mehr als einem Team hat — als Dropdown mit Checkboxen (Mehrfachauswahl) in derselben Form wie auf `/termine`, auf jeder Bildschirmbreite bedienbar. Der gewählte Filter steht als kommaseparierte ID-Liste im Query-Parameter `team` (`?team=3,7`); eine einzelne ID bleibt gültig. Die leere und die vollständige Auswahl sind derselbe Zustand „kein Filter": `team` verschwindet dann aus der URL und alle Kästchen erscheinen wieder angehakt.

Der Filter SHALL **clientseitig** auf der geladenen Menge wirken, nicht über `?team_id=` an `GET /api/mitfahrgelegenheiten`: die Antwort trägt `teamIds` je Spiel, und die ungefilterte Menge ist ohnehin die geladene. Ein Filterklick löst damit keine neue Anfrage aus, und die Mannschaftsliste eines Mehr-Team-Spiels bleibt vollständig sichtbar (der serverseitige Filter verkürzte über `GROUP_CONCAT` auch die angezeigten Mannschaften auf die gefilterte). Ein Spiel ist sichtbar, sobald **mindestens eine** seiner Mannschaften gewählt ist. Der Query-Parameter `team_id` der API bleibt unverändert bestehen; die Seite nutzt ihn nur nicht mehr.

#### Scenario: Nutzer mit einem Team sieht keinen Dropdown
- **WHEN** ein Nutzer Zugang zu genau einem Team hat
- **THEN** ist kein Mannschafts-Filter sichtbar und alle Events dieses Teams werden ohne Filterinteraktion angezeigt

#### Scenario: Nutzer mit mehreren Teams kann filtern
- **WHEN** ein Nutzer Zugang zu mehreren Teams hat und im Dropdown eine Mannschaft abwählt
- **THEN** zeigt die Seite nur noch Events der übrigen Mannschaften
- **THEN** wird dafür keine neue Anfrage an `GET /api/mitfahrgelegenheiten` gestellt

#### Scenario: Filter auf mehrere Teams
- **WHEN** ein Nutzer `/mitfahrten?team=1,2` öffnet
- **THEN** sind Events der Teams 1 und 2 sichtbar und Events anderer Teams nicht

#### Scenario: Kein Filter zeigt alle zugänglichen Teams
- **WHEN** ein Nutzer mit mehreren Teams die letzte angehakte Mannschaft abwählt
- **THEN** verschwindet `team` aus der URL und Events aller zugänglichen Teams werden angezeigt

#### Scenario: Team-Filter und Ansicht (Teams/Meine) sind kombinierbar
- **WHEN** ein Nutzer gleichzeitig einen Team-Filter und „Meine" aktiviert hat
- **THEN** werden nur eigene Einträge der gewählten Mannschaften angezeigt
