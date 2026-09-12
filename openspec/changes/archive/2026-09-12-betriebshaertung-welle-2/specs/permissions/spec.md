## ADDED Requirements

### Requirement: Objektrechte werden mechanisch geprüft

Für jede Route mit einem Objekt-Parameter (`{id}`) MUST ein Test existieren, der ein fremdes Objekt anlegt und als nicht berechtigter Nutzer 403 oder 404 erwartet. Routen ohne einen solchen Test MUST den Test fehlschlagen lassen, sofern sie nicht mit Begründung in einer Allowlist bewusst offener Routen stehen. Ein verwaister Allowlist-Eintrag MUST ebenfalls fehlschlagen.

#### Scenario: Neue Objekt-Route ohne Fixture
- **WHEN** eine Route `GET /api/foo/{id}` hinzukommt, ohne Fixture-Erzeuger und ohne Allowlist-Eintrag
- **THEN** schlägt die Objekt-Matrix fehl und nennt die Route

#### Scenario: Fremdes Objekt bleibt verborgen
- **WHEN** Nutzer A ein Objekt von Nutzer B über eine `{id}`-Route anspricht
- **THEN** antwortet der Server mit 403 oder 404
