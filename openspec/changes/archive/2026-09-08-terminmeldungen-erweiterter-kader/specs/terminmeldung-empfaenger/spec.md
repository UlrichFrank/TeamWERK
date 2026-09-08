## Purpose

Legt fest, wer eine Benachrichtigung zu einem Mannschaftstermin bekommt — einmal, für
alle Meldungen zu Spielen, Trainings, Erinnerungen und Termin-Hinweisen — und grenzt
diese Menge gegen Dienstpflicht und Terminsichtbarkeit ab.

## ADDED Requirements

### Requirement: Empfängermenge einer Terminmeldung

Das System SHALL für **jede** Benachrichtigung zu einem Mannschaftstermin dieselbe
Empfängermenge auflösen. Sie besteht aus:

- den Mitgliedern des **Stammkaders** der betroffenen Mannschaft,
- den Mitgliedern des **erweiterten Kaders** derselben Mannschaft,
- den **Elternteilen beider Gruppen** (über `family_links`),
- den **Trainern** des Kaders (`kader_trainers`) — ohne Eltern-Zweig: sie stehen
  in ihrer Funktion als Trainer in der Menge, nicht als Kind,

jeweils bezogen auf die **aktive Saison** und beschränkt auf Mitglieder mit verknüpftem
Nutzerkonto. Ein Nutzer SHALL höchstens einmal in der Menge erscheinen, auch wenn er über
mehrere Mannschaften, über beide Kader-Listen oder zugleich als Elternteil erfasst ist.

Die Regel SHALL an **genau einer** Stelle im System implementiert sein. Eine Domäne SHALL
NICHT ihre eigene Auflösung mitbringen — divergierende Kopien haben in der Vergangenheit
dazu geführt, dass dieselbe Personengruppe je nach Terminart benachrichtigt wurde oder
nicht.

Der **Auslöser** einer Änderung SHALL NICHT herausgefiltert werden: wer einen Termin
anlegt, ändert oder absagt, bekommt die eigene Meldung mit. Sie ist die Bestätigung, dass
die Meldung rausgegangen ist, und zeigt im Wortlaut, was die Mannschaft gelesen hat.

Ein Termin ohne Mannschaftsbezug SHALL eine leere Menge liefern; der Aufrufer versendet
dann nichts.

#### Scenario: Erweiterter Kader ist Teil der Empfängermenge
- **WHEN** eine Meldung zu einem Termin einer Mannschaft versendet wird
- **THEN** enthält die Empfängermenge die Nutzerkonten der Mitglieder des erweiterten Kaders dieser Mannschaft in der aktiven Saison

#### Scenario: Eltern des erweiterten Kaders sind Teil der Empfängermenge
- **WHEN** ein Mitglied des erweiterten Kaders über `family_links` mit einem Elternteil verknüpft ist
- **THEN** enthält die Empfängermenge auch das Nutzerkonto dieses Elternteils

#### Scenario: Doppelte Zugehörigkeit erzeugt keinen doppelten Empfänger
- **WHEN** ein Mitglied sowohl im Stammkader als auch im erweiterten Kader derselben Mannschaft steht, oder ein Termin mehrere Mannschaften betrifft, in denen dasselbe Mitglied steht
- **THEN** erscheint sein Nutzerkonto genau einmal in der Empfängermenge

#### Scenario: Trainer des Kaders ist Teil der Empfängermenge
- **WHEN** eine Meldung zu einem Termin einer Mannschaft versendet wird
- **THEN** enthält die Empfängermenge die Nutzerkonten der Trainer des Kaders dieser Mannschaft in der aktiven Saison
- **THEN** enthält sie NICHT die Elternteile dieser Trainer

#### Scenario: Auslöser bekommt die eigene Meldung
- **WHEN** ein Trainer einen Termin seiner Mannschaft anlegt, ändert oder absagt
- **THEN** ist sein eigenes Nutzerkonto Teil der Empfängermenge dieser Meldung

#### Scenario: Mitglied ohne Nutzerkonto erzeugt keinen Empfänger
- **WHEN** ein Kader-Mitglied kein verknüpftes Nutzerkonto hat
- **THEN** erscheint es nicht in der Empfängermenge, ohne dass die Auflösung fehlschlägt

#### Scenario: Kader einer nicht-aktiven Saison zählt nicht
- **WHEN** ein Mitglied nur im Kader einer vergangenen Saison derselben Mannschaft steht
- **THEN** erscheint es nicht in der Empfängermenge

### Requirement: Abgrenzung zu Dienstmeldungen und Sichtbarkeit

Die Empfängermenge einer Terminmeldung SHALL NICHT für Meldungen der Dienstbörse gelten.
Dienstmeldungen (offene Dienste, Dienst-Erinnerungen, entfallene Dienste) SHALL weiterhin
den Stammkader und dessen Eltern ansprechen: die Dienstpflicht folgt der
Kader-Zugehörigkeit, und wer im erweiterten Kader aushilft, schuldet dem Verein keine
Dienststunden.

Die Empfängermenge SHALL NICHT als Sichtbarkeitsregel verwendet werden. Wer einen Termin
sehen darf, entscheidet die bestehende Terminsichtbarkeit; die Empfängermenge entscheidet
ausschließlich, wer aktiv benachrichtigt wird.

#### Scenario: Dienstmeldung erreicht den erweiterten Kader nicht
- **WHEN** ein Dienst-Slot ausgeschrieben wird oder eine Dienst-Erinnerung fällig ist
- **THEN** erhalten Mitglieder, die nur im erweiterten Kader stehen, keine Benachrichtigung

#### Scenario: Benachrichtigung ersetzt keine Berechtigungsprüfung
- **WHEN** ein Nutzer eine Terminmeldung erhält
- **THEN** bleibt sein Zugriff auf den Termin und dessen Detaildaten unverändert an die bestehende Terminsichtbarkeit gebunden
