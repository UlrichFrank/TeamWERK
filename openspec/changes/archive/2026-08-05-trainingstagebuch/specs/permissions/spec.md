## ADDED Requirements

### Requirement: Trainingstagebuch-Schreibzugriff ist auf den Eigentümer beschränkt

Die Routen `POST /api/training-diary`, `PUT /api/training-diary/{id}`,
`DELETE /api/training-diary/{id}`, `POST /api/training-diary/{id}/proof` und
`DELETE /api/training-diary/{id}/proof` SHALL ausschließlich durch das Mitglied ausgeführt werden
können, dem der Eintrag gehört (`member.user_id == claims.UserID`). Keine Vereinsfunktion und
keine System-Rolle — auch nicht `admin`, `sportliche_leitung` oder der Trainer des Kaders —
verschafft Schreibzugriff auf ein fremdes Tagebuch.

`GET /api/training-diary` und `POST /api/training-diary` setzen zusätzlich voraus, dass der
Aufrufer überhaupt einen Mitglieds-Datensatz besitzt; Nutzer ohne verknüpftes Mitglied (etwa reine
Elternkonten) erhalten HTTP 403.

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ❌ (fremd) | ❌ | ❌ | ❌ | ❌ | ❌ (fremd) | ❌ | ❌ (fremd) | ❌ | ✅ (eigenes) | ❌ |

> Schreiben ist ausschließlich an die Eigentümerschaft gebunden. Eltern dürfen das Tagebuch ihres
> Kindes **lesen**, aber nicht befüllen — die Erfassung ist die Selbstauskunft des Spielers.

#### Scenario: Trainer ändert den Eintrag eines Spielers
- **WHEN** ein Trainer des Kaders `PUT /api/training-diary/{id}` auf den Eintrag eines seiner
  Spieler aufruft
- **THEN** antwortet der Server mit 403 und der Eintrag bleibt unverändert

#### Scenario: Trainer lädt einen Nachweis für einen Spieler hoch
- **WHEN** ein Trainer des Kaders `POST /api/training-diary/{id}/proof` auf einen fremden Eintrag
  aufruft
- **THEN** antwortet der Server mit 403 und es wird keine Datei geschrieben

#### Scenario: Nutzer ohne Mitglieds-Datensatz erfasst eine Einheit
- **WHEN** ein eingeloggter Nutzer ohne verknüpftes Mitglied `POST /api/training-diary` aufruft
- **THEN** antwortet der Server mit 403 und es wird kein Eintrag angelegt

#### Scenario: Unbekannte Eintrags-ID ist nicht per Statuscode enumerierbar
- **WHEN** ein beliebiger eingeloggter Nutzer `PUT /api/training-diary/{id}` mit einer nicht
  existierenden ID aufruft
- **THEN** antwortet der Server mit 404 und nicht mit 403

---

### Requirement: Trainingstagebuch-Lesezugriff folgt der Anwesenheitsstatistik

Die Routen `GET /api/members/{id}/training-diary`, `GET /api/training-diary/{id}/proof` und
`GET /api/teams/{id}/training-diary-stats` SHALL Lesezugriff ausschließlich gewähren an: das
Mitglied selbst, ein Elternteil über `family_links`, einen Trainer, der über
`trainer_memberships` × `kader` in der **aktiven Saison** eine Mannschaft betreut, in deren Stamm-
oder erweitertem Kader das Mitglied steht, sowie `sportliche_leitung` und `admin`.

`vorstand`, `vorstand_beisitzer` und `kassierer` SHALL **keinen** Zugriff erhalten — anders als bei
den Mitglieder- und Änderungsantrags-Routen begründet Vereinsverwaltung hier kein Leserecht. Das
Tagebuch ist persönlich.

Existiert die angefragte Member-ID nicht, SHALL der Server mit HTTP 403 antworten (nicht mit 500
und nicht mit 404) und dadurch nicht preisgeben, ob die ID vergeben ist.

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ❌ | ❌ | ❌ | ❌ | ✅ (eigener Kader) | ✅ (eigener Kader) | ✅ (alle) | ✅ (alle) | ✅ (eigenes) | ✅ (eigenes Kind) |

#### Scenario: Mannschaftskamerad liest ein fremdes Tagebuch
- **WHEN** Persona `spieler` `GET /api/members/{id}/training-diary` mit der Member-ID eines
  Mitspielers aus demselben Kader aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: Mannschaftskamerad ruft einen fremden Nachweis ab
- **WHEN** Persona `spieler` `GET /api/training-diary/{id}/proof` für den Eintrag eines
  Mitspielers aufruft
- **THEN** antwortet der Server mit 403 und liefert keine Bytes

#### Scenario: Vorstand liest ein fremdes Tagebuch
- **WHEN** Persona `vorstand` `GET /api/members/{id}/training-diary` für ein beliebiges Mitglied
  aufruft
- **THEN** antwortet der Server mit 403

#### Scenario: Trainer einer fremden Mannschaft
- **WHEN** ein Trainer `GET /api/teams/{id}/training-diary-stats` für eine Mannschaft aufruft, die
  er nicht betreut
- **THEN** antwortet der Server mit 403

#### Scenario: Kaderwechsel entzieht den Zugriff
- **WHEN** ein Mitglied aus dem Kader eines Trainers entfernt wird
- **THEN** antwortet der Server dem Trainer beim nächsten Abruf mit 403, ohne dass eine Nachpflege
  nötig ist

#### Scenario: Spieler ruft die Mannschaftsübersicht ab
- **WHEN** Persona `spieler` `GET /api/teams/{id}/training-diary-stats` für die eigene Mannschaft
  aufruft
- **THEN** antwortet der Server mit 403

---

### Requirement: Gelöschte Nachweise sind vom Fehlen unterscheidbar

`GET /api/training-diary/{id}/proof` SHALL für einen Berechtigten mit **HTTP 410** antworten, wenn
der Nachweis durch die Retention entfernt wurde (`proof_purged_at` gesetzt), und mit **HTTP 404**,
wenn nie einer hinterlegt war. Die Unterscheidung SHALL erst **nach** der Zugriffsprüfung erfolgen,
damit Unberechtigte daraus nichts über den Zustand des Eintrags ableiten können.

#### Scenario: Berechtigter ruft einen bereinigten Nachweis ab
- **WHEN** der Eigentümer den Nachweis eines Eintrags abruft, dessen `proof_purged_at` gesetzt ist
- **THEN** antwortet der Server mit 410

#### Scenario: Unberechtigter ruft einen bereinigten Nachweis ab
- **WHEN** ein fremder Spieler denselben Endpoint aufruft
- **THEN** antwortet der Server mit 403 und nicht mit 410
