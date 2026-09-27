## MODIFIED Requirements

### Requirement: Abgrenzung zu Dienstmeldungen und Sichtbarkeit

Die Empfängermenge einer Terminmeldung SHALL NICHT für Meldungen der Dienstbörse gelten.
Dienstmeldungen über **offene** Dienste (neu ausgeschriebene Slots, Erinnerungen an noch
offene Slots) SHALL weiterhin den Stammkader und dessen Eltern ansprechen: die Dienstpflicht
folgt der Kader-Zugehörigkeit, und wer im erweiterten Kader aushilft, schuldet dem Verein
keine Dienststunden. Die Meldung über einen **entfallenen** Dienst SHALL dagegen alle
Eingetragenen des Slots erreichen — auch eine Aushilfe aus dem erweiterten Kader, die sich
eingetragen hat.

Die Empfängermenge SHALL NICHT als Sichtbarkeitsregel verwendet werden. Wer einen Termin
sehen darf, entscheidet die bestehende Terminsichtbarkeit; die Empfängermenge entscheidet
ausschließlich, wer aktiv benachrichtigt wird.

#### Scenario: Dienstmeldung erreicht den erweiterten Kader nicht
- **WHEN** ein Dienst-Slot ausgeschrieben wird oder eine Dienst-Erinnerung fällig ist
- **THEN** erhalten Mitglieder, die nur im erweiterten Kader stehen, keine Benachrichtigung

#### Scenario: Benachrichtigung ersetzt keine Berechtigungsprüfung
- **WHEN** ein Nutzer eine Terminmeldung erhält
- **THEN** bleibt sein Zugriff auf den Termin und dessen Detaildaten unverändert an die bestehende Terminsichtbarkeit gebunden

#### Scenario: Eingetragene Aushilfe erfährt von der Absage
- **WHEN** ein Dienst-Slot gelöscht wird, auf dem ein Mitglied eingetragen ist, das nur im erweiterten Kader des Teams steht
- **THEN** erhält dieses Mitglied die Meldung über den entfallenen Dienst
