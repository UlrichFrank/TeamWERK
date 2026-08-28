## ADDED Requirements

### Requirement: Ein Dienst kann bei Ablösung enden

Ein Diensttyp und eine Vorlagen-Zeile SHALL ein Kennzeichen `end_at_next_duty` tragen
können. Ist es gesetzt **und** steht der Dauer-Modus auf `dynamisch`, SHALL das Ende des
erzeugten Slots der **frühere** der beiden folgenden Zeitpunkte sein:

- der Start des nächsten gleichartigen Dienstes am selben Spieltag, oder
- das aus End-Anker und End-Versatz aufgelöste Ende (der **Deckel**).

Existiert kein solcher Nachfolger, SHALL der Deckel unverändert gelten. Die Ablösung
SHALL ein Ende nur nach vorn ziehen können, nie nach hinten.

**Gleichartig** SHALL bedeuten: derselbe Diensttyp, unter dem der Slot tatsächlich
entsteht — also nach einer eventuellen Varianten-Reduktion. Ein Termin ohne Slot dieses
Diensttyps SHALL den Vorgänger **nicht** verlängern.

Als Nachfolger SHALL ausschließlich ein Slot in Frage kommen, dessen Startzeit **nach**
der eigenen liegt. Ein zeitgleicher oder früher liegender Slot SHALL nicht ablösen; in
diesem Fall greift der Deckel.

Die Ablösung SHALL sich auf **alle** an diesem Tag bestehenden Dienst-Slots stützen,
unabhängig davon, wie sie entstanden sind: manuell angelegte Dienste (`is_custom=1`) und
Dienste an Terminen, die von einer Massenregeneration ausgenommen wurden, lösen ebenso ab
wie automatisch erzeugte. Gekappt SHALL dagegen nur werden, was das Kennzeichen selbst
trägt — manuell angelegte Dienste behalten ihre Dauer.

Greift die Varianten-Logik, SHALL das Kennzeichen des **Varianten-Diensttyps** gelten, wie
bei Modus, End-Anker und End-Versatz auch.

Im Modus `absolut` SHALL das Kennzeichen bedeutungslos sein — es SHALL gespeichert, aber
nicht angewendet werden, damit ein Moduswechsel hin und zurück den Wert nicht verliert.

Ein gesetztes Kennzeichen SHALL **keine** zusätzliche Eingabe-Validierung nach sich
ziehen: Es kann eine Definition nicht unmöglich machen, weil es eine Dauer ausschließlich
verkürzt und nur Nachfolger nach dem eigenen Start berücksichtigt. Die gekappte Dauer
SHALL daher immer positiv bleiben, und die Kappung SHALL nie dazu führen, dass ein bereits
entstandener Slot wieder entfällt.

Die Oberfläche SHALL das Kennzeichen als Häkchen unter End-Anker und End-Versatz
anbieten und nur im Modus „Startzeit + Endzeit" zeigen. Die Vorlagen-Zeile SHALL es wie
die übrigen Item-Felder per Copy-on-pick vom Diensttyp übernehmen und danach eigenständig
führen.

Die Einzeltermin-Vorschau SHALL den Deckel zeigen. Sie rechnet gegen einen einzelnen
Termin und kann die Kette nicht kennen; ihre Angabe ist damit eine obere Schranke.

#### Scenario: Der Nachfolger löst ab

- **WHEN** an einem Spieltag zwei Slots desselben Diensttyps entstehen, der erste mit
  einem Deckel um 12:00 Uhr, der zweite mit Startzeit 11:15 Uhr
- **AND** der erste Diensttyp trägt das Kennzeichen
- **THEN** endet der erste Slot um 11:15 Uhr
- **AND** entspricht seine Dauer der Spanne von seiner Startzeit bis 11:15 Uhr

#### Scenario: Der Letzte in der Kette behält den Deckel

- **WHEN** kein weiterer Slot desselben Diensttyps an diesem Tag nach diesem beginnt
- **THEN** endet der Slot zum aufgelösten Ende aus End-Anker und End-Versatz
- **AND** ist seine Dauer identisch zu der ohne gesetztes Kennzeichen

#### Scenario: Nachfolger liegt hinter dem Deckel

