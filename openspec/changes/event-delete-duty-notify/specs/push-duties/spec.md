## ADDED Requirements

### Requirement: Notification an Dienst-Zugewiesene bei Event-Löschung

Beim Löschen eines Spiels oder generischen Ereignisses (`DELETE /api/kalender/{id}`) SHALL das System alle Nutzer benachrichtigen, die einen `duty_assignment` für einen Slot des betroffenen Events hatten — unabhängig vom Assignment-Status (`pending` oder `fulfilled`). Die Benachrichtigung erfolgt über die `notifications.Send`-Fassade in der Kategorie `duties`, sodass Push- und Email-Präferenzen pro Nutzer respektiert werden.

#### Scenario: Spiel mit zugewiesenen Diensten wird gelöscht

- **WHEN** ein Trainer ein Spiel mit drei Diensten löscht, von denen zwei zugesagt (`pending`) und einer erbracht (`fulfilled`) sind
- **THEN** erhalten alle drei Dienst-Zugewiesenen eine Notification mit dem Titel „Dienst entfällt" und dem Body „Dein Dienst zum {Gegnername} am {Datum} wurde gelöscht."
- **THEN** wird der Link „/dienste" mitgegeben

#### Scenario: Generisches Event mit Dienst wird gelöscht

- **WHEN** ein Trainer ein generisches Event (z.B. „Vereinsfest") mit Diensten löscht
- **THEN** erhalten die Zugewiesenen die Notification mit dem Event-Namen im Body („Dein Dienst zum Vereinsfest am 14.06. wurde gelöscht.")

#### Scenario: Event ohne Dienste wird gelöscht

- **WHEN** ein Trainer ein Event ohne zugewiesene Dienste löscht
- **THEN** wird keine `duties`-Notification verschickt
- **WHEN** das Event ein Spiel ist
- **THEN** wird trotzdem die bestehende `games`-Notification „Spiel abgesagt" an die Team-Responder verschickt

#### Scenario: Nutzer hat Email aktiv für Dienste

- **WHEN** ein Dienst-Zugewiesener `email_enabled=1` für `duties` hat und sein Event gelöscht wird
- **THEN** erhält der Nutzer eine Email mit dem persönlich formulierten Body und dem Direktlink

### Requirement: Konto-Rekomputation bei Cascade-Delete

Wenn beim Event-Delete ein Assignment mit Status `fulfilled` mitgelöscht wird, SHALL das System `duty_accounts.ist` für jeden betroffenen Nutzer und die Event-Saison neu aus den verbliebenen `fulfilled`-Assignments aggregieren. Die Rekomputation MUSS in derselben Transaction wie das Event-Delete erfolgen.

#### Scenario: Fulfilled-Dienst wird per Event-Delete entfernt

- **WHEN** ein Spiel mit einem `fulfilled`-Dienst (2h-`duty_type`) gelöscht wird
- **AND** der Nutzer bisher `duty_accounts.ist = 5h` für die Saison hatte
- **THEN** ist nach dem Delete `duty_accounts.ist = 3h` für (Nutzer, Saison)

#### Scenario: Pending-Dienst wird per Event-Delete entfernt

- **WHEN** ein Event mit ausschließlich `pending`-Diensten gelöscht wird
- **THEN** bleiben alle `duty_accounts.ist`-Werte der Zugewiesenen unverändert

#### Scenario: Rekomputation läuft transaktional

- **WHEN** während der Konto-Rekomputation ein DB-Fehler auftritt
- **THEN** wird die gesamte Transaktion zurückgerollt — Event und Dienste bleiben erhalten, Konten unverändert
