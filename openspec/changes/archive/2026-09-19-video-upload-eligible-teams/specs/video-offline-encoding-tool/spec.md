## ADDED Requirements

### Requirement: Auswahl folgt der Upload-Berechtigung

Das Tool SHALL die Mannschafts- und die Spielauswahl ausschließlich aus dem Berechtigungs-Endpoint für eigene upload-berechtigte Spiele ableiten und dafür weder die allgemeine Mannschaftsliste noch die allgemeine Terminliste des Servers abfragen. Jede Mannschaft, für die der Server einen Upload zulässt, MUST wählbar sein — auch wenn der Nutzer mit ihr nur über eine Video-Dienst-Zuweisung verbunden ist. Die Spielliste einer Mannschaft SHALL nur bereits gespielte Spiele der aktiven Saison enthalten, jüngstes zuerst. Der Upload ohne Spielbezug („Freier Titel") MUST nur für Mannschaften angeboten werden, für die der Server ihn zulässt; für alle anderen Mannschaften MUST ein Spiel gewählt werden.

#### Scenario: Mannschaft nur über Video-Dienst erreichbar
- **WHEN** ein angemeldeter Nutzer weder Trainer, Spieler noch Elternteil bei Mannschaft A ist, aber einen Video-Dienst für ein vergangenes Spiel von Mannschaft A hat
- **THEN** bietet das Tool Mannschaft A zur Auswahl an und listet unter ihr genau dieses Spiel, ohne die Option „Freier Titel"

#### Scenario: Trainer der eigenen Mannschaft
- **WHEN** ein Trainer seine eigene Mannschaft wählt
- **THEN** listet das Tool alle vergangenen Spiele dieser Mannschaft in der aktiven Saison und bietet zusätzlich „Freier Titel" an

#### Scenario: Keine Berechtigung
- **WHEN** der Server für den angemeldeten Nutzer keine Mannschaft zurückliefert
- **THEN** bleibt die Mannschaftsauswahl leer und das Tool zeigt einen Hinweis, dass für dieses Konto weder eine Trainer-/Vorstands-Berechtigung noch ein Video-Dienst hinterlegt ist

#### Scenario: Mannschaft mit Dienst, aber ohne vergangenes Spiel
- **WHEN** die einzige Dienst-Zuweisung des Nutzers ein zukünftiges Spiel betrifft
- **THEN** ist die Mannschaft wählbar, die Spielliste bleibt leer und das Tool weist darauf hin, dass ein Upload erst nach dem Spiel möglich ist
