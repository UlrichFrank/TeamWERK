# Spec Delta

## ADDED Requirements

### Requirement: Fokus-Parameter öffnet einen Termin
Die KalenderPage SHALL zusätzlich zu `?date=` einen optionalen Query-Parameter `focus=<type>-<id>` mit `type ∈ {game, training}` und numerischer `id` akzeptieren. Bei gültigem `focus` MUSS die Seite nach dem Laden:

1. den Monat aus `date` zeigen (bestehendes Verhalten des `date`-Parameters),
2. den Termin-Dialog des fokussierten Termins öffnen — mit denselben Inhalten und Aktionen wie beim Klick auf den Termin im Kalender,
3. den Termin im Monatsgitter kurz (ca. 2 Sekunden) visuell hervorheben, sofern er dort sichtbar ist,
4. `focus` danach aus der URL entfernen (History-Replace), sodass Neuladen oder Zurück-Navigation den Dialog nicht erneut öffnen; `date` bleibt erhalten.

Der Dialog SHALL auch dann öffnen, wenn der Termin durch einen aktiven Kalender-Filter (Typ, Mannschaft, Textfilter) im Gitter ausgeblendet ist oder nicht in den bereits geladenen Termindaten steht, solange der Nutzer ihn über die bestehenden Lese-Routen sehen darf.

#### Scenario: Deep-Link auf ein Spiel
- **WHEN** ein Nutzer `/kalender?date=2026-10-11&focus=game-17` öffnet
- **THEN** zeigt der Kalender Oktober 2026
- **THEN** ist der Termin-Dialog für Spiel 17 geöffnet
- **THEN** enthält die URL danach `date=2026-10-11`, aber kein `focus`

#### Scenario: Deep-Link auf ein Training
- **WHEN** ein Nutzer `/kalender?date=2026-10-14&focus=training-42` öffnet
- **THEN** ist der Termin-Dialog für Training 42 geöffnet

#### Scenario: Dialog schließen
- **WHEN** der per `focus` geöffnete Dialog geschlossen wird
- **THEN** bleibt der Kalender im Monat des Termins
- **THEN** öffnet ein Neuladen der Seite den Dialog nicht erneut

#### Scenario: Termin nicht verfügbar
- **WHEN** ein Nutzer `/kalender?date=2026-10-11&focus=game-99999` öffnet und das Spiel existiert nicht oder ist für ihn nicht sichtbar
- **THEN** zeigt die Seite den Hinweis „Dieser Termin ist nicht verfügbar“
- **THEN** zeigt sie ansonsten den Oktober 2026 normal, ohne Dialog

#### Scenario: Ungültiges Fokus-Format
- **WHEN** ein Nutzer `/kalender?date=2026-10-11&focus=foobar` öffnet
- **THEN** wird `focus` ignoriert und der Oktober 2026 normal angezeigt
