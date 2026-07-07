## ADDED Requirements

### Requirement: HTTP-Caching für Immutable-Referenzrouten

Das System SHALL für nicht-geheime, quasi-unveränderliche Referenzrouten (`GET /api/encryption-pubkey`, `GET /api/push/vapid-public-key`) einen `ETag` sowie einen `Cache-Control`-Header mit langer `max-age` setzen und `If-None-Match`-Requests mit `304 Not Modified` (leerer Body) beantworten.

#### Scenario: Erneuter Abruf des Public Keys liefert 304

- **WHEN** ein Client `GET /api/encryption-pubkey` mit `If-None-Match` gleich dem zuvor gelieferten `ETag` aufruft
- **THEN** antwortet das System mit `304 Not Modified` und leerem Body

#### Scenario: VAPID-Key ist als immutable markiert

- **WHEN** ein Client `GET /api/push/vapid-public-key` aufruft
- **THEN** trägt die Antwort einen `Cache-Control`-Header mit `immutable`
- **AND** der Body enthält den konfigurierten VAPID-Public-Key

### Requirement: ETag-Revalidierung für Referenzdaten

Das System SHALL für Referenzrouten mit potenziell nutzergefilterten Daten (`GET /api/seasons`, `GET /api/teams`, `GET /api/venues`, `GET /api/age-class-rules`, `GET /api/duty-types`) einen schwachen `ETag` aus einem günstigen Content-Fingerprint (z. B. `COUNT` + `MAX(updated_at)`) setzen, `Cache-Control: private, no-cache` verwenden und `If-None-Match`-Requests mit `304` beantworten. Das System SHALL für diese Routen KEINEN geteilten (`public`) `max-age` setzen.

#### Scenario: Unveränderte Referenzdaten werden revalidiert

- **WHEN** ein Client `GET /api/seasons` mit dem `If-None-Match` eines noch gültigen `ETag` aufruft und sich der Saison-Bestand seither nicht geändert hat
- **THEN** antwortet das System mit `304 Not Modified`

#### Scenario: ETag ändert sich nach Mutation

- **WHEN** eine Saison angelegt, geändert oder gelöscht wurde
- **THEN** liefert der nächste `GET /api/seasons` einen anderen `ETag` als zuvor
- **AND** die vollständige Antwort (nicht `304`) wird ausgeliefert

#### Scenario: Nutzergefilterte Route bleibt privat

- **WHEN** das System eine Antwort für `GET /api/teams` ausliefert
- **THEN** enthält der `Cache-Control`-Header KEIN `public`
- **AND** eine für Nutzer A gecachte Antwort wird niemals an Nutzer B ausgeliefert

### Requirement: Clientseitiger TTL-Cache mit Single-Flight

Das Frontend SHALL Referenz-Endpunkte über einen In-Memory-Cache mit routen-spezifischer TTL bedienen, gleichzeitige In-Flight-Requests derselben URL zu einem Request zusammenfassen (Single-Flight) und den Cache bei passenden Live-Update-Events invalidieren. Der Cache SHALL nur für eine explizite Allowlist von Referenzrouten greifen; alle übrigen API-Calls bleiben unverändert.

#### Scenario: Zweiter Abruf innerhalb der TTL trifft den Cache

- **WHEN** zwei Komponenten dieselbe Referenzroute innerhalb der TTL abrufen
- **THEN** wird höchstens ein HTTP-Request ausgelöst
- **AND** beide Komponenten erhalten dieselben Daten

#### Scenario: Live-Update invalidiert den Cache

- **WHEN** ein Live-Update-Event eintrifft, das eine gecachte Referenzroute betrifft (z. B. `seasons`)
- **THEN** wird der zugehörige Cache-Eintrag verworfen
- **AND** der nächste Abruf lädt frisch

### Requirement: Service-Worker-Strategie für Referenzdaten

Der Service Worker SHALL Referenz-Endpunkte mit `StaleWhileRevalidate` bedienen und für die generische `api-cache` eine Obergrenze (`maxEntries` und `maxAgeSeconds`) setzen, damit der Cache nicht unbegrenzt wächst.

#### Scenario: Referenzdaten werden sofort aus dem Cache bedient und im Hintergrund erneuert

- **WHEN** eine Referenzroute erneut angefragt wird und ein Cache-Eintrag existiert
- **THEN** liefert der Service Worker den gecachten Stand sofort aus
- **AND** aktualisiert den Cache im Hintergrund aus dem Netz

#### Scenario: api-cache ist begrenzt

- **WHEN** die Zahl der Einträge in `api-cache` die konfigurierte Obergrenze überschreitet
- **THEN** werden die ältesten Einträge verworfen
