## MODIFIED Requirements

### Requirement: Abwesenheit anlegen

Ein Spieler (Rolle `spieler`) oder Elternteil (Rolle `elternteil`) SHALL einen Abwesenheitszeitraum (Typ `vacation` oder `injury`, Start- und Enddatum, optionale Notiz) für sich selbst bzw. ein oder mehrere verlinkte Kinder anlegen können. Ein Elternteil MUSS via `family_links` mit jedem in `member_ids` aufgeführten Member verknüpft sein. Members ohne eigenen User-Account können keine Abwesenheiten erhalten.

Werden mehrere `member_ids` übergeben, gilt **all-or-nothing**: Bei auch nur einem Konflikt oder Berechtigungs-Fehler wird kein Eintrag angelegt.

#### Scenario: Spieler legt eigene Abwesenheit an

- **WHEN** ein eingeloggter Spieler `POST /api/absences` mit `type`, `start_date`, `end_date` aufruft
- **THEN** wird eine neue Abwesenheit für seinen verlinkten Member angelegt und HTTP 201 zurückgegeben

#### Scenario: Elternteil legt Abwesenheit für ein Kind an (Legacy member_id)

- **WHEN** ein Elternteil `POST /api/absences` mit `member_id` eines verlinkten Kindes aufruft
- **THEN** wird die Abwesenheit für das Kind angelegt und HTTP 201 zurückgegeben

#### Scenario: Elternteil legt Abwesenheit für mehrere Kinder an

- **WHEN** ein Elternteil `POST /api/absences` mit `member_ids: [c1, c2]` (beide verlinkte Kinder) und einem konfliktfreien Zeitraum aufruft
- **THEN** wird je eine Zeile in `member_absences` für jedes Kind angelegt
- **AND** liefert die API HTTP 201 mit Body `{"absence_ids": [id1, id2]}` in derselben Reihenfolge wie `member_ids`

#### Scenario: Elternteil für nicht-verlinktes Kind abgewiesen

- **WHEN** ein Elternteil eine Abwesenheit für eine `member_id` (oder ein Element von `member_ids`) ohne `family_links`-Eintrag anlegen will
- **THEN** antwortet die API mit HTTP 403, **AND** es wird keine einzige Zeile angelegt (auch nicht für andere, berechtigte Kinder im selben Aufruf)

#### Scenario: Multi-Child mit Konflikt bei einem Kind — all-or-nothing

- **WHEN** ein Elternteil `member_ids: [c1, c2]` sendet und Kind `c1` hat bereits eine überlappende Abwesenheit gleichen Typs im Zeitraum
- **THEN** antwortet die API mit HTTP 409 und Body `{"error":"overlap","conflicts":[{"member_id":c1,"member_name":"…"}]}`
- **AND** es wird keine Zeile in `member_absences` angelegt — auch nicht für Kind `c2`

#### Scenario: Multi-Child mit Konflikten bei mehreren Kindern

- **WHEN** ein Elternteil `member_ids: [c1, c2, c3]` sendet und Kinder `c1` und `c3` haben überlappende Abwesenheiten
- **THEN** enthält der 409-Response `conflicts` mit beiden Einträgen (`c1` und `c3`)

### Requirement: Preview vor dem Anlegen

Das System SHALL via `GET /api/absences/preview` die Events auflisten, die bei einem geplanten Zeitraum betroffen wären (bestehende `confirmed`-Responses im Zeitraum). Es akzeptiert entweder `member_id` (Einzel) oder `member_ids` (kommaseparierte Liste). Werden mehrere IDs übergeben, ist die Antwort die deduplizierte Vereinigung der pro Member betroffenen Events.

#### Scenario: Preview ohne Konflikte

- **WHEN** der Nutzer einen Zeitraum ohne bestehende Zusagen abfragt
- **THEN** gibt die API eine leere Liste zurück

#### Scenario: Preview mit Konflikten für ein Kind

- **WHEN** der Nutzer einen Zeitraum mit mindestens einer `confirmed` Training- oder Spiel-Zusage abfragt
- **THEN** gibt die API eine Liste der betroffenen Events (Name, Datum, Typ) zurück

#### Scenario: Preview für mehrere Kinder dedupliziert Events

- **WHEN** der Nutzer `member_ids=c1,c2` mit einem Zeitraum sendet und beide Kinder eine `confirmed`-Response auf demselben Spiel haben
- **THEN** erscheint dieses Spiel **einmal** in der Antwort, nicht zweimal

## ADDED Requirements

### Requirement: Frontend Multi-Select für Elternteil mit mehreren Kindern

Der Abwesenheits-Wizard auf `KalenderPage` SHALL für eingeloggte Elternteile die Anzahl ihrer verlinkten Kinder berücksichtigen:

- **0 Kinder** (z.B. Spieler-Account): keine Kind-Auswahl, Abwesenheit gilt automatisch für den eigenen Member
- **1 Kind**: keine Kind-Auswahl-UI rendern; das eine Kind wird automatisch verwendet
- **>1 Kinder**: Checkbox-Liste mit den Kindernamen, mindestens eines muss ausgewählt werden, sonst Submit blockiert

Typ, Start-/Enddatum und Notiz gelten einheitlich für alle ausgewählten Kinder.

#### Scenario: Elternteil mit 2 Kindern öffnet Wizard

- **WHEN** ein Elternteil mit 2 verlinkten Kindern den Abwesenheits-Schritt im Wizard öffnet
- **THEN** sieht er eine Checkbox-Liste mit beiden Kindernamen
- **AND** beide sind initial nicht aktiv (Default-Zustand)

#### Scenario: Elternteil mit 1 Kind öffnet Wizard

- **WHEN** ein Elternteil mit genau 1 verlinkten Kind den Abwesenheits-Schritt im Wizard öffnet
- **THEN** wird kein Auswahl-Element gerendert
- **AND** die nachfolgenden Aufrufe (Preview, POST) verwenden automatisch dieses eine Kind

#### Scenario: Submit ohne ausgewähltes Kind blockiert

- **WHEN** ein Elternteil mit mehreren Kindern „Speichern" klickt, ohne ein Kind angehakt zu haben
- **THEN** erscheint der Fehler „Bitte mindestens ein Kind auswählen."
- **AND** es wird kein Request an die API gesendet

#### Scenario: Konflikt-Fehler nennt das betroffene Kind

- **WHEN** der POST aufgrund von Konflikten mit HTTP 409 und `conflicts[]` zurückkommt
- **THEN** zeigt der Wizard eine Fehlermeldung mit allen genannten Kindernamen („Eintragung abgebrochen — {Name1}, {Name2} hat/haben in diesem Zeitraum bereits eine Abwesenheit.")
- **AND** der Wizard bleibt geöffnet, damit der Nutzer korrigieren kann
