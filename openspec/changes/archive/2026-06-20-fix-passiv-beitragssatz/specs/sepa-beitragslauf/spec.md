## MODIFIED Requirements

### Requirement: Beitragsberechnung pro Mitglied
Der Beitragslauf MUST für jedes Mitglied anhand von `members.status` die Beitragsgruppe und den fälligen Jahresbeitrag zum **Stichtag 01.07. der Saison** bestimmen. Status `aktiv`/`verletzt` → Gruppe `aktiv` (Kategorie `aktiv_mit` bzw. `aktiv_ohne` je nach Stammverein); Status `pausiert`/`passiv` → Kategorie `passiv`. Für jede einzuziehende Kategorie MUST zum Stichtag ein gültiger Beitragssatz (`valid_from <= Stichtag`) existieren; fehlt er, wird das Mitglied mit Begründung `kein_beitragssatz` ausgeschlossen.

Die Beitragsmatrix MUST für die Kategorie `passiv` ab dem frühestmöglichen Saisonstart (01.07.2026) einen gültigen Satz enthalten, damit passive Mitglieder in der laufenden Saison erkannt und nicht fälschlich ausgeschlossen werden.

#### Scenario: Passives Mitglied in Saison 2026/27 wird einbezogen
- **WHEN** der Beitragslauf für eine Saison mit Start `2026-07-01` ausgeführt wird und ein Mitglied `status='passiv'` mit gültigem SEPA-Mandat, IBAN und vollständiger Adresse hat
- **THEN** wird das Mitglied mit Kategorie `passiv` und Betrag 6000 ct (60 €) einbezogen und **nicht** mit `kein_beitragssatz` ausgeschlossen

#### Scenario: Pausiertes Mitglied zählt als passiv
- **WHEN** ein Mitglied `status='pausiert'` im Lauf für Saison 2026/27 verarbeitet wird
- **THEN** wird es der Kategorie `passiv` zugeordnet und mit dem ab `2026-07-01` gültigen Passiv-Satz berechnet
