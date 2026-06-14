## MODIFIED Requirements

### Requirement: Abwesenheiten im Kalender farblich nach Herkunft unterschieden
Der Kalender SHALL eigene Abwesenheiten (`is_own: true`) in `brand-yellow` und Team-Abwesenheiten (`is_own: false`) in `brand-blue` darstellen.

#### Scenario: Eigene Abwesenheit bleibt gelb
- **WHEN** eine Abwesenheit `is_own = true` hat
- **THEN** wird sie mit `bg-brand-yellow/20 border-brand-yellow/60` dargestellt (unverändert)

#### Scenario: Team-Abwesenheit erscheint blau
- **WHEN** eine Abwesenheit `is_own = false` hat
- **THEN** wird sie mit `bg-brand-blue/20 border-brand-blue/60` dargestellt

---

### Requirement: Personendetails für Team-Abwesenheiten nur per Tooltip und Click
Team-Abwesenheiten (`is_own: false`) SHALL den Namen des Mitglieds und den Typ im `title`-Attribut (Tooltip) anzeigen. Per Click öffnet sich die vorhandene Detailansicht (InfoModal).

#### Scenario: Tooltip zeigt Name und Typ
- **WHEN** ein Nutzer über eine Team-Abwesenheit hovert
- **THEN** zeigt der Browser-Tooltip: `{member_name}: {type} {start_date}–{end_date}`

#### Scenario: Click öffnet Detailansicht
- **WHEN** ein Nutzer auf eine Team-Abwesenheit klickt
- **THEN** öffnet sich das Info-Modal mit den Details der Abwesenheit (Typ, Zeitraum, ggf. Notiz)