- **WHEN** der nächste gleichartige Dienst erst nach dem aufgelösten Ende beginnt
- **THEN** bleibt es beim Deckel
- **AND** wird die Dauer nicht verlängert

#### Scenario: Rückwärts liegender Slot löst nicht ab

- **WHEN** ein Slot desselben Diensttyps am selben Tag **vor** dem eigenen Start beginnt
- **THEN** wird er als Ablösung nicht berücksichtigt
- **AND** greift der Deckel
- **AND** bleibt der Slot bestehen

#### Scenario: Nur derselbe Diensttyp löst ab

- **WHEN** zwischen zwei Bewirtungsdiensten ein Slot eines anderen Diensttyps beginnt
- **THEN** kappt dieser den Bewirtungsdienst nicht

#### Scenario: Heimspiel ohne gleichartigen Dienst verlängert nicht

- **WHEN** auf den letzten Bewirtungsdienst eines Tages noch zwei Heimspiele folgen, an
  denen die Rotation keinen Bewirtungsdienst zugeteilt hat
- **THEN** endet der Bewirtungsdienst an seinem Deckel
- **AND** wird er nicht bis zu diesen Spielen verlängert

#### Scenario: Manuell angelegter Dienst löst ab, wird aber nicht gekappt

- **WHEN** am selben Tag ein Dienst desselben Diensttyps von Hand angelegt wurde
  (`is_custom=1`)
- **THEN** löst er einen davorliegenden automatisch erzeugten Dienst ab
- **AND** behält er selbst seine gepflegte Dauer

#### Scenario: Ausgenommener Termin löst trotzdem ab

- **WHEN** ein Massenlauf einen Termin ausnimmt, an dem ein Slot desselben Diensttyps
  bestehen bleibt
- **THEN** zählt dieser Slot als Ablösung für die neu erzeugten Dienste des Tages

#### Scenario: Variante bestimmt das Kennzeichen

- **WHEN** die Varianten-Logik greift und der Varianten-Diensttyp das Kennzeichen **nicht**
  trägt
- **THEN** wird der erzeugte Slot nicht gekappt
- **AND** gilt allein sein aufgelöstes Ende

#### Scenario: Absoluter Modus ist nicht betroffen

- **WHEN** ein Diensttyp mit gesetztem Kennzeichen im Modus `absolut` steht
- **THEN** trägt der erzeugte Slot exakt die gepflegte Stundenzahl
- **AND** bleibt das Kennzeichen gespeichert

#### Scenario: Kappung lässt nie einen Dienst entfallen

- **WHEN** ein Slot durch die Ablösung gekappt wird
- **THEN** ist seine Dauer weiterhin größer als null
- **AND** entsteht kein `invalid_span`-Eintrag
- **AND** wird keine darauf liegende Zusage als entfernt gemeldet

#### Scenario: Häkchen nur im dynamischen Modus

- **WHEN** ein Vorstand in der Diensttyp- oder Vorlagen-Maske auf „Startzeit + Dauer"
  umschaltet
- **THEN** verschwindet das Häkchen
- **AND** bleibt der gespeicherte Wert beim Zurückschalten erhalten

## MODIFIED Requirements

### Requirement: Ein Diensttyp kann eine dynamische Dauer haben

Ein Diensttyp SHALL einen Dauer-Modus tragen: `absolut` oder `dynamisch`. Im Modus
`absolut` SHALL sich nichts gegenüber dem bisherigen Verhalten ändern — die Dauer ist die
gepflegte Stundenzahl.

Im Modus `dynamisch` SHALL das Ende des Dienstes über einen **End-Anker** und einen
**End-Versatz** bestimmt werden, mit denselben zwei Ankern wie der Start: `start`
(Anpfiff) und `end` (Spielende). Der Versatz SHALL in beide Richtungen zulässig sein.
Die Dauer SHALL dort ausschließlich die Differenz aus aufgelöster End- und Startzeit
sein; die gepflegte Stundenzahl SHALL keine Rolle spielen. Trägt der Diensttyp zusätzlich
das Kennzeichen `end_at_next_duty`, SHALL das so aufgelöste Ende als **Deckel** wirken und
durch eine Ablösung nach vorn gezogen werden können (siehe Requirement „Ein Dienst kann
bei Ablösung enden").

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
