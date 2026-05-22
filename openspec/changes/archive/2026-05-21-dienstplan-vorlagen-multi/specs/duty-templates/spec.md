## ADDED Requirements

### Requirement: Mehrere Dienstplan-Vorlagen verwalten
Das System SHALL mehrere Dienstplan-Vorlagen (`game_templates`) parallel unterstützen. Jede Vorlage hat einen Typ (`template_type`: `heim`, `auswärts`, `generisch`). Mehrere Vorlagen gleichen Typs sind erlaubt.

#### Scenario: Tabellarische Übersicht laden
- **WHEN** ein Admin die Seite `/admin/dienstplan-vorlagen` aufruft
- **THEN** zeigt das System eine Tabelle aller vorhandenen Vorlagen mit Name, Typ und Anzahl der Items an

#### Scenario: Neue Vorlage anlegen
- **WHEN** ein Admin auf „Neue Vorlage" klickt und Name + Typ eingibt
- **THEN** legt das System eine neue leere Vorlage an und navigiert zur Detailseite

#### Scenario: Vorlage löschen
- **WHEN** ein Admin in der Listenansicht auf „Löschen" klickt und bestätigt
- **THEN** löscht das System die Vorlage samt aller zugehörigen Items

### Requirement: Vorlage-Typ je Eintrag einstellbar
Jede Vorlage SHALL einen `template_type` haben, der auf der Detailseite geändert werden kann. Gültige Werte sind `heim`, `auswärts` und `generisch`.

#### Scenario: Typ ändern und speichern
- **WHEN** ein Admin auf der Detailseite den Typ von `generisch` auf `heim` ändert und speichert
- **THEN** aktualisiert das System den `template_type` der Vorlage in der Datenbank

#### Scenario: Ungültiger Typ wird abgelehnt
- **WHEN** eine PUT-Anfrage an `/api/admin/duty-templates/{id}` mit einem ungültigen `template_type` gesendet wird
- **THEN** antwortet das System mit HTTP 400

### Requirement: Automatische Vorlagenauswahl bei Slot-Generierung
Das System SHALL bei der Slot-Generierung (`CreateGame`, `RegenerateSlots`, `PreviewSlots`) automatisch die passende Vorlage anhand des Spieltyps wählen.

#### Scenario: Heimspiel wählt Heim-Vorlage
- **WHEN** ein Spiel mit `is_home=true` erstellt oder Slots regeneriert werden
- **THEN** verwendet das System die erste Vorlage mit `template_type='heim'` (ORDER BY id ASC)

#### Scenario: Auswärtsspiel wählt Auswärts-Vorlage
- **WHEN** ein Spiel mit `is_home=false` erstellt oder Slots regeneriert werden
- **THEN** verwendet das System die erste Vorlage mit `template_type='auswärts'` (ORDER BY id ASC)

#### Scenario: Fallback auf generische Vorlage
- **WHEN** kein passendes spezifisches Template (`heim`/`auswärts`) existiert
- **THEN** verwendet das System die erste Vorlage mit `template_type='generisch'` als Fallback

#### Scenario: Kein Template gefunden
- **WHEN** weder ein spezifisches noch ein generisches Template existiert
- **THEN** bricht das System die Slot-Generierung mit einem Fehler ab und gibt eine aussagekräftige Fehlermeldung zurück

### Requirement: REST-Schnittstelle unter neuem Pfad
Das System SHALL alle Endpunkte für Dienstplan-Vorlagen unter `/api/admin/duty-templates` bereitstellen. Der alte Pfad `/api/admin/game-template` entfällt.

#### Scenario: Liste aller Vorlagen abrufen
- **WHEN** ein Admin `GET /api/admin/duty-templates` aufruft
- **THEN** gibt das System eine JSON-Liste aller Vorlagen zurück (id, name, template_type, items)

#### Scenario: Einzelne Vorlage abrufen
- **WHEN** ein Admin `GET /api/admin/duty-templates/{id}` aufruft
- **THEN** gibt das System die Vorlage mit allen Items zurück

#### Scenario: Vorlage erstellen
- **WHEN** ein Admin `POST /api/admin/duty-templates` mit Name und template_type aufruft
- **THEN** legt das System eine neue Vorlage an und gibt sie mit ihrer neuen ID zurück

#### Scenario: Vorlage aktualisieren
- **WHEN** ein Admin `PUT /api/admin/duty-templates/{id}` mit geänderten Feldern aufruft
- **THEN** aktualisiert das System Name, template_type und Items der Vorlage

#### Scenario: Vorlage löschen
- **WHEN** ein Admin `DELETE /api/admin/duty-templates/{id}` aufruft
- **THEN** löscht das System die Vorlage und antwortet mit HTTP 200

#### Scenario: Alter Pfad existiert nicht mehr
- **WHEN** eine Anfrage an `/api/admin/game-template` gesendet wird
- **THEN** antwortet das System mit HTTP 404
