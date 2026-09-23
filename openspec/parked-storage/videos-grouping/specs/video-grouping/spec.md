## ADDED Requirements

### Requirement: Gruppen-Schlüssel-Ermittlung

Das System SHALL für jedes Video einen Gruppen-Schlüssel bestimmen, der als Bündelungskriterium in der Anzeige dient. Das Schema MUSS dafür unverändert bleiben (keine neue Spalte, keine Migration).

#### Scenario: Video mit Spiel-Bezug

- **WHEN** ein Video die Eigenschaft `game_id` mit einem nicht-`NULL`-Wert hat
- **THEN** lautet sein Gruppen-Schlüssel `game:<game_id>`

#### Scenario: Video ohne Spiel-Bezug

- **WHEN** ein Video `game_id IS NULL` hat und einen nicht-leeren `title` besitzt
- **THEN** lautet sein Gruppen-Schlüssel `title:<trim(title)>` (case-sensitive, Whitespace an den Rändern entfernt)

#### Scenario: Video ohne Spiel und ohne Titel

- **WHEN** ein Video weder `game_id` noch einen nicht-leeren `title` hat
- **THEN** bildet das Video eine eigene Einzelgruppe mit dem Schlüssel `video:<id>` und wird nie mit anderen gebündelt

### Requirement: Sortierung innerhalb einer Gruppe

Das System SHALL die Videos einer Gruppe stabil und vorhersagbar reihen, sodass typischerweise die erste hochgeladene Aufnahme zuerst erscheint (1. Halbzeit vor 2. Halbzeit).

#### Scenario: Mehrere Videos in einer Gruppe

- **WHEN** eine Gruppe mehrere Videos enthält
- **THEN** werden sie aufsteigend nach `created_at` sortiert; bei gleichem `created_at` als Tiebreaker aufsteigend nach `id`

### Requirement: Permissions-Filter ist vorgeschaltet

Die Gruppierung SHALL nie Videos sichtbar machen, die der angemeldete Nutzer nicht lesen darf. Die bestehende rollen- und teambasierte Lese-Berechtigung MUSS vor der Gruppierung greifen.

#### Scenario: Eingeschränkter Lesezugriff

- **WHEN** ein Nutzer Videos auflistet und in einer Gruppe befinden sich Videos verschiedener Teams, von denen er nur eines lesen darf
- **THEN** enthält die ihm angezeigte Gruppe nur die für ihn lesbaren Videos; nicht lesbare Videos werden weder gezählt noch erwähnt

### Requirement: Video-Liste zeigt gruppierte Karten

Die Video-Listenseite SHALL Videos pro Gruppe als eine Karte rendern.

#### Scenario: Gruppe mit genau einem Video

- **WHEN** eine Gruppe nur ein einziges Video enthält
- **THEN** wird die Karte als kompakte Einzel-Video-Karte gerendert (kein Aufklappen, direkter Klick führt zur Detailseite des Videos)

#### Scenario: Gruppe mit mehreren Videos

- **WHEN** eine Gruppe zwei oder mehr Videos enthält
- **THEN** wird eine Sammel-Karte mit Spiel- bzw. Titel-Bezeichnung, Anzahl der Videos und einer Vorschau (Titel des ersten Videos der Gruppe) gerendert
- **AND** die Karte ist standardmäßig eingeklappt
- **AND** ein Aufklappen blendet alle Videos der Gruppe in der definierten Sortierreihenfolge ein, jedes einzeln verlinkt auf seine Detailseite

### Requirement: Detailseite zeigt Geschwister-Videos

Die Video-Detailseite SHALL unter dem Player eine Liste der übrigen Videos derselben Gruppe darstellen.

#### Scenario: Aktuelles Video gehört zu einer Gruppe mit weiteren Videos

- **WHEN** der Nutzer ein Video öffnet, dessen Gruppe noch weitere lesbare Videos enthält
- **THEN** erscheint unter dem Player eine Sektion „Weitere Videos zu …" mit den anderen Gruppen-Mitgliedern in der definierten Sortierreihenfolge; jeder Eintrag verlinkt auf die jeweilige Detailseite

#### Scenario: Aktuelles Video ist Einzel-Video

- **WHEN** der Nutzer ein Video öffnet, dessen Gruppe nur dieses eine Video enthält
- **THEN** wird die Geschwister-Sektion nicht gerendert

### Requirement: Upload-Hinweis bei bestehender Gruppe

Der Video-Upload SHALL den Nutzer informieren, sobald die gewählte Spiel-Zuordnung **oder** der eingegebene Titel zu einer bereits existierenden, für ihn lesbaren Gruppe passt.

#### Scenario: Spiel-Zuordnung trifft bestehende Gruppe

- **WHEN** der Nutzer ein Spiel auswählt, zu dem bereits eines oder mehrere für ihn lesbare Videos existieren
- **THEN** zeigt das Upload-Formular einen nicht-blockierenden Hinweis „Es gibt bereits N Video(s) zu diesem Spiel — dies wird Video Nr. N+1"
- **AND** das Titel-Feld erhält als Default-/Placeholder-Vorschlag einen logischen Folge-Titel (z. B. „2. Halbzeit", falls der bestehende Titel „1. Halbzeit" oder ähnlich lautet; sonst generisch „Video N+1")

#### Scenario: Titel trifft bestehende Titel-Gruppe (ohne Spiel)

- **WHEN** der Nutzer ohne Spiel-Zuordnung einen Titel eingibt, der nach Trim exakt mit dem Titel mindestens eines bestehenden, für ihn lesbaren Videos (`game_id IS NULL`) übereinstimmt
- **THEN** zeigt das Upload-Formular denselben Hinweis und Titel-Vorschlag

#### Scenario: Hinweis ist nicht blockierend

- **WHEN** der Upload-Hinweis erscheint
- **THEN** kann der Nutzer den Upload unverändert fortsetzen, den vorgeschlagenen Titel überschreiben oder ignorieren; das Absenden wird durch den Hinweis nie verhindert

### Requirement: Keine Schema- oder API-Vertragsänderung als Voraussetzung

Die Funktion SHALL ohne Datenbank-Migration und ohne Bruch des bestehenden `GET /api/videos`-Antwortformats auskommen.

#### Scenario: Bestehender Listen-Endpoint reicht

- **WHEN** das Frontend die Liste der Videos lädt
- **THEN** verwendet es den bestehenden `GET /api/videos`-Endpoint und führt die Gruppierung im Speicher durch, ohne dass dessen Antwortform geändert werden muss
