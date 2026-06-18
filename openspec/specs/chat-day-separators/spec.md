## ADDED Requirements

### Requirement: Tageswechsel-Separator im Chat-Verlauf

Das System SHALL im offenen Chat-Verlauf vor jeder Nachricht (regulär oder System-Message) genau dann einen horizontal zentrierten Trenner rendern, wenn sich das Kalenderdatum (lokale Zeitzone) ihrer Sendezeit vom Datum der unmittelbar vorhergehenden Nachricht unterscheidet. Vor der ersten Nachricht der Liste SHALL ebenfalls ein Trenner stehen. Der Trenner SHALL das Datum der nachfolgenden (neueren) Nachricht tragen.

#### Scenario: Zwei Nachrichten am selben Tag
- **WHEN** zwei aufeinanderfolgende Nachrichten denselben lokalen Datums-Schlüssel haben
- **THEN** wird zwischen ihnen kein Separator gerendert

#### Scenario: Nachrichten an zwei verschiedenen Tagen
- **WHEN** Nachricht A am 16.06.2026 23:55 und Nachricht B am 17.06.2026 00:05 vorliegen
- **THEN** wird zwischen A und B genau ein Separator mit dem Datum von B gerendert

#### Scenario: Erste Nachricht der Liste
- **WHEN** die Liste mindestens eine Nachricht enthält
- **THEN** wird vor der ersten Nachricht ein Separator mit deren Datum gerendert

#### Scenario: Tageswechsel über eine System-Message
- **WHEN** zwischen zwei regulären Nachrichten eine System-Message (`isSystem === true`) liegt und die System-Message an einem neuen Tag liegt
- **THEN** wird vor der System-Message ein Separator gerendert; die System-Message wird danach in ihrem regulären zentrierten Stil dargestellt

#### Scenario: Kalendarische Lücke zwischen zwei Tagen
- **WHEN** Nachricht A am 10.06.2026 und Nachricht B am 14.06.2026 vorliegen, dazwischen keine weiteren Nachrichten existieren
- **THEN** wird genau ein Separator zwischen A und B gerendert, mit dem Datum von B; es werden keine Separatoren für die dazwischenliegenden leeren Tage erzeugt

### Requirement: Label-Abstufung nach Distanz

Das Label des Separators SHALL nach der kalendarischen Distanz zwischen Nachrichtendatum und aktuellem lokalem Datum abgestuft werden:
- 0 Kalendertage Distanz → `"Heute"`
- 1 Kalendertag Distanz → `"Gestern"`
- ≥ 2 Kalendertage Distanz → Wochentag, Tag, Monat und Jahr in der Locale `de-DE` (z.B. `"Mittwoch, 15. April 2026"`)

Die Distanz SHALL über lokale Mitternachts-Grenzen berechnet werden, nicht als 24-Stunden-Differenz.

#### Scenario: Nachricht von heute
- **WHEN** `now = 2026-06-18 14:00` und `messageDate = 2026-06-18 09:30`
- **THEN** ergibt das Label `"Heute"`

#### Scenario: Nachricht von gestern, kurze Distanz
- **WHEN** `now = 2026-06-18 00:30` und `messageDate = 2026-06-17 23:30` (Distanz < 1 Stunde, aber Tageswechsel)
- **THEN** ergibt das Label `"Gestern"`

#### Scenario: Nachricht vor zwei Tagen
- **WHEN** `now = 2026-06-18` und `messageDate = 2026-06-16 12:00`
- **THEN** ergibt das Label `"Dienstag, 16. Juni 2026"`

#### Scenario: Nachricht aus dem Vorjahr
- **WHEN** `now = 2026-06-18` und `messageDate = 2025-12-24 18:00`
- **THEN** ergibt das Label `"Mittwoch, 24. Dezember 2025"`

#### Scenario: Sommerzeit-Übergang
- **WHEN** zwischen `now` und `messageDate` ein Wechsel der lokalen Sommerzeit (Europe/Berlin) liegt
- **THEN** wird die kalendarische Distanz korrekt berechnet (Anzahl tatsächlicher Kalendertage, nicht beeinflusst durch ±1h DST-Verschiebung)

### Requirement: Darstellung des Separators

Der Separator SHALL als horizontaler Hairline-Divider mit zentriertem Label dargestellt werden — links und rechts dünne Linien (1px, `brand-border-subtle`), in der Mitte das Datums-Label (`text-xs`, `brand-text-muted`). Der Separator SHALL keinen Hintergrund, keine Pille und keinen Rahmen tragen.

#### Scenario: Visuelle Struktur
- **WHEN** ein Separator gerendert wird
- **THEN** besteht er aus einem Flex-Container mit zwei `flex-1`-Hairlines und einem zentralen Label-Element

#### Scenario: Farb-Token
- **WHEN** ein Separator gerendert wird
- **THEN** nutzt er `brand-border-subtle` für die Linien und `brand-text-muted` für das Label — keine Raw-Tailwind-Farben

### Requirement: Bubble-Timestamp bleibt schlank

Inline-Timestamps an Nachrichten-Bubbles SHALL weiterhin ausschließlich Stunden und Minuten zeigen (`HH:MM` in Locale `de-DE`). Datum oder Wochentag SHALL nicht an der Bubble angezeigt werden, unabhängig vom Alter der Nachricht — die Tagesinformation kommt ausschließlich vom Separator.

#### Scenario: Heutige Nachricht
- **WHEN** eine Bubble für eine heutige Nachricht gerendert wird
- **THEN** zeigt sie unter dem Bubble-Body `"14:23"` (Beispiel) ohne Datums-Präfix

#### Scenario: Mehrere Tage alte Nachricht
- **WHEN** eine Bubble für eine Nachricht vom 16.06.2026 gerendert wird, während heute der 18.06.2026 ist
- **THEN** zeigt sie unter dem Bubble-Body `"14:23"` (Beispiel) ohne Datums-Präfix; der Separator darüber trägt das Datum
