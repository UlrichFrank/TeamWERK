## ADDED Requirements

### Requirement: Beitragslauf-Bestätigung schreibt ein append-only Protokoll ohne Klartext-Bankdaten
`POST /api/fee-run/confirm` SHALL das Ergebnis in ein pro-Saison append-only Textprotokoll
schreiben, das ausschließlich Mitgliedsnummer, Name, Betrag und Erfolg enthält — **niemals eine
IBAN oder andere Klartext-Bankdaten**. Ein zweiter Lauf SHALL den vorherigen Block nicht
überschreiben.

#### Scenario: Bestätigter Lauf schreibt Protokoll
- **WHEN** ein Kassierer/Vorstand einen Lauf mit Ergebnissen bestätigt
- **THEN** die Route SHALL 200 liefern und eine Protokolldatei mit Mitgliedsnummer, Betrag und Erfolgsstatus schreiben

#### Scenario: Protokoll enthält keine Bankdaten
- **WHEN** das Protokoll eines Laufs geschrieben wurde
- **THEN** der Dateiinhalt SHALL keine IBAN und keinen Ciphertext enthalten

#### Scenario: Zweiter Lauf hängt an
- **WHEN** zwei Läufe für dieselbe Saison bestätigt werden
- **THEN** die Datei SHALL beide Lauf-Blöcke enthalten (append-only, kein Überschreiben)

#### Scenario: Erfolgreiche und fehlgeschlagene Einträge getrennt
- **WHEN** ein Lauf mit einem Erfolg und einem Fehlschlag bestätigt wird
- **THEN** die Antwort SHALL `erfolgreich=1` und `nicht_erfolgreich=1` liefern und das Protokoll SHALL einen „Erfolgreich"- und einen „Nicht erfolgreich"-Block enthalten

#### Scenario: Unbekannte Saison
- **WHEN** eine unbekannte Saison-ID bestätigt wird
- **THEN** die Route SHALL 404 liefern und keine Datei schreiben

### Requirement: Protokoll-Rücklesen unterscheidet fehlende Saison von fehlendem Lauf
`GET /api/fee-run/protocol` SHALL das Protokoll einer Saison als `text/plain` zurückgeben. Eine
unbekannte Saison SHALL 404 ergeben; eine gültige Saison ohne bisherigen Lauf SHALL 200 mit
leerem Body ergeben (nicht 404).

#### Scenario: Rücklesen nach Bestätigung
- **WHEN** nach einem bestätigten Lauf das Protokoll abgerufen wird
- **THEN** die Route SHALL 200 (`text/plain`) mit dem Lauf-Block liefern, ohne IBAN

#### Scenario: Gültige Saison ohne Lauf
- **WHEN** eine gültige Saison ohne bisherigen Lauf abgefragt wird
- **THEN** die Route SHALL 200 mit leerem Body liefern (nicht 404)

#### Scenario: Unbekannte Saison
- **WHEN** eine unbekannte Saison-ID abgefragt wird
- **THEN** die Route SHALL 404 liefern

### Requirement: SEPA-Export lehnt ungültige oder ausgeschlossene Mitglieder ab
`POST /api/fee-run/export-data` SHALL mit 400 antworten, wenn ein angefordertes Mitglied
unbekannt oder vom Lauf ausgeschlossen ist (kein SEPA-Mandat oder keine Bankdaten), sowie bei
ungültigem Body. Die Antwort SHALL weiterhin nur Ciphertext-Envelopes liefern (kein Klartext).

#### Scenario: Mitglied ohne SEPA-Mandat
- **WHEN** der Export ein Mitglied ohne gültiges SEPA-Mandat anfordert
- **THEN** die Route SHALL 400 liefern

#### Scenario: Mitglied ohne Bankdaten-Envelope
- **WHEN** der Export ein Mitglied ohne hinterlegte (verschlüsselte) Bankdaten anfordert
- **THEN** die Route SHALL 400 liefern

#### Scenario: Unbekannte Mitglieds-ID
- **WHEN** der Export eine Mitglieds-ID anfordert, die nicht im Lauf enthalten ist
- **THEN** die Route SHALL 400 liefern

### Requirement: Beitragslauf-Vorschau summiert den Einzugsbetrag korrekt
Die Vorschau (`GET /api/fee-run/preview`) SHALL neben den Einzelposten korrekte Summen liefern:
Anzahl einbezogener Mitglieder, Gesamtbetrag der einbezogenen Posten sowie die für den
Kassierer sichtbare Einzugssumme.

#### Scenario: Summen über einbezogene und ausgeschlossene Mitglieder
- **WHEN** die Vorschau ein Set aus einbezogenen und ausgeschlossenen Mitgliedern berechnet
- **THEN** `included_count` und `total_cent` SHALL nur die einbezogenen Posten aggregieren, und die ausgewiesene Gesamtsumme SHALL dieser Aggregation entsprechen

### Requirement: Halbierungsmatrix ist vollständig für unterjährigen Austritt mit Stammverein
Die Beitragsberechnung SHALL für ein unterjährig ausgetretenes Mitglied mit gesetztem
`home_club_id` die Kategorie `aktiv_mit` mit halbiertem Betrag liefern.

#### Scenario: Unterjähriger Austritt mit Stammverein
- **WHEN** ein Mitglied `status='ausgetreten'` mit `exit_date` im Saisonfenster und gesetztem `home_club_id` berechnet wird (join_date vor der Saison, Saison nicht inaugural)
- **THEN** die Vorschau SHALL Kategorie `aktiv_mit`, `half=true`, `half_reason=austritt` und den halbierten `aktiv_mit`-Betrag liefern
