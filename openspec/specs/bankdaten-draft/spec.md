# bankdaten-draft Specification

## Purpose

Diese Spezifikation beschreibt die Capability `bankdaten-draft`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Bankdaten als atomarer Draft

Das System SHALL Änderungen an IBAN und Kontoinhaber als einen einzigen Draft mit `field_name='bankdaten'` und `new_value={iban, account_holder}` speichern.

#### Scenario: Nutzer speichert Bankdaten

- **WHEN** ein Nutzer mit verknüpftem Mitglied IBAN und/oder Kontoinhaber im Profil ändert und speichert
- **THEN** wird genau ein Draft-Eintrag mit `field_name='bankdaten'` erzeugt oder aktualisiert (UPSERT)

#### Scenario: Bestehender bankdaten-Draft wird aktualisiert

- **WHEN** ein Nutzer erneut Bankdaten ändert, obwohl bereits ein ausstehender `bankdaten`-Draft existiert
- **THEN** wird der bestehende Draft aktualisiert (neuer `new_value`, neue `created_at`), kein zweiter Draft erstellt

#### Scenario: Admin akzeptiert bankdaten-Draft

- **WHEN** ein Admin einen `bankdaten`-Draft akzeptiert
- **THEN** werden IBAN und Kontoinhaber des Mitglieds gleichzeitig aktualisiert und der Draft gelöscht

#### Scenario: Admin lehnt bankdaten-Draft ab

- **WHEN** ein Admin einen `bankdaten`-Draft ablehnt
- **THEN** wird der Draft gelöscht ohne Änderung der Mitgliedsdaten

#### Scenario: Ungültiger field_name wird abgelehnt

- **WHEN** ein Nutzer eine Änderungsanfrage mit `field_name='iban'` oder `field_name='account_holder'` sendet
- **THEN** antwortet das Backend mit HTTP 400


### Requirement: Separate iban- und account_holder-Drafts

**Reason**: Ersetzt durch den kombinierten `bankdaten`-Draft; IBAN und Kontoinhaber sind fachlich zusammengehörig und müssen atomar genehmigt werden.
**Migration**: Bestehende `iban`/`account_holder`-Drafts in der DB können vom Admin abgelehnt werden; neue Bankdaten-Anfragen erzeugen ausschließlich `bankdaten`-Drafts.
