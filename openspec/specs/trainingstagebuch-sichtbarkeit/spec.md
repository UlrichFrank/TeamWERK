# trainingstagebuch-sichtbarkeit Specification

## Purpose

Wer fremde Trainingstagebuch-Einträge lesen darf — Mitglied selbst, Eltern über `family_links`,
Trainer des Kaders in der aktiven Saison, `sportliche_leitung` und `admin` — sowie die
aggregierte Trainer-Übersicht über eine Mannschaft, analog `attendance.canSeeMemberStats`.

## Requirements

### Requirement: Lesezugriff auf fremde Tagebücher ist abschließend geregelt

Das System SHALL Lesezugriff auf die Trainingstagebuch-Einträge eines Mitglieds ausschließlich
folgenden Nutzern gewähren:

- dem Mitglied selbst (über `members.user_id`),
- einem Elternteil des Mitglieds (über `family_links.parent_user_id`),
- einem Nutzer mit der Vereinsfunktion `trainer`, der über `trainer_memberships` × `kader` in der
  **aktiven Saison** Trainer einer Mannschaft ist, in deren Stamm- oder erweitertem Kader das
  Mitglied steht,
- einem Nutzer mit der Vereinsfunktion `sportliche_leitung`,
- einem Nutzer mit der System-Rolle `admin`.

Allen anderen — insbesondere **anderen Spielern**, Trainern fremder Mannschaften sowie Nutzern mit
`vorstand`, `vorstand_beisitzer` oder `kassierer` ohne eine der obigen Eigenschaften — SHALL der
Zugriff mit HTTP 403 verweigert werden.

Diese Prüfung SHALL für `GET /api/members/{id}/training-diary` und für
`GET /api/training-diary/{id}/proof` identisch gelten.

#### Scenario: Spieler liest sein eigenes Tagebuch

- **WHEN** ein Spieler `GET /api/members/{id}/training-diary` mit seiner eigenen `member_id`
  aufruft
- **THEN** antwortet das System mit HTTP 200

#### Scenario: Spieler versucht, ein fremdes Tagebuch zu lesen

- **WHEN** ein Spieler `GET /api/members/{id}/training-diary` mit der `member_id` eines
  Mannschaftskameraden aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Elternteil liest das Tagebuch des Kindes

- **WHEN** ein über `family_links` verknüpfter Elternteil das Tagebuch seines Kindes abruft
- **THEN** antwortet das System mit HTTP 200

#### Scenario: Trainer liest einen Spieler seines Kaders

- **WHEN** ein Trainer einer Mannschaft der aktiven Saison das Tagebuch eines Mitglieds abruft,
  das im Stamm- oder erweiterten Kader dieser Mannschaft steht
- **THEN** antwortet das System mit HTTP 200

#### Scenario: Trainer einer fremden Mannschaft

- **WHEN** ein Trainer das Tagebuch eines Mitglieds abruft, das in keinem seiner Kader steht
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Sportliche Leitung liest vereinsweit

- **WHEN** ein Nutzer mit `sportliche_leitung` ein beliebiges Tagebuch abruft
- **THEN** antwortet das System mit HTTP 200

#### Scenario: Vorstand ohne weitere Funktion

- **WHEN** ein Nutzer mit ausschließlich der Vereinsfunktion `vorstand` ein fremdes Tagebuch abruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Kaderwechsel entzieht den Zugriff sofort

- **WHEN** ein Mitglied aus dem Kader eines Trainers entfernt wird
- **THEN** antwortet das System dem Trainer beim nächsten Abruf des Tagebuchs mit HTTP 403,
  ohne dass eine Nachpflege nötig ist

---

### Requirement: Trainer erhalten eine aggregierte Mannschaftsübersicht

Das System SHALL über `GET /api/teams/{id}/training-diary-stats?season=<id>` je Kadermitglied der
Mannschaft die Anzahl der Einheiten, die Summe der Minuten und den auf eine Nachkommastelle
gerundeten Durchschnitts-RPE im Saisonzeitraum liefern. Mitglieder ohne Einträge SHALL mit
Nullwerten enthalten sein, damit die Übersicht vollständig ist.

Zugriff haben Trainer der Mannschaft in der aktiven Saison, `sportliche_leitung` und `admin`;
alle anderen erhalten HTTP 403.

Ohne aktive Saison und ohne `?season=`-Parameter SHALL das System eine leere Liste liefern statt
eines Fehlers.

#### Scenario: Trainer ruft die Übersicht der eigenen Mannschaft ab

- **WHEN** ein Trainer `GET /api/teams/{id}/training-diary-stats` für seine Mannschaft aufruft
- **THEN** antwortet das System mit HTTP 200
- **THEN** enthält die Antwort für jedes Kadermitglied Einheiten, Minuten und Durchschnitts-RPE

#### Scenario: Mitglied ohne Einträge erscheint mit Nullwerten

- **WHEN** ein Kadermitglied im Saisonzeitraum keinen Eintrag erfasst hat
- **THEN** ist es in der Antwort mit `entries=0` und `minutes=0` enthalten

#### Scenario: Spieler ruft die Team-Übersicht ab

- **WHEN** ein Spieler der Mannschaft `GET /api/teams/{id}/training-diary-stats` aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Trainer ruft eine fremde Mannschaft ab

- **WHEN** ein Trainer die Übersicht einer Mannschaft abruft, die er nicht betreut
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Keine aktive Saison

- **WHEN** die Übersicht ohne aktive Saison und ohne `?season=` abgerufen wird
- **THEN** antwortet das System mit HTTP 200 und einer leeren Liste

---

### Requirement: Die Trainer-Sicht erlaubt den Absprung in die Einzeleinträge

Das Frontend SHALL die Mannschaftsübersicht nach dem Muster der Anwesenheitsstatistik aufbauen:
eine Liste je Mitglied mit Kennzahlen, aus der ein Klick die Einzeleinträge dieses Mitglieds
nachlädt (`GET /api/members/{id}/training-diary`) — inklusive Art, Dauer, RPE, Notiz und Nachweis.

Die Übersicht SHALL erkennbar machen, dass sie **Erfassungsdisziplin** misst und nicht
Trainingsfleiß: ein Mitglied ohne Einträge kann nicht trainiert oder nur nichts eingetragen haben.

#### Scenario: Drill-down in ein Mitglied

- **WHEN** ein Trainer in der Übersicht auf eine Zeile klickt
- **THEN** werden die Einzeleinträge dieses Mitglieds nachgeladen und angezeigt

#### Scenario: Hinweis auf die Aussagekraft

- **WHEN** die Mannschaftsübersicht angezeigt wird
- **THEN** weist ein Hinweistext darauf hin, dass die Zahlen auf Selbstauskunft beruhen
