# trainingstagebuch-retention Specification

## Purpose

Saison-Zuordnung von Trainingstagebuch-Einträgen über die aktive Saison (nicht über einen
Datumsvergleich) sowie automatische, tägliche Löschung der Nachweisdateien 90 Tage nach
Saisonende — der Eintrag selbst bleibt dauerhaft als Historie erhalten.

## Requirements

### Requirement: Einträge werden der bei Erfassung aktiven Saison zugeordnet

Das System SHALL beim Anlegen eines Trainingstagebuch-Eintrags `season_id` aus der Saison mit
`seasons.is_active = 1` übernehmen — **nicht** aus einem Vergleich von `trained_on` mit den
Saisonzeiträumen.

Grund: Saisons werden als freie Zeiträume gepflegt und schließen nicht lückenlos aneinander an.
Eine datumsbasierte Zuordnung ließe alle Einträge aus der Sommerpause ohne Saison zurück — genau
den Zeitraum, für den das Tagebuch gedacht ist.

Existiert keine aktive Saison, SHALL `season_id` auf `NULL` gesetzt werden. Ein späterer
Saisonwechsel SHALL bestehende Einträge **nicht** umhängen.

#### Scenario: Eintrag in der Sommerpause

- **WHEN** ein Spieler am 10. Juli einen Eintrag erfasst, während die Saison 25/26 aktiv ist und
  am 31. Mai endete
- **THEN** trägt der Eintrag die `season_id` der Saison 25/26
- **THEN** ist er damit an deren Retention gebunden

#### Scenario: Keine aktive Saison

- **WHEN** ein Eintrag erfasst wird, während keine Saison aktiv ist
- **THEN** wird `season_id = NULL` gespeichert
- **THEN** antwortet das System dennoch mit HTTP 201

#### Scenario: Saisonwechsel hängt Bestandseinträge nicht um

- **WHEN** der Vorstand eine neue Saison aktiv schaltet
- **THEN** behalten bereits erfasste Einträge ihre bisherige `season_id`

---

### Requirement: Nachweisdateien werden 90 Tage nach Saisonende automatisch gelöscht

Das System SHALL täglich alle Einträge ermitteln, deren zugeordnete Saison vor **mehr als 90
Tagen** endete und die noch eine Nachweisdatei besitzen, und für diese

- die Datei aus dem Speicherverzeichnis entfernen,
- `proof_purged_at` auf den Löschzeitpunkt setzen,
- `proof_disk_name` auf `NULL` setzen.

Datum, Art, Dauer, Intensität und Notiz des Eintrags SHALL dabei **unverändert** erhalten bleiben.

Einträge mit `season_id IS NULL` SHALL **niemals** automatisch bereinigt werden.

Der Vorgang SHALL idempotent sein: ein zweiter Lauf am selben Tag verändert nichts und schlägt
nicht fehl. Eine bereits fehlende Datei SHALL kein Fehlerfall sein.

Es SHALL **keine** Vorwarnung per Push oder E-Mail versendet werden.

#### Scenario: Nachweis nach Ablauf der Frist

- **WHEN** die Saison eines Eintrags vor 91 Tagen endete und der Eintrag einen Nachweis hat
- **THEN** ist die Datei nach dem Lauf entfernt
- **THEN** ist `proof_purged_at` gesetzt und `proof_disk_name` leer
- **THEN** sind `trained_on`, `kind`, `duration_min` und `rpe` unverändert

#### Scenario: Nachweis innerhalb der Frist

- **WHEN** die Saison eines Eintrags vor 89 Tagen endete
- **THEN** bleibt die Datei erhalten und `proof_purged_at` ungesetzt

#### Scenario: Eintrag ohne Saison

- **WHEN** ein Eintrag mit `season_id IS NULL` einen Nachweis hat
- **THEN** wird dieser Nachweis in keinem Lauf gelöscht

#### Scenario: Zweiter Lauf am selben Tag

- **WHEN** der Retention-Lauf zweimal hintereinander ausgeführt wird
- **THEN** verändert der zweite Lauf keinen Datensatz und meldet keinen Fehler

#### Scenario: Datei bereits von Hand entfernt

- **WHEN** die Datei eines fälligen Nachweises im Dateisystem fehlt
- **THEN** setzt der Lauf dennoch `proof_purged_at` und `proof_disk_name = NULL`, ohne Fehler

---

### Requirement: Gelöschte Nachweise sind als solche erkennbar

Das System SHALL auf `GET /api/training-diary/{id}/proof` mit **HTTP 410 Gone** antworten, wenn
`proof_purged_at` gesetzt ist — abgegrenzt von HTTP 404 für einen Eintrag, an dem nie ein Nachweis
hing.

Die Listen-Endpunkte SHALL je Eintrag erkennbar machen, ob ein Nachweis vorhanden, nie vorhanden
gewesen oder durch die Retention entfernt worden ist.

Das Frontend SHALL für einen entfernten Nachweis einen Hinweis („Nachweis gelöscht") anzeigen statt
eines fehlschlagenden Bildabrufs, und an den Erfassungs- und Übersichtsseiten dauerhaft darauf
hinweisen, dass Nachweise 90 Tage nach Saisonende automatisch gelöscht werden.

#### Scenario: Abruf eines gelöschten Nachweises

- **WHEN** ein Berechtigter den Nachweis eines bereinigten Eintrags abruft
- **THEN** antwortet das System mit HTTP 410

#### Scenario: Abruf bei nie vorhandenem Nachweis

- **WHEN** ein Berechtigter den Nachweis eines Eintrags abruft, der nie einen hatte
- **THEN** antwortet das System mit HTTP 404

#### Scenario: Anzeige eines gelöschten Nachweises

- **WHEN** ein Eintrag mit gesetztem `proof_purged_at` angezeigt wird
- **THEN** erscheint der Hinweis „Nachweis gelöscht" anstelle eines Bildes
- **THEN** wird kein Bildabruf ausgelöst

#### Scenario: Hinweis auf die Aufbewahrungsdauer

- **WHEN** ein Spieler das Erfassungsformular öffnet
- **THEN** ist der Hinweis sichtbar, dass Nachweise 90 Tage nach Saisonende gelöscht werden
