## REMOVED Requirements

### Requirement: Eine unauflösbare dynamische Dauer fällt auf die absolute zurück

**Reason:** Der Rückfall macht eine Fehldefinition dauerhaft unsichtbar — der Slot trägt
eine plausible, aber falsche Dauer, die Dienstbörse zeigt eine Spanne und das Dienstkonto
bucht Stunden, ohne dass irgendwo auffällt, dass die Versätze nicht zusammenpassen. An
seine Stelle treten zwei Stufen: Was nie funktionieren kann, wird beim Pflegen abgewiesen
(neue Requirement „Eine unmögliche Zeitspanne wird abgewiesen"); was erst gegen einen
konkreten Termin scheitert, erzeugt keinen Slot und wird in der Regen-Zusammenfassung
gemeldet (neue Requirement „Eine unauflösbare dynamische Dauer erzeugt keinen Slot").

**Migration:** Keine. `duty_types.hours_value` und `game_template_items.hours_value`
bleiben unverändert gespeichert und gelten weiter im Modus `absolut`; im Modus
`dynamisch` werden sie nur nicht mehr gelesen. Bestandsslots behalten ihre Dauer bis zum
nächsten Regen-Lauf.

## ADDED Requirements

### Requirement: Eine unmögliche Zeitspanne wird abgewiesen

`POST /api/duty-types`, `PUT /api/duty-types/{id}` und `PUT /api/duty-templates/{id}`
SHALL im Modus `dynamisch` eine Kombination abweisen, deren Ende **an keinem Termin**
nach dem Start liegen kann: Start-Anker und End-Anker sind gleich **und** der End-Versatz
ist kleiner oder gleich dem Start-Versatz. Die Antwort SHALL HTTP 400 sein, geprüft
**vor** jedem Schreibvorgang — bei `PUT` bleibt der Bestand vollständig unverändert.

Bei **verschiedenen** Ankern SHALL nicht validiert werden. Die Dauer hängt dort von der
Spieldauer des konkreten Termins ab, die zum Pflegezeitpunkt nicht feststeht: „Start bei
Anpfiff, Ende 15 min vor Spielende" ist eine gültige Definition, die bei jedem
hinreichend langen Spiel eine positive Dauer ergibt.

Im Modus `absolut` SHALL die Prüfung nicht greifen — End-Anker und End-Versatz sind dort
bedeutungslos.

Das Frontend SHALL dieselbe Regel anwenden und das Speichern mit einer Meldung am Feld
blockieren, statt den Server antworten zu lassen.

#### Scenario: Gleicher Anker, Ende nicht nach dem Start

- **WHEN** ein Vorstand einen Diensttyp im Modus `dynamisch` mit Start-Anker `start`
  +40 min und End-Anker `start` +25 min speichert
- **THEN** antwortet der Server mit HTTP 400
- **AND** ist nichts persistiert

#### Scenario: Verschiedene Anker bleiben erlaubt

- **WHEN** ein Vorstand Start-Anker `start` +0 min und End-Anker `end` −15 min speichert
- **THEN** wird die Definition angenommen
- **AND** ergibt sie bei jedem Termin, dessen Spieldauer 15 Minuten übersteigt, eine
  positive Dauer

#### Scenario: Absoluter Modus ist von der Prüfung nicht betroffen

- **WHEN** dieselbe unmögliche Anker-/Versatz-Kombination im Modus `absolut` gespeichert
  wird
- **THEN** wird sie angenommen
- **AND** trägt der erzeugte Slot die gepflegte Stundenzahl

#### Scenario: Vorlagen-Zeile mit unmöglicher Spanne

- **WHEN** eine Vorlage mit einem Eintrag gespeichert wird, dessen Spanne unmöglich ist
- **THEN** antwortet der Server mit HTTP 400
- **AND** bleiben die bestehenden Einträge der Vorlage unverändert

### Requirement: Eine unauflösbare dynamische Dauer erzeugt keinen Slot

Ergibt die dynamische Auflösung gegen einen konkreten Termin keine positive Dauer, SHALL
für diesen Termin **kein** Slot entstehen. Es SHALL keinen Rückfall auf die gepflegte
Stundenzahl geben.

Der Ausfall SHALL in der Regen-Zusammenfassung als eigener Eintrag mit Datum und
Diensttyp erscheinen (`invalid_span`), getrennt von `skipped` — `skipped` meint eine
gewollte Auslassung der Varianten-Logik, `invalid_span` einen Definitionsfehler. Die
Oberfläche SHALL ihn entsprechend als Fehler ausweisen.

Eine Zusage auf einem dadurch entfallenen Bestandsslot SHALL wie jeder andere entfallene
Dienst behandelt werden: Der Helfer wird über die Entfernung benachrichtigt.

#### Scenario: Spieldauer lässt die Spanne zusammenschrumpfen

- **WHEN** ein Dienst bei Spielende −0 min startet und bei Anpfiff +30 min enden soll
- **AND** das Spiel 60 Minuten dauert
- **THEN** entsteht kein Slot
- **AND** meldet die Zusammenfassung einen `invalid_span`-Eintrag mit Datum und Diensttyp

#### Scenario: Kein Rückfall auf die gepflegte Stundenzahl

- **WHEN** derselbe Termin regeneriert wird und der Diensttyp eine `hours_value` von 2
  Stunden trägt
- **THEN** entsteht kein Slot mit 2 Stunden Dauer

#### Scenario: Helfer auf einem entfallenen Dienst wird benachrichtigt

- **WHEN** auf dem betroffenen Bestandsslot eine Zusage lag
- **THEN** erhält der Helfer die reguläre „Dienst wurde entfernt"-Meldung

## MODIFIED Requirements

### Requirement: Ein Diensttyp kann eine dynamische Dauer haben

Ein Diensttyp SHALL einen Dauer-Modus tragen: `absolut` oder `dynamisch`. Im Modus
`absolut` SHALL sich nichts gegenüber dem bisherigen Verhalten ändern — die Dauer ist die
gepflegte Stundenzahl.

Im Modus `dynamisch` SHALL das Ende des Dienstes über einen **End-Anker** und einen
**End-Versatz** bestimmt werden, mit denselben zwei Ankern wie der Start: `start`
(Anpfiff) und `end` (Spielende). Der Versatz SHALL in beide Richtungen zulässig sein.
Die Dauer SHALL dort ausschließlich die Differenz aus aufgelöster End- und Startzeit
sein; die gepflegte Stundenzahl SHALL keine Rolle spielen.

Die Auflösung des End-Ankers SHALL identisch zur Auflösung des Start-Ankers sein:
`end` verwendet die gepflegte Endzeit des Termins, und andernfalls den Anpfiff zuzüglich
der ermittelten Spieldauer.

Die Vorlagen-Zeile SHALL Modus, End-Anker und End-Versatz wie die übrigen Item-Felder
per Copy-on-pick vom Diensttyp übernehmen und danach eigenständig führen.

Die Oberfläche SHALL die beiden Modi nach der Entscheidung benennen, die sie treffen
lassen — `absolut` als **„Startzeit + Dauer"**, `dynamisch` als **„Startzeit + Endzeit"** —
und die Felder in der Reihenfolge der Rechnung zeigen: Modus, dann **Start-Anker** und
**Start-Versatz**, darunter je nach Modus die **Dauer** oder **End-Anker** und
**End-Versatz**. Im Modus `dynamisch` SHALL kein Dauer-Feld angeboten werden. Die
gespeicherten Werte (`absolut`/`dynamisch`) SHALL die Umbenennung nicht berühren.

#### Scenario: Dienst dauert so lang wie das Spiel

- **WHEN** ein Diensttyp im Modus `dynamisch` bei Anpfiff −30 min startet und bei
  Spielende +15 min endet
- **AND** zwei Termine unterschiedlicher Altersklassen dieselbe Vorlage nutzen
- **THEN** tragen die erzeugten Slots **unterschiedliche** Dauern
- **AND** entspricht jede der Spieldauer des jeweiligen Termins zuzüglich der Versätze

#### Scenario: Gepflegte Endzeit hat Vorrang

- **WHEN** der Termin eine eigene Endzeit trägt und der End-Anker `end` ist
- **THEN** rechnet die Dauer gegen diese Endzeit
- **AND** nicht gegen Anpfiff zuzüglich der Spieldauer

#### Scenario: Halbzeit-Dienst über zwei Anpfiff-Versätze

- **WHEN** Start und Ende beide am Anker `start` hängen, mit +25 min und +40 min
- **THEN** ist die Dauer des Slots 15 Minuten
- **AND** bleibt sie unabhängig von der Spieldauer des Termins

#### Scenario: Absoluter Modus bleibt unverändert

- **WHEN** ein Diensttyp im Modus `absolut` steht
- **THEN** trägt der erzeugte Slot exakt die gepflegte Stundenzahl
- **AND** spielen End-Anker und End-Versatz keine Rolle

#### Scenario: Kein Dauer-Feld im Modus „Startzeit + Endzeit"

- **WHEN** ein Vorstand in der Diensttyp- oder Vorlagen-Maske auf „Startzeit + Endzeit"
  umschaltet
- **THEN** verschwindet das Dauer-Feld
- **AND** erscheinen stattdessen End-Anker und End-Versatz unter den Start-Feldern

### Requirement: Manuell angelegte Dienste bleiben absolut

Ein per `POST /api/duty-slots` angelegter Dienst SHALL eine absolute Dauer tragen. Der
Dauer-Modus SHALL für solche Slots nicht angeboten werden, da sie `is_custom=1` tragen und
vom Regen nie erneuert werden — eine als dynamisch etikettierte Dauer würde dort nie
nachgeführt.

Das Anlege-Formular SHALL die Dauer aus einer dynamischen Typ-Definition jedoch berechnen
und vorbelegen dürfen.

Der Termin-Dialog SHALL beim Anlegen **und** beim Bearbeiten eines Dienstes darauf
hinweisen, dass der Dienst dadurch als manuell gepflegt gilt und von der automatischen
Regeneration nicht mehr angefasst wird. Beim Bearbeiten eines bisher automatisch
erzeugten Dienstes ist das eine Nebenwirkung des Speicherns und SHALL vor dem Speichern
sichtbar sein.

#### Scenario: Dienst mit dynamischem Typ von Hand hinzufügen

- **WHEN** ein Vorstand über „+ Dienst hinzufügen" einen Diensttyp im Modus `dynamisch`
  auswählt
- **THEN** wird die Dauer aus Anker und Versätzen gegen diesen Termin berechnet vorbelegt
- **AND** ist der so entstandene Slot danach eine feste Zahl, die der Regen nicht anrührt

#### Scenario: Hinweis auf die Herausnahme aus der Regeneration

- **WHEN** ein Vorstand einen automatisch erzeugten Dienst im Termin-Dialog bearbeitet
- **THEN** steht im Dialog, dass der Dienst danach manuell gepflegt ist und nicht mehr
  automatisch regeneriert wird
