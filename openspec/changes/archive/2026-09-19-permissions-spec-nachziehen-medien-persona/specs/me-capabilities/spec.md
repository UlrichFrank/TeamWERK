## MODIFIED Requirements

### Requirement: Capability-Vokabular

Das System SHALL die folgenden Capability-Strings über `GET /api/me` ausliefern. Sie werden
zentral in `policy.Capabilities(claims)` berechnet; das Frontend MUSS Feature-/Button-Sichtbarkeit
ausschließlich daraus (bzw. aus per-Item `can.*`) ableiten. Die Liste ist abschließend: eine
Capability, die `policy.Capabilities` liefert, MUSS hier stehen, und umgekehrt.

| Capability | Personas (zzgl. `admin`) |
|---|---|
| `manage_members` | `vorstand` |
| `manage_games` | `vorstand`, `trainer`, `sportliche_leitung` |
| `import_games` | `vorstand` |
| `manage_duties` | `vorstand`, `trainer`, `sportliche_leitung` |
| `manage_kader` | `vorstand`, `trainer`, `sportliche_leitung` |
| `manage_users`, `manage_seasons`, `manage_duty_types` | `vorstand` |
| `manage_club`, `manage_fees` | `vorstand`, `kassierer` |
| `manage_trainings` | `trainer`, `sportliche_leitung` |
| `fulfill_duties` | `trainer`, `sportliche_leitung` |
| `broadcast_messages` | `vorstand`, `sportliche_leitung`, `trainer` |
| `create_root_folder` | `vorstand` |
| `suppress_event_notification` | `vorstand` |
| `bulk_regen_duties` | `vorstand` |
| `manage_documents` | — (nur `admin`) |
| `moderate_chat` | — (nur `admin`) |
| `impersonate` | — (nur `admin`) |

`broadcast_messages` SHALL das **einzige** Mitteilungs-Recht sein. Eine zweite, engere Stufe
(früher `broadcast_all`) SHALL es nicht geben und SHALL in keiner `GET /api/me`-Antwort mehr
vorkommen. Für `trainer` bedeutet die Capability nur den Zugang zum Mitteilungs-Composer; welche
Ziele er wählen darf (ausschließlich die Standardgruppen der Kader, die er in der aktiven Saison
trainiert), entscheidet der Server pro Absender über die Ziel-Allowlist (siehe `chat-broadcasts`).
Ein Trainer ohne Kader in der aktiven Saison hat die Capability, aber eine leere Ziel-Allowlist.

Relationship-Marker (`is_parent`) und eigene Vereinsfunktionen für eigene Profil-Features
(z.B. `spieler` für Dienst-Erinnerungen, `medien` für die Spielbericht-Freigabe) bleiben über
die JWT-Claims abbildbar und sind KEINE Capabilities.

#### Scenario: Trainer erhält manage_trainings, aber nicht broadcast_all
- **WHEN** ein User mit Vereinsfunktion `trainer` (ohne `vorstand`/`sportliche_leitung`) `GET /api/me` aufruft
- **THEN** enthält `capabilities` die Werte `"manage_trainings"` und `"broadcast_messages"`
- **AND** enthält NICHT `"broadcast_all"` (die Capability existiert nicht mehr)
- **AND** enthält NICHT `"manage_members"`

#### Scenario: Reiner Vorstand erhält broadcast_all, aber nicht manage_trainings
- **WHEN** ein User mit Vereinsfunktion `vorstand` (ohne `trainer`/`sportliche_leitung`) `GET /api/me` aufruft
- **THEN** enthält `capabilities` den Wert `"broadcast_messages"`
- **AND** enthält NICHT `"manage_trainings"` und NICHT `"fulfill_duties"`
- **AND** enthält NICHT `"broadcast_all"` (die Capability existiert nicht mehr)

#### Scenario: Sportliche Leitung erhält broadcast_messages
- **WHEN** ein User mit Vereinsfunktion `sportliche_leitung` `GET /api/me` aufruft
- **THEN** enthält `capabilities` den Wert `"broadcast_messages"`

#### Scenario: broadcast_all existiert für keine Persona
- **WHEN** ein User beliebiger Rolle und Vereinsfunktion `GET /api/me` aufruft
- **THEN** enthält `capabilities` NICHT den Wert `"broadcast_all"`

#### Scenario: Kassierer erhält genau die Finanz-Capabilities
- **WHEN** ein User mit Vereinsfunktion `kassierer` (ohne weitere Funktionen) `GET /api/me` aufruft
- **THEN** enthält `capabilities` genau `"manage_club"` und `"manage_fees"`

#### Scenario: Vokabular ist deckungsgleich mit der Implementierung
- **WHEN** die Menge aller Capability-Strings, die `policy.Capabilities` für irgendeine Persona liefern kann, mit der Tabelle verglichen wird
- **THEN** sind beide Mengen identisch
