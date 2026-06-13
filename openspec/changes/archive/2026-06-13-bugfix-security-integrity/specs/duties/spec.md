## MODIFIED Requirements

### Requirement: Duty board (Dienstbörse)
Das System SHALL eine Dienstbörse mit allen Duty-Slots anzeigen. Das Beanspruchen eines Slots MUSS race-frei implementiert sein: die Prüfung auf verfügbare Kapazität, das Eintragen des Nutzers und das Aktualisieren des Zählers MÜSSEN als eine atomare Operation erfolgen, die auch bei gleichzeitigen Anfragen korrekt funktioniert.

#### Scenario: Claim a duty slot — kein Familienmitglied vorhanden
- **WHEN** a user without linked children with proxy accounts claims an open slot
- **THEN** the system records the assignment directly for that user, decrements the vacancy count, updates the claimant's duty account, and the claimant's name appears in the assignee list

#### Scenario: Claim a duty slot — Elternteil mit Kind-Auswahl
- **WHEN** ein Elternteil mit mindestens einem verknüpften Kind mit Proxy-Account auf „Eintragen" klickt
- **THEN** erscheint ein „Für wen?"-Dialog mit dem eigenen Namen als Default und je einem Eintrag pro Kind mit Proxy-Account

#### Scenario: Concurrent claim — letzter freier Slot
- **WHEN** zwei Nutzer gleichzeitig den letzten freien Slot beanspruchen
- **THEN** gelingt genau einem der Claim (HTTP 204), der andere erhält HTTP 409
- **THEN** ist `slots_filled` danach exakt gleich `slots_total` (kein Überlauf)

#### Scenario: Slot fully filled
- **WHEN** the last vacancy of a slot is claimed
- **THEN** the slot no longer shows vacancies but the assignee names remain visible

#### Scenario: Cannot claim already-assigned slot
- **WHEN** a user attempts to claim a slot they or their family already hold
- **THEN** the system returns a validation error

#### Scenario: Unclaim — atomare Gegenbuchung
- **WHEN** ein Nutzer seinen Dienst-Claim aufhebt
- **THEN** wird die `duty_assignments`-Zeile gelöscht UND `slots_filled` dekrementiert in einer einzigen Transaktion
- **THEN** bei einem Datenbankfehler während der Transaktion bleibt der Zähler konsistent (kein partieller State)
