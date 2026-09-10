## ADDED Requirements

### Requirement: Dienst-Rangliste im Authenticated-Tier mit Handler-Scope

`GET /api/duty-fairness/rangliste` SHALL im Authenticated-Tier liegen: jede
eingeloggte Persona erreicht die Route, ohne Token antwortet sie mit HTTP 401. Die
Einschränkung auf eigene Teams (HTTP 403 für ein Team ohne eigene Kader- oder
`family_links`-Verbindung) und die Namens-Maskierung SHALL der Handler selbst
durchsetzen — für die System-Rolle `admin` und die Vereinsfunktion `vorstand`
entfällt beides.

Der Sidebar-Eintrag „Dienst-Rangliste" (`/dienste/rangliste`) SHALL für jede
eingeloggte Persona sichtbar sein.

#### Scenario: Eingeloggte Persona erreicht die Rangliste
- **WHEN** eine beliebige Persona `GET /api/duty-fairness/rangliste` mit gültigem
  Token aufruft
- **THEN** antwortet das System nicht mit 401

#### Scenario: Aufruf ohne Token
- **WHEN** `GET /api/duty-fairness/rangliste` ohne Token aufgerufen wird
- **THEN** antwortet das System mit HTTP 401

#### Scenario: Sidebar zeigt die Rangliste für alle
- **WHEN** eine beliebige eingeloggte Persona die Navigation lädt
- **THEN** enthält sie den Eintrag „Dienst-Rangliste" mit Ziel `/dienste/rangliste`
