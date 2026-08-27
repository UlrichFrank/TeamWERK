## ADDED Requirements

### Requirement: Ein Diensttyp kann eine dynamische Dauer haben

Ein Diensttyp SHALL einen Dauer-Modus tragen: `absolut` oder `dynamisch`. Im Modus
`absolut` SHALL sich nichts gegenüber dem bisherigen Verhalten ändern — die Dauer ist die
gepflegte Stundenzahl.

Im Modus `dynamisch` SHALL das Ende des Dienstes über einen **End-Anker** und einen
**End-Versatz** bestimmt werden, mit denselben zwei Ankern wie der Start: `start`
(Anpfiff) und `end` (Spielende). Der Versatz SHALL in beide Richtungen zulässig sein.

Die Auflösung des End-Ankers SHALL identisch zur Auflösung des Start-Ankers sein:
`end` verwendet die gepflegte Endzeit des Termins, und andernfalls den Anpfiff zuzüglich
der ermittelten Spieldauer.

Die Vorlagen-Zeile SHALL Modus, End-Anker und End-Versatz wie die übrigen Item-Felder
per Copy-on-pick vom Diensttyp übernehmen und danach eigenständig führen.

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

### Requirement: Eine unauflösbare dynamische Dauer fällt auf die absolute zurück

Ergibt die dynamische Auflösung keine positive Dauer — weil die errechnete Endzeit vor der
Startzeit liegt —, SHALL der Slot trotzdem entstehen und die gepflegte absolute Dauer
tragen. Die Stundenzahl SHALL deshalb auch im Modus `dynamisch` pflegbar bleiben.

Ein Slot SHALL nach jedem Regen-Lauf eine Dauer größer als null tragen, unabhängig davon,
wie Anker und Versätze gesetzt sind.

#### Scenario: Endzeit läge vor der Startzeit

- **WHEN** ein dynamischer Diensttyp bei Anpfiff −30 min startet und bei Anpfiff −60 min
  enden würde
- **THEN** entsteht der Slot
- **AND** trägt er die gepflegte absolute Dauer
- **AND** fällt kein Dienst aus dem Plan

### Requirement: Manuell angelegte Dienste bleiben absolut

Ein per `POST /api/duty-slots` angelegter Dienst SHALL eine absolute Dauer tragen. Der
Dauer-Modus SHALL für solche Slots nicht angeboten werden, da sie `is_custom=1` tragen und
vom Regen nie erneuert werden — eine als dynamisch etikettierte Dauer würde dort nie
nachgeführt.

Das Anlege-Formular SHALL die Dauer aus einer dynamischen Typ-Definition jedoch berechnen
und vorbelegen dürfen.

#### Scenario: Dienst mit dynamischem Typ von Hand hinzufügen

- **WHEN** ein Vorstand über „+ Dienst hinzufügen" einen Diensttyp im Modus `dynamisch`
  auswählt
- **THEN** wird die Dauer aus Anker und Versätzen gegen diesen Termin berechnet vorbelegt
- **AND** ist der so entstandene Slot danach eine feste Zahl, die der Regen nicht anrührt
