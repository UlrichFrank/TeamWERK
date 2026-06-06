## ADDED Requirements

### Requirement: App-Version im Sidebar-Footer anzeigen

Die App SHALL die aktuell laufende Server-Version (buildHash) im Sidebar-Footer unterhalb des Abmelden-Buttons anzeigen. Die Anzeige SHALL durch eine horizontale Trennlinie (gleicher Stil wie die bestehende Trennlinie über E-Mail und Abmelden) vom Rest des Footers abgesetzt sein. Im DEV-Modus (kein SSE) SHALL nichts angezeigt werden.

#### Scenario: Version erscheint nach SSE-Verbindung

- **WHEN** der Nutzer eingeloggt ist und die SSE-Verbindung das erste `__version:`-Event empfangen hat
- **THEN** zeigt der Sidebar-Footer unterhalb des Abmelden-Buttons eine Trennlinie und darunter den Text `v <hash>` an
- **THEN** ist der Text im gleichen muted-Stil wie die E-Mail-Adresse darüber (`text-brand-black/40 text-xs`)

#### Scenario: Version im DEV-Modus ausgeblendet

- **WHEN** die App im DEV-Modus läuft (`import.meta.env.DEV === true`)
- **THEN** wird keine Version und keine zusätzliche Trennlinie angezeigt

#### Scenario: Version noch nicht empfangen

- **WHEN** die SSE-Verbindung noch nicht das erste `__version:`-Event empfangen hat
- **THEN** wird keine Version angezeigt (kein Platzhalter, keine Lücke)
