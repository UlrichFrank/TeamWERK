## ADDED Requirements

### Requirement: README als Einstieg
Der Repository MUST eine `README.md` enthalten, die TeamWERK in wenigen Sätzen erklärt, mindestens einen Screenshot zeigt, einen Quickstart (lokaler Start) gibt und Lizenz sowie weiterführende Dokumente verlinkt.

#### Scenario: Neuer Besucher versteht das Projekt
- **WHEN** eine Person die `README.md` auf GitHub öffnet
- **THEN** erkennt sie Zweck, Tech-Stack, Lizenz (AGPL-3.0) und findet Links zu Features- und Architektur-Doku
- **AND** mindestens ein Screenshot ist eingebettet

### Requirement: Fachliche Feature-Beschreibung as-built
Es MUST eine `docs/FEATURES.md` geben, die den **tatsächlich implementierten** Funktionsumfang nach Modulen beschreibt, mit je einem Screenshot, und nicht implementierte Vision-Features klar abgrenzt.

#### Scenario: Modul mit Bild dokumentiert
- **WHEN** ein Leser ein Modul (z. B. Dienste, SEPA-Beitragslauf) in `docs/FEATURES.md` aufschlägt
- **THEN** findet er eine fachliche Beschreibung, die zugehörige Rolle/Berechtigung und einen Screenshot
- **AND** der Screenshot enthält keine echten Personendaten

#### Scenario: Keine Falschversprechen
- **WHEN** ein Feature nur in der Vision (`docs/VISION.md`) existiert, aber nicht im Code
- **THEN** wird es nicht als vorhandenes Feature in `docs/FEATURES.md` dargestellt

### Requirement: Technische Architektur mit Diagrammen
Es MUST eine `docs/ARCHITECTURE.md` geben, die den realen Stack (Go/Chi/SQLite/React) beschreibt und mit Mermaid-Diagrammen die Systemtopologie, den Request-/SSE-Fluss, die Auth-Tiers und das ER-Kernmodell visualisiert.

#### Scenario: Contributor versteht den Aufbau
- **WHEN** ein Contributor `docs/ARCHITECTURE.md` liest
- **THEN** kann er den Weg einer Anfrage (Router → Auth → Handler → SQLite → Broadcast/SSE) nachvollziehen
- **AND** die Diagramme rendern nativ auf GitHub (Mermaid)

#### Scenario: Doku widerspricht nicht dem Code
- **WHEN** die Architektur-Doku den Stack benennt
- **THEN** nennt sie Go/Chi/SQLite/React (as-built), nicht den ursprünglich geplanten Laravel/Vue/PostgreSQL-Stack

### Requirement: AGPL-3.0-Lizenz
Der Repository MUST eine `LICENSE`-Datei mit dem AGPL-3.0-Volltext und einem Copyright-Vermerk enthalten. Die README MUST die Lizenz nennen.

#### Scenario: Lizenz eindeutig auffindbar
- **WHEN** GitHub die Lizenz erkennt
- **THEN** wird „AGPL-3.0" angezeigt
- **AND** die `LICENSE`-Datei enthält den unveränderten AGPL-3.0-Text mit „Copyright (C) 2026 Ulrich Frank"
