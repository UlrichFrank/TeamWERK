## ADDED Requirements

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

### Requirement: Abwesenheiten auflisten
Das System SHALL dem eingeloggten Nutzer seine eigenen Abwesenheiten und — für Elternteile — die seiner verlinkten Kinder zurückgeben.

#### Scenario: Eigene Abwesenheiten abrufen
- **WHEN** ein eingeloggter Nutzer `GET /api/absences` aufruft
- **THEN** erhält er alle Abwesenheiten, für die er berechtigt ist (eigene + Kinder)

### Requirement: Abwesenheit löschen
Der Ersteller einer Abwesenheit (oder ein Admin) SHALL sie löschen können. Beim Löschen werden alle auto-declined Responses mit dieser `absence_id` per CASCADE entfernt.

#### Scenario: Eigene Abwesenheit löschen
- **WHEN** der Ersteller `DELETE /api/absences/{id}` aufruft
- **THEN** wird die Abwesenheit gelöscht und alle zugehörigen auto-declined Responses entfernt

#### Scenario: Fremde Abwesenheit löschen abgewiesen
- **WHEN** ein Nutzer eine Abwesenheit löschen will, die nicht ihm gehört und er kein Admin ist
- **THEN** antwortet die API mit HTTP 403

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

### Requirement: Auto-decline beim Anlegen
Beim Anlegen einer Abwesenheit SHALL das System für alle `training_sessions` und `games` im Zeitraum, bei denen der Member im Kader ist, eine `declined`-Response mit gesetztem `absence_id` anlegen (INSERT OR REPLACE). Bestehende `confirmed`/`maybe`-Responses werden überschrieben.

#### Scenario: Bestehende Zusage wird überschrieben
- **WHEN** eine Abwesenheit angelegt wird und der Member eine `confirmed`-Response für ein Event im Zeitraum hat
- **THEN** wird die Response auf `declined` mit gesetzter `absence_id` geändert

#### Scenario: Kein Event im Zeitraum
- **WHEN** eine Abwesenheit angelegt wird und keine Events im Zeitraum liegen
- **THEN** wird die Abwesenheit ohne weitere Änderungen angelegt

### Requirement: Auto-decline bei neuen Events
Wenn eine neue `training_session` oder ein neues `game` angelegt wird, SHALL das System für alle Kader-Members mit einer Abwesenheit, die das Event-Datum überdeckt, sofort eine auto-declined Response anlegen.

#### Scenario: Training in Abwesenheitszeitraum angelegt
- **WHEN** ein Trainer ein Training anlegt, dessen Datum in der Abwesenheit eines Kader-Members liegt
- **THEN** erhält dieser Member sofort eine `declined`-Response mit gesetzter `absence_id`

### Requirement: Auto-declined Responses sind gesperrt
Eine Response mit gesetzter `absence_id` DARF von keiner Rolle (einschließlich Trainer und Admin) manuell geändert werden. Der Nutzer MUSS die Abwesenheit löschen, um wieder zusagen zu können.

#### Scenario: Manuelles Ändern einer auto-declined Response abgewiesen
- **WHEN** ein Nutzer versucht, eine Response mit `absence_id IS NOT NULL` zu ändern
- **THEN** antwortet die API mit HTTP 403

### Requirement: Kalender-Abwesenheits-Endpunkt
Das System SHALL via `GET /api/absences/calendar?from=&to=` die Abwesenheiten zurückgeben, die der eingeloggte Nutzer im Kalender sehen darf: eigene + Kinder immer; Abwesenheiten anderer Members nur wenn deren `absences_public = 1`.

#### Scenario: Trainer sieht nur öffentliche Abwesenheiten
- **WHEN** ein Trainer `GET /api/absences/calendar` aufruft
- **THEN** erhält er nur Abwesenheiten von Members mit `absences_public = 1`

#### Scenario: Spieler sieht eigene Abwesenheiten immer
- **WHEN** ein Spieler `GET /api/absences/calendar` aufruft
- **THEN** erhält er seine eigenen Abwesenheiten unabhängig von `absences_public`

### Requirement: Sichtbarkeits-Toggle im Profil
Ein Member SHALL via `PUT /api/profile/absence-visibility` steuern können, ob seine Abwesenheiten für Trainer im Kalender sichtbar sind (`absences_public`). Default ist `false`.

#### Scenario: Sichtbarkeit aktivieren
- **WHEN** ein Spieler `PUT /api/profile/absence-visibility` mit `{"public": true}` aufruft
- **THEN** wird `members.absences_public` auf `1` gesetzt

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

---

### Requirement: Profil-Toggle „Abwesenheiten für Trainer sichtbar" liest und speichert korrekt
Das System SHALL den Wert von `absences_public` aus der Datenbank korrekt über `GET /api/profile/me` zurückgeben, sodass der Toggle in `ProfileMiscTab` den gespeicherten Zustand anzeigt. `PUT /api/profile/absence-visibility` speichert den Wert weiterhin korrekt.

#### Scenario: Toggle zeigt gespeicherten Wert
- **WHEN** ein Nutzer `absences_public = 1` gesetzt hat und `GET /api/profile/me` aufruft
- **THEN** enthält `own_member.absences_public` den Wert `1` (oder `true`) und der Toggle wird als aktiv angezeigt

#### Scenario: Toggle zeigt inaktiv nach Deaktivierung
- **WHEN** ein Nutzer `PUT /api/profile/absence-visibility` mit `{"public": false}` aufruft und danach `GET /api/profile/me` aufruft
- **THEN** enthält `own_member.absences_public` den Wert `0` (oder `false`) und der Toggle wird als inaktiv angezeigt
