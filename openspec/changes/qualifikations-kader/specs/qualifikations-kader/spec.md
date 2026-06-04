## ADDED Requirements

### Requirement: Qualifikationskader anlegen
Ein Admin SHALL einen neuen Kader vom Typ `qualification` für eine Altersklasse/Geschlecht innerhalb der aktiven Saison anlegen können. Der Kader wird wie ein regulärer Kader befüllt (Spieler, Trainer).

#### Scenario: Erfolgreicher Quali-Kader anlegen
- **WHEN** Admin sendet `POST /api/admin/kader` mit `type='qualification'`, gültiger `age_class`, `gender` und `season_id`
- **THEN** ein neuer Kader mit `is_active=0` wird angelegt und die ID zurückgegeben

#### Scenario: Doppelter aktiver Quali-Kader wird abgelehnt
- **WHEN** für eine `(season_id, age_class, gender)`-Kombination bereits ein aktiver Qualifikationskader existiert und ein zweiter aktiviert wird
- **THEN** der erste wird automatisch auf `is_active=0` gesetzt (Aktivierungsendpunkt macht dies atomar)

### Requirement: Kader aktivieren
Ein Admin SHALL einen Kader (regulär oder Qualifikation) explizit aktivieren können. Die Aktivierung MUSS atomar alle anderen Kader desselben Typs und derselben `(season_id, age_class, gender)`-Kombination deaktivieren.

#### Scenario: Quali-Kader aktivieren
- **WHEN** Admin sendet `PUT /api/admin/kader/:id/activate` für einen Qualifikationskader
- **THEN** der Kader wird auf `is_active=1` gesetzt; alle anderen Qualifikationskader derselben Altersklasse/Geschlecht/Saison werden auf `is_active=0` gesetzt

#### Scenario: Aktivierung eines regulären Kaders
- **WHEN** Admin sendet `PUT /api/admin/kader/:id/activate` für einen regulären Kader
- **THEN** der Kader wird auf `is_active=1` gesetzt; alle anderen regulären Kader derselben `(season_id, age_class, gender, team_number)` werden deaktiviert

### Requirement: Kader deaktivieren
Ein Admin SHALL einen aktiven Qualifikationskader explizit deaktivieren können, ohne ihn zu löschen. Der inaktive Kader bleibt als historischer Datensatz erhalten.

#### Scenario: Aktiven Quali-Kader deaktivieren
- **WHEN** Admin sendet `PUT /api/admin/kader/:id/deactivate`
- **THEN** der Kader wird auf `is_active=0` gesetzt; kein anderer Kader wird beeinflusst

#### Scenario: Deaktivierter Kader bleibt erhalten
- **WHEN** ein Kader auf `is_active=0` gesetzt wird
- **THEN** alle verknüpften `kader_members`, `kader_trainers` und Spielzuordnungen bleiben unverändert erhalten

### Requirement: Aktive Kader-Auswahl im Admin-UI
Im Saisons-Tab der Admin-Einstellungen (`/admin/einstellungen?tab=saisons`) SHALL pro Altersklasse/Geschlecht angezeigt werden: der aktive reguläre Kader und — optional — ein aktiver Qualifikationskader. Der Admin kann dort einen anderen Kader aktivieren oder einen neuen Qualifikationskader anlegen.

#### Scenario: Saisons-Tab zeigt aktive Kader
- **WHEN** Admin ruft den Saisons-Tab auf
- **THEN** werden pro Altersklasse/Geschlecht-Gruppe der aktive reguläre Kader und (falls vorhanden) der aktive Qualifikationskader angezeigt

#### Scenario: Neuen Qualifikationskader anlegen aus Saisons-Tab
- **WHEN** Admin klickt „Quali-Kader anlegen" für eine Altersklasse/Geschlecht
- **THEN** öffnet sich ein Modal mit Pflichtfeldern Name, Altersklasse, Geschlecht; nach Bestätigung wird der neue Kader angelegt und kann sofort aktiviert werden

### Requirement: Kader-Listing filtert auf aktive Kader
Alle Kader-Listen-Endpunkte und UI-Ansichten SHALL standardmäßig nur `is_active=1`-Kader zurückgeben.

#### Scenario: Standard-Listing zeigt nur aktive Kader
- **WHEN** ein API-Client `GET /api/admin/kader` aufruft (ohne Parameter)
- **THEN** werden ausschließlich Kader mit `is_active=1` zurückgegeben

#### Scenario: Kader-Typ wird in Response mitgeliefert
- **WHEN** ein Kader-Datensatz zurückgegeben wird
- **THEN** enthält die Response die Felder `type` (`regular` | `qualification`) und `is_active`
