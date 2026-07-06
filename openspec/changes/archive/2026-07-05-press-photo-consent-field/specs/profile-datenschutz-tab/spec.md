## MODIFIED Requirements

### Requirement: DSGVO-Anzeige (read-only) im Datenschutz-Tab

Der Datenschutz-Tab SHALL die DSGVO-Einwilligungen des eigenen Members anzeigen:
„Datenverarbeitung eingewilligt", „Datenweitergabe eingewilligt" und „Foto-Veröffentlichung
eingewilligt" — je mit Status (Ja/Nein) und Datum (falls vorhanden). Zu **jedem** der drei
Schalter SHALL ein kurzer Erklärtext angezeigt werden, der beschreibt, was mit der Einwilligung
verbunden ist (Verarbeitung der Mitgliedsdaten; Weitergabe an Dritte; Veröffentlichung von Fotos
auf öffentlichen Kanälen des Vereins). Das visuelle Control SHALL dem Stil von
`MemberDatenschutzTab` im Admin entsprechen, ist aber **gesperrt** (read-only). Änderungen SHALL
weiterhin nur über den bestehenden Draft-Workflow beantragt werden.

#### Scenario: DSGVO-Status wird angezeigt

- **WHEN** der Tab geladen wird
- **THEN** sieht der Nutzer den Status von `dsgvo_verarbeitung`, `dsgvo_weitergabe` und `foto_veroeffentlichung` mit zugehörigen `_date`-Werten
- **AND** alle drei Controls sind nicht bedienbar (kein Schreiben aus diesem Tab heraus)

#### Scenario: Erklärtext je Schalter

- **WHEN** der DSGVO-Block gerendert wird
- **THEN** steht unter jedem der drei Schalter ein erläuternder Text zu seiner Bedeutung

#### Scenario: Hinweis auf Änderungsweg

- **WHEN** der Tab gerendert wird
- **THEN** zeigt er einen Hinweis, dass Änderungen über den Draft-Workflow beantragt werden müssen
