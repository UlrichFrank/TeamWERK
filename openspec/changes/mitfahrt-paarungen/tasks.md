## 1. Datenbank-Migration

- [x] 1.1 Migration `013_mitfahrt_paarungen.up.sql` schreiben: `mitfahrgelegenheiten`-Tabelle neu erstellen ohne `UNIQUE(game_id, user_id)`, Daten kopieren, alte Tabelle droppen, partiellen Index `CREATE UNIQUE INDEX idx_mitfahr_biete_unique ON mitfahrgelegenheiten(game_id, user_id) WHERE typ = 'biete'` anlegen
- [x] 1.2 Neue Tabelle `mitfahrt_paarungen` in der Migration anlegen (Felder: id, biete_id FK, suche_id FK, initiiert_von, status, created_at, updated_at, UNIQUE(biete_id, suche_id))
- [x] 1.3 Down-Migration `013_mitfahrt_paarungen.down.sql` schreiben (partiellen Index droppen, Tabelle zurückbauen mit altem UNIQUE-Constraint, `mitfahrt_paarungen` droppen)

## 2. Backend — Bestehende Handler anpassen

- [x] 2.1 `Upsert`-Handler: Upsert-Logik nur noch für `typ='biete'` beibehalten; für `typ='suche'` stattdessen immer einen neuen Eintrag anlegen (INSERT ohne ON CONFLICT)
- [x] 2.2 `Upsert`-Handler: Validierung `plaetze ≥ 1` für `typ='suche'` (400 Bad Request wenn fehlt)
- [x] 2.3 `Delete`-Handler: beim Löschen eines Bieter-Eintrags alle zugehörigen `pending`/`confirmed` Paarungen kaskadieren (via ON DELETE CASCADE) und betroffene Sucher per Push benachrichtigen
- [x] 2.4 `Delete`-Handler: beim Löschen eines Suche-Eintrags Bieter der betroffenen `confirmed`-Paarung per Push benachrichtigen
- [x] 2.5 `List`-Handler: `paarungen`-Array pro Spiel in die API-Antwort aufnehmen (Bieter-Name, Sucher-Name, suche.plaetze als Anzahl, Status; rejected ausblenden)

## 3. Backend — Neue Paarungs-Endpunkte

- [x] 3.1 `POST /api/mitfahrt-paarungen` implementieren: Paarungsanfrage anlegen (Bieter oder Sucher initiiert); Kapazitätsprüfung (pending + confirmed gegen biete.plaetze); Duplikat-Check; Push an Gegenseite
- [x] 3.2 `POST /api/mitfahrt-paarungen/{id}/confirm` implementieren: Gegenseite bestätigt; Kapazitätsprüfung als Absicherung gegen Race Condition; Push an Initiator
- [x] 3.3 `POST /api/mitfahrt-paarungen/{id}/reject` implementieren: Anfrage ablehnen oder bestätigte Paarung stornieren; Push an Gegenseite; 403 wenn fremde Paarung
- [x] 3.4 Neue Routen in `cmd/teamwerk/main.go` registrieren (authenticated-Gruppe)

## 4. Backend — Push-Benachrichtigungen

- [x] 4.1 Push bei neuer Anfrage (pending): Gegenseite benachrichtigen mit Sender-Name und Spiel-Datum
- [x] 4.2 Push bei Bestätigung: Initiator benachrichtigen
- [x] 4.3 Push bei Ablehnung/Stornierung: Gegenseite benachrichtigen, Text klar zwischen "Anfrage abgelehnt" und "Bestätigung storniert" unterscheiden
- [x] 4.4 Kapazitätsanzeige im Board: Bieter-Eintrag zeigt freie Plätze (biete.plaetze minus pending+confirmed) damit Sucher vor dem Anfragen sehen ob noch Platz ist

## 5. Frontend — Formular anpassen

- [x] 5.1 `FormModal`: für `typ='suche'` Pflichtfeld „Anzahl Personen" (`plaetze ≥ 1`) hinzufügen
- [x] 5.2 `FormModal`: für `typ='suche'` Upsert-Logik entfernen — jede Einreichung ist ein neuer Eintrag; eigene Suche-Einträge weiterhin löschbar

## 6. Frontend — Paarungs-UI in GameCard

- [x] 6.1 API-Typen erweitern: `CarpoolEntry` um Paarungs-Felder ergänzen (`paarungId?`, `paarungStatus?`); neue `PaaringEntry`-Schnittstelle für das `paarungen`-Array
- [x] 6.2 `EntryCard` (Bieter-Seite): „Anfragen"-Button anzeigen für Sucher ohne aktive Paarung zu diesem Bieter; führt zu Paarungsanfrage `POST /api/mitfahrt-paarungen`
- [x] 6.3 `EntryCard` (Sucher-Seite): „Einladen"-Button anzeigen für eigene Bieter-Einträge; führt zu Paarungsanfrage `POST /api/mitfahrt-paarungen`
- [x] 6.4 Pending-Paarungen in der jeweiligen EntryCard anzeigen: "⌛ Anfrage von [Name]" mit Bestätigen/Ablehnen-Buttons für die Gegenseite
- [x] 6.5 Bestätigte Paarungen als eigene Sektion „Fahrgemeinschaften" pro GameCard anzeigen (öffentlich sichtbar): Bieter-Name → Sucher-Name (Anzahl Personen)
- [x] 6.6 Kapazitätsanzeige beim Bieter-Eintrag: „X/Y Plätze belegt" basierend auf confirmed Paarungen

## 7. Manuelle Tests

- [ ] 7.1 Flow Sucher initiiert: Anfrage stellen → Bieter bestätigt → beide sehen confirmed Paarung
- [ ] 7.2 Flow Bieter initiiert: Einladung senden → Sucher bestätigt → beide sehen confirmed Paarung
- [ ] 7.3 Kapazitätsgrenze: Bieter mit 1 Platz hat pending Anfrage von Sucher A → Anfrage von Sucher B wird mit 409 sofort abgewiesen
- [ ] 7.4 Stornierung: Bieter löscht Eintrag → Sucher sieht Push-Benachrichtigung
- [ ] 7.5 Stornierung: Sucher storniert bestätigte Paarung → Bieter sieht Push-Benachrichtigung
- [ ] 7.6 Mehrere Gesuche: Nutzer legt zwei Gesuche für dasselbe Spiel an, beide erscheinen in der Liste
