## ADDED Requirements

### Requirement: SEPA-Stammdaten des Vereins
Die Vereins-Konfiguration MUST die SEPA-Gläubigerdaten `glaeubiger_id`, `iban`, `bic` und `kontoinhaber` führen. `GET /api/club` MUST diese Felder zurückgeben; `PUT /api/club` MUST sie setzen können. Beim Setzen MUST `glaeubiger_id` (Format `DE\d{2}[A-Z0-9]{3}\d{11}`), `iban` (Mod-97-Prüfsumme) und `bic` (8 oder 11 Zeichen) validiert werden. Ungültige Werte MÜSSEN mit HTTP 400 abgelehnt werden.

#### Scenario: Stammdaten setzen und lesen
- **WHEN** ein Vorstand `PUT /api/club` mit gültiger Gläubiger-ID, IBAN, BIC und Kontoinhaber aufruft und anschließend `GET /api/club`
- **THEN** liefert die Antwort die zuvor gesetzten vier SEPA-Felder zurück

#### Scenario: Ungültige IBAN
- **WHEN** ein `PUT /api/club` mit einer IBAN mit falscher Mod-97-Prüfsumme erfolgt
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Ungültige Gläubiger-ID
- **WHEN** ein `PUT /api/club` mit einer Gläubiger-ID erfolgt, die dem Format nicht entspricht
- **THEN** antwortet der Server mit HTTP 400
