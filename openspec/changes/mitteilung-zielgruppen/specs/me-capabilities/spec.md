## MODIFIED Requirements

### Requirement: Capability-Vokabular

Das System SHALL die folgenden Capability-Strings über `GET /api/me` ausliefern. Sie werden
zentral in `policy.Capabilities(claims)` berechnet; das Frontend MUSS Feature-/Button-Sichtbarkeit
ausschließlich daraus (bzw. aus per-Item `can.*`) ableiten.

| Capability | Personas (zzgl. `admin`) |
|---|---|
| `manage_members` | `vorstand` |
| `manage_games` | `vorstand`, `trainer`, `sportliche_leitung` |
| `manage_duties` | `vorstand`, `trainer`, `sportliche_leitung` |
| `manage_kader` | `vorstand`, `trainer`, `sportliche_leitung` |
| `manage_users`, `manage_seasons`, `manage_club`, `manage_duty_types` | `vorstand` |
| `manage_trainings` | `trainer`, `sportliche_leitung` |
| `fulfill_duties` | `trainer`, `sportliche_leitung` |
| `broadcast_messages` | `vorstand`, `sportliche_leitung` |
| `manage_documents` | — (nur `admin`) |
| `moderate_chat` | — (nur `admin`) |
| `impersonate` | — (nur `admin`) |

`broadcast_messages` SHALL das **einzige** Mitteilungs-Recht sein. Eine zweite, engere Stufe
(früher `broadcast_all`) SHALL es nicht geben und SHALL in keiner `GET /api/me`-Antwort mehr
vorkommen: da `admin`, `vorstand` und `sportliche_leitung` dieselben Zielgruppen wählen
dürfen, trennte sie keine zwei Mengen mehr.

`trainer` SHALL `broadcast_messages` **nicht** erhalten. Der Empfängerkreis eines Teams ist
über die Team-Standardgruppen des Chats (`GET /api/chat/team-groups`) erreichbar — mit
Rückkanal und ohne zweiten Weg zum selben Publikum.

Relationship-Marker (`is_parent`) und eigene Vereinsfunktionen für eigene Profil-Features
(z.B. `spieler` für Dienst-Erinnerungen) bleiben über die JWT-Claims abbildbar und sind KEINE
Capabilities.

#### Scenario: Trainer erhält manage_trainings, aber nicht broadcast_all
- **WHEN** ein User mit Vereinsfunktion `trainer` (ohne `vorstand`/`sportliche_leitung`) `GET /api/me` aufruft
- **THEN** enthält `capabilities` den Wert `"manage_trainings"`
- **AND** enthält NICHT `"broadcast_all"` (die Capability existiert nicht mehr)
- **AND** enthält NICHT `"broadcast_messages"`

#### Scenario: Reiner Vorstand erhält broadcast_all, aber nicht manage_trainings
- **WHEN** ein User mit Vereinsfunktion `vorstand` (ohne `trainer`/`sportliche_leitung`) `GET /api/me` aufruft
- **THEN** enthält `capabilities` den Wert `"broadcast_messages"`
- **AND** enthält NICHT `"manage_trainings"`
- **AND** enthält NICHT `"broadcast_all"` (die Capability existiert nicht mehr)

#### Scenario: Sportliche Leitung erhält broadcast_messages
- **WHEN** ein User mit Vereinsfunktion `sportliche_leitung` `GET /api/me` aufruft
- **THEN** enthält `capabilities` den Wert `"broadcast_messages"`

#### Scenario: broadcast_all existiert für keine Persona
- **WHEN** ein User beliebiger Rolle und Vereinsfunktion `GET /api/me` aufruft
- **THEN** enthält `capabilities` NICHT den Wert `"broadcast_all"`
