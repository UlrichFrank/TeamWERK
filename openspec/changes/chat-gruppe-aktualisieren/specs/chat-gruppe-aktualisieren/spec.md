# Spec Delta

## Purpose

Eine Chat-Gruppe, die aus Standard-Gruppen (Mannschafts-Kacheln, „Alle Trainer", Übungsgruppen) entstanden ist, merkt sich diese Herkunft. Ihr Ersteller kann sie damit per Vorschau und Bestätigung wieder an den aktuellen Kader angleichen.

## ADDED Requirements

### Requirement: Herkunft einer Gruppen-Konversation

Das System SHALL je Gruppen-Konversation eine Menge von 0–n **Herkunfts-Kacheln** speichern. Eine Kachel ist ein Tripel `{groupType, refId, kind}`. Zulässig sind genau die Kacheln, die `GET /api/chat/team-groups` liefern kann:
- `groupType = 'team'`, `refId` = `teams.id`, `kind` ∈ `trainer`/`spieler`/`eltern`
- `groupType = 'team'`, `refId = 0`, `kind = 'alle_trainer'`
- `groupType = 'practice'`, `refId` = `kader.id` einer Übungsgruppe, `kind` ∈ `trainer`/`spieler`/`eltern`

Doppelte Tripel SHALL zu einem zusammengefasst werden. Direkt-Konversationen haben keine Herkunft. Beim Einführen der Funktion SHALL das System keiner bestehenden Gruppe eine Herkunft zuweisen. Bestandsgruppen bleiben ohne Herkunft, bis ihr Ersteller sie festlegt.

#### Scenario: Bestandsgruppe hat nach dem Deploy keine Herkunft
- **WHEN** eine vor dem Deploy angelegte Gruppe abgeglichen werden soll
- **THEN** liefert die Vorschau für eine leere Kachel-Menge `sources = []`, und weder `add` noch `remove` enthalten Einträge, die aus einer geratenen Herkunft stammen

### Requirement: Herkunft beim Anlegen speichern

`POST /api/chat/conversations` mit `type = 'group'` SHALL ein optionales Feld `sources: [{groupType, refId, kind}]` annehmen und die Kacheln als Herkunft der neuen Gruppe speichern. Jede Kachel MUSS für den Caller sichtbar sein, nach denselben Regeln wie beim Auflösen der Kachel. Eine unzulässige Kachel (unbekannter `kind`, unbekannter `groupType`, Kachel existiert in der aktiven Saison nicht) SHALL mit HTTP 400 abgelehnt werden, eine nicht sichtbare mit HTTP 403. In beiden Fällen entsteht keine Konversation. Ohne `sources` verhält sich der Endpoint wie bisher.

#### Scenario: Gruppe aus zwei Kacheln anlegen
- **WHEN** ein Trainer von T1 eine Gruppe mit `memberIds` und `sources = [{team, T1, spieler}, {team, T1, eltern}]` anlegt
- **THEN** antwortet der Server mit HTTP 201, und die Gruppe trägt genau diese zwei Kacheln als Herkunft

#### Scenario: Fremde Kachel beim Anlegen
- **WHEN** ein Spieler von T1 eine Gruppe mit `sources = [{team, T2, spieler}]` anlegt und T2 nicht sehen darf
- **THEN** antwortet der Server mit HTTP 403, und es entsteht keine Konversation

#### Scenario: Ungültiger kind beim Anlegen
- **WHEN** eine Gruppe mit `sources = [{team, T1, foobar}]` angelegt wird
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Abgleich-Vorschau

Das System SHALL `POST /api/chat/conversations/{id}/sync/preview` mit Body `{sources: [...]}` bereitstellen. Die Antwort beschreibt, was ein Abgleich gegen diese Kacheln ändern würde. Die Route SHALL nichts schreiben und nichts broadcasten. Die übergebenen `sources` ersetzen für die Rechnung die gespeicherte Herkunft. Fehlt das Feld, gilt die gespeicherte Herkunft.

Die **Soll-Menge** ist die Vereinigung der aufgelösten Kacheln (Regeln aus `chat-team-groups`, inklusive Ausschluss ausgetretener Mitglieder) zuzüglich des Erstellers. Die Antwort enthält:
- `sources`: die verwendeten Kacheln, je mit `label` (Anzeigename wie im Dialog „Neues Gespräch"), `total` (Größe der aufgelösten Kachel ohne Ersteller) und `alreadyIn` (davon aktive Mitglieder der Gruppe)
- `add: [{id, name}]`: Personen in der Soll-Menge, die kein aktives Mitglied sind (`left_at IS NULL`). Dazu zählen auch früher Ausgetretene oder Entfernte (`left_at` gesetzt).
- `remove: [{id, name}]`: aktive Mitglieder, die nicht in der Soll-Menge sind. Der Ersteller SHALL nie enthalten sein.
- `suggestions`: alle für den Caller sichtbaren Kacheln (wie `GET /api/chat/team-groups`), je mit `total` und `alreadyIn`. Sie dienen nur der Auswahl und sind nie vorausgewählt.

Ist `sources` leer, SHALL `add` und `remove` leer sein. Ein Abgleich gegen „nichts" entfernt niemanden.

Nur der Ersteller (`conversations.created_by`) darf die Vorschau abrufen (sonst HTTP 403). Existiert die Konversation nicht, antwortet der Server mit HTTP 404, ist sie keine Gruppe, mit HTTP 400. Eine formal ungültige Kachel (unbekannter `groupType`/`kind`) ergibt HTTP 400. Diese Prüfungen SHALL erst nach der Ersteller-Prüfung laufen.

**Sperre statt Massenentfernung:** Ist eine Kachel für den Caller nicht sichtbar (z. B. nach einer Übergabe der Gruppe an jemanden ohne Zugriff auf das Team) oder löst sie zu **null** Personen auf (z. B. nach dem Saisonwechsel, solange der neue Kader leer ist), SHALL die Vorschau mit HTTP 200 antworten. Die Kachel trägt dann `problem = 'not_visible'` bzw. `'empty'`, die Antwort `blocked = true`, und `add`/`remove` sind leer. Andernfalls schlüge jedes Mitglied dieser Kachel in `remove` auf. Die Kachel-Bezeichnung (`label`) wird auch bei `not_visible` geliefert, damit der Ersteller sie erkennen und abwählen kann.

#### Scenario: Neuer Spieler erscheint in add
- **WHEN** die Gruppe Herkunft `{team, T1, spieler}` hat, nach dem Anlegen ein Spieler S in den Kader von T1 aufgenommen wurde und der Ersteller die Vorschau abruft
- **THEN** enthält `add` den Spieler S, und `remove` ist leer

#### Scenario: Ausgetretener Spieler erscheint in remove
- **WHEN** ein Mitglied der Gruppe inzwischen `members.status = 'ausgetreten'` hat
- **THEN** enthält `remove` dieses Mitglied

#### Scenario: Manuell hinzugefügte Person erscheint in remove
- **WHEN** der Ersteller eine Person ohne Bezug zu den Kacheln von Hand hinzugefügt hat
- **THEN** enthält `remove` diese Person (der Ersteller kann sie im Modal abwählen)

#### Scenario: Ersteller wird nie entfernt
- **WHEN** der Ersteller selbst in keiner Kachel enthalten ist
- **THEN** steht er nicht in `remove`

#### Scenario: Überlappung je Kachel
- **WHEN** eine Kachel 12 Personen außer dem Ersteller auflöst und 11 davon aktive Mitglieder sind
- **THEN** trägt die Kachel `total = 12` und `alreadyIn = 11`

#### Scenario: Leere Herkunft ändert nichts
- **WHEN** die Vorschau mit `sources = []` abgerufen wird
- **THEN** sind `add` und `remove` leer, und `suggestions` enthält die sichtbaren Kacheln mit ihren Überlappungen

#### Scenario: Leerer Kader nach Saisonwechsel sperrt den Abgleich
- **WHEN** die Herkunft `{team, T1, spieler}` enthält und T1 in der aktiven Saison keinen Spieler hat
- **THEN** antwortet die Vorschau mit `blocked = true`, die Kachel trägt `problem = 'empty'`, und `remove` ist leer

#### Scenario: Nicht sichtbare Kachel sperrt den Abgleich
- **WHEN** die Gruppe an einen neuen Ersteller übergeben wurde, der die Herkunfts-Kachel nicht sehen darf, und dieser die Vorschau abruft
- **THEN** antwortet der Server mit HTTP 200, `blocked = true`, die Kachel trägt `problem = 'not_visible'` und ihr `label`

#### Scenario: Nicht-Ersteller ruft die Vorschau ab
- **WHEN** ein aktives Mitglied, das nicht Ersteller ist, die Vorschau aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Unbekannte Konversation
- **WHEN** die Vorschau für eine nicht existierende Konversations-ID aufgerufen wird
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Direkt-Konversation
- **WHEN** die Vorschau für eine Direkt-Konversation aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Abgleich anwenden

Das System SHALL `POST /api/chat/conversations/{id}/sync/apply` mit Body `{sources, addUserIds, removeUserIds}` bereitstellen. Berechtigung und Fehlerfälle entsprechen der Vorschau (404 → 403 Ersteller → 400 Gruppe/Kacheln). Wäre die Vorschau zu diesen `sources` gesperrt (`blocked`), SHALL `apply` mit HTTP 409 `sync_blocked` antworten und nichts ändern. Der Server SHALL den Diff zum Zeitpunkt des Anwendens neu berechnen und darauf aufbauend:
- jede ID in `addUserIds` MUSS im neu berechneten `add` stehen, jede ID in `removeUserIds` im neu berechneten `remove`. Sonst antwortet er mit HTTP 409 `sync_stale` und ändert nichts.
- jede hinzuzufügende Person MUSS `canContactUser` bestehen, sonst HTTP 403 ohne Änderung.
- `sources` als neue Herkunft speichern (ersetzt die bisherige vollständig), die Auswahl anwenden und beides in **einer** Transaktion erledigen.

Hinzufügen reaktiviert eine bestehende Mitgliedszeile (`left_at = NULL`) oder legt eine an. Entfernen setzt `left_at`. Je hinzugefügter Person SHALL eine Systemnachricht „wurde hinzugefügt" entstehen, je entfernter „wurde entfernt", wie bei den Einzel-Routen. Nach dem Commit SHALL der Server an alle aktiven Mitglieder `chat:new-message:<id>` senden, falls jemand hinzugefügt wurde, und `chat:member-left:<id>` an alle aktiven Mitglieder und die Entfernten, falls jemand entfernt wurde. Eine Anwendung mit leeren Listen speichert nur die Herkunft und sendet `chat:new-message:<id>` an den Ersteller, damit seine übrigen offenen Sitzungen nachladen. Antwort: HTTP 200 mit `{added, removed}` (Anzahl).

#### Scenario: Auswahl anwenden
- **WHEN** der Ersteller `apply` mit `addUserIds = [S]` und `removeUserIds = [A]` aufruft, S im `add` und A im `remove` steht
- **THEN** ist S aktives Mitglied, A hat `left_at` gesetzt, es gibt je eine Systemnachricht, die Herkunft ist gespeichert, und die Antwort lautet `{added: 1, removed: 1}`

#### Scenario: Abgewählte Person bleibt unberührt
- **WHEN** `remove` die Personen A und B enthält und der Ersteller nur `removeUserIds = [A]` schickt
- **THEN** bleibt B aktives Mitglied

#### Scenario: Fremde ID wird abgelehnt
- **WHEN** `addUserIds` eine ID enthält, die nicht im neu berechneten `add` steht
- **THEN** antwortet der Server mit HTTP 409, und weder Mitglieder noch Herkunft ändern sich

#### Scenario: Ersteller kann sich nicht selbst entfernen
- **WHEN** `removeUserIds` die ID des Erstellers enthält
- **THEN** antwortet der Server mit HTTP 409 und ändert nichts

#### Scenario: Nicht-Ersteller wendet an
- **WHEN** ein Mitglied, das nicht Ersteller ist, `apply` aufruft
- **THEN** antwortet der Server mit HTTP 403 und ändert nichts

#### Scenario: Herkunft für Bestandsgruppe festlegen
- **WHEN** der Ersteller einer Gruppe ohne Herkunft `apply` mit `sources = [{team, T1, spieler}]` und leeren Listen aufruft
- **THEN** trägt die Gruppe danach diese Herkunft, und die Mitglieder bleiben unverändert

### Requirement: Aktualisieren-Modal

Das Teilnehmer-Modal einer Gruppen-Konversation SHALL für den Ersteller einen Knopf „Aktualisieren" zeigen, für alle anderen nicht. Er öffnet ein eigenes Modal. Es lädt die Vorschau zunächst **ohne** `sources`, also mit der gespeicherten Herkunft, und zeigt:
- einem kurzen Erklärtext: Die Gruppe wird mit den gewählten Standard-Gruppen abgeglichen. Fehlende Personen werden ergänzt, Personen, die nicht mehr dazugehören (z. B. ausgetreten oder nicht mehr im Kader), werden entfernt. Die Auswahl lässt sich unten anpassen, geändert wird erst mit „OK".
- dem Abschnitt **Abgleich mit**: die Herkunfts-Kacheln als entfernbare Chips plus einer Auswahl weiterer sichtbarer Kacheln mit Überlappung („11 von 12 schon drin"). Bei einer Gruppe ohne Herkunft ist keine Kachel vorausgewählt, und ein Hinweis erklärt, dass die Herkunft einmalig festgelegt werden muss.
- den Listen **Hinzufügen** und **Entfernen** mit je einer Checkbox pro Person, anfangs alle angehakt. Jede Änderung der Kacheln lädt die Vorschau neu. Bereits abgewählte Personen bleiben abgewählt, solange sie in der neuen Vorschau noch vorkommen.
- bei `blocked` einen Hinweis an der betroffenen Kachel („nicht sichtbar" bzw. „derzeit leer") und ein deaktiviertes „OK", bis die Kachel entfernt ist
- den Aktionen **Abbrechen** (schließt ohne Änderung) und **OK** (ruft `apply` mit den angehakten Personen und den gewählten Kacheln auf, schließt bei Erfolg und lädt die Teilnehmerliste neu). Antwortet der Server mit 409 `sync_stale`, SHALL das Modal die Vorschau neu laden und einen Hinweis zeigen, statt zu schließen.

#### Scenario: Nicht-Ersteller sieht keinen Knopf
- **WHEN** ein Mitglied, das nicht Ersteller ist, das Teilnehmer-Modal öffnet
- **THEN** gibt es keinen Knopf „Aktualisieren"

#### Scenario: Abbrechen ändert nichts
- **WHEN** der Ersteller das Modal öffnet, Personen abwählt und „Abbrechen" wählt
- **THEN** wird `apply` nicht aufgerufen

#### Scenario: OK schickt nur angehakte Personen
- **WHEN** der Ersteller in „Entfernen" eine von zwei Personen abwählt und „OK" wählt
- **THEN** enthält `removeUserIds` nur die angehakte Person

#### Scenario: Bestandsgruppe ohne Vorauswahl
- **WHEN** der Ersteller das Modal für eine Gruppe ohne Herkunft öffnet
- **THEN** ist keine Kachel gewählt, „Hinzufügen" und „Entfernen" sind leer, und die Kacheln zeigen ihre Überlappung
