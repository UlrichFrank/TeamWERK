## ADDED Requirements

### Requirement: Änderungsantrag-Routen erzwingen Mitglieds-Ownership

Die Self-Service-Routen `GET /api/members/{id}/change-drafts` und `POST /api/members/{id}/change-request` SHALL nur dann Mitgliedsdaten lesen oder schreiben, wenn der Aufrufer eine Beziehung zum Ziel-Mitglied `{id}` hat: Eigentümer (`member.user_id == claims.UserID`), Elternteil des Mitglieds (`family_links`), `admin`, `vorstand` oder `kassierer`. Für alle anderen Aufrufer SHALL der Server mit HTTP 403 antworten, BEVOR Antrags- oder Mitgliedsdaten (insbesondere der `old_value`-Snapshot) gelesen, zurückgegeben oder verändert werden.

| a | v | ve | vb | ka | t | te | s | se | sp | e |
|---|---|---|---|---|---|---|---|---|---|---|
| ✅ | ✅ (alle) | ✅ | ❌ (fremd) | ✅ (alle) | ❌ (fremd) | ❌ (fremd) | ❌ (fremd) | ❌ (fremd) | ❌ (fremd) | ✅ (eigenes Kind) |

> „fremd" = Member-ID gehört nicht zum Aufrufer/Kind. Eigentümer und Eltern erreichen ausschließlich das eigene bzw. das Kind-Mitglied; `vorstand`/`kassierer`/`admin` erreichen alle Mitglieder.

#### Scenario: Fremder Spieler liest Änderungsanträge eines anderen Mitglieds
- **WHEN** Persona `spieler` `GET /api/members/{id}/change-drafts` mit einer Member-ID aufruft, die nicht zu ihrem eigenen Account gehört
- **THEN** antwortet der Server mit 403 und liefert keine Antragsdaten und keinen `old_value`-Snapshot

#### Scenario: Eigentümer liest eigene Änderungsanträge
- **WHEN** der Eigentümer eines Mitglieds `GET /api/members/{id}/change-drafts` für die eigene Member-ID aufruft
- **THEN** antwortet der Server mit 200 und den eigenen Anträgen

#### Scenario: Elternteil liest Anträge des eigenen Kindes
- **WHEN** Persona `elternteil` `GET /api/members/{id}/change-drafts` für die Member-ID des eigenen Kindes aufruft
- **THEN** antwortet der Server NICHT mit 403

#### Scenario: Vorstand liest fremde Anträge
- **WHEN** Persona `vorstand` `GET /api/members/{id}/change-drafts` für ein beliebiges Mitglied aufruft
- **THEN** antwortet der Server mit 200

#### Scenario: Fremder Nutzer legt Änderungsantrag für anderes Mitglied an
- **WHEN** Persona `spieler` `POST /api/members/{id}/change-request` mit einer fremden Member-ID aufruft
- **THEN** antwortet der Server mit 403 und es wird kein Draft erzeugt, aktualisiert oder verdrängt

---

### Requirement: Bankdaten-Anträge nur durch Eigentümer oder Elternteil

Für `field_name='bankdaten'` SHALL `POST /api/members/{id}/change-request` ausschließlich Aufrufer akzeptieren, die Eigentümer oder Elternteil des Mitglieds sind (Selbstbedienungsmodell). Andere Aufrufer — auch `vorstand` und `kassierer`, deren Rolle die Genehmigung, nicht die Einreichung ist — SHALL mit HTTP 403 antworten. Dadurch kann kein fremder Aufrufer einen verschlüsselten Bankdaten-Envelope unter dem Namen eines anderen Mitglieds hinterlegen.

#### Scenario: Fremder unterschiebt Bankdaten-Envelope
- **WHEN** Persona `spieler` `POST /api/members/{id}/change-request` mit `field_name='bankdaten'` und einem Envelope `{bank_ciphertext, bank_dek_enc}` für eine fremde Member-ID sendet
- **THEN** antwortet der Server mit 403 und es wird kein `bankdaten`-Draft angelegt oder überschrieben

#### Scenario: Eigentümer reicht eigene Bankdaten ein
- **WHEN** der Eigentümer eines Mitglieds `POST .../change-request` mit `field_name='bankdaten'` und gültigem Envelope für die eigene Member-ID sendet
- **THEN** antwortet der Server mit 2xx und der `bankdaten`-Draft wird angelegt oder aktualisiert (UPSERT-Verhalten unverändert)

#### Scenario: Kassierer kann keinen Bankdaten-Antrag für ein Mitglied einreichen
- **WHEN** Persona `kassierer` `POST .../change-request` mit `field_name='bankdaten'` für ein fremdes Mitglied sendet
- **THEN** antwortet der Server mit 403 (Korrektur erfolgt über `PUT /api/members/{id}/bank-details`, nicht über den Antragsweg)
