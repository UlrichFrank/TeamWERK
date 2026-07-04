## ADDED Requirements

### Requirement: Virtualisiertes Rendering langer Listen

Das Frontend SHALL lange Listen (Mitglieder, Duty-Slots, Chat-Historie) so rendern, dass nur die im Viewport sichtbaren Zeilen (zuzüglich eines kleinen Puffers) im DOM liegen. Beim Scrollen SHALL das Frontend Zeilen austauschen, statt die gesamte Liste dauerhaft zu materialisieren. Es SHALL kein Element dauerhaft ausgeblendet werden — jede Zeile bleibt durch Scrollen erreichbar.

#### Scenario: Nur sichtbare Zeilen im DOM

- **WHEN** eine Liste deutlich mehr Einträge als die Viewport-Höhe hat
- **THEN** sind nur die sichtbaren Zeilen (plus Puffer) im DOM
- **AND** Scrollen tauscht die gerenderten Zeilen aus, ohne Einträge zu verlieren

### Requirement: Geladener Paginierungs-Zustand bleibt über Live-Updates erhalten

Das Frontend SHALL bei einem Live-Update-Event in einer paginierten Ansicht (insbesondere `VideosPage`) die bereits geladenen Seiten und die Scroll-Position NICHT verwerfen. Betroffene Elemente SHALL per ID im vorhandenen Bestand aktualisiert oder entfernt werden; neu hinzugekommene Elemente SHALL nachladbar angezeigt werden (z. B. Hinweis-Chip), ohne die Liste zurückzusetzen.

#### Scenario: Kein Reset auf Seite 0 bei Live-Update

- **WHEN** in `VideosPage` mehrere Seiten geladen sind und ein `video-updated`- oder `video-ready`-Event eintrifft
- **THEN** bleiben die geladenen Seiten und die Scroll-Position erhalten
- **AND** das betroffene Element wird per ID im Bestand aktualisiert

#### Scenario: Neue Elemente werden angeboten, nicht erzwungen

- **WHEN** ein `video-queued`-Event eintrifft
- **THEN** wird ein Hinweis auf neue Einträge angezeigt
- **AND** die Liste wird erst auf Nutzeraktion nachgeladen

### Requirement: On-Demand-Laden aufklappbarer Inhalte

Das Frontend SHALL Inhalte, die hinter einer Aufklapp- oder Fokus-Interaktion liegen (z. B. Rosters nicht-fokussierter Teams), erst beim Sichtbarwerden laden und nicht bereits beim Mount vorab abrufen. Innerhalb der Session bereits geladene Inhalte SHALL das Frontend behalten und nicht erneut abrufen.

#### Scenario: Roster lädt erst beim Aufklappen

- **WHEN** `MeinTeamPage` gemountet wird und mehrere Teams existieren
- **THEN** werden nur die Rosters fokussierter/sichtbarer Teams geladen
- **AND** das Roster eines weiteren Teams wird erst bei dessen Aufklappen/Fokus geladen

#### Scenario: Kein erneuter Abruf bereits geladener Inhalte

- **WHEN** ein bereits geladenes Roster innerhalb derselben Session erneut aufgeklappt wird
- **THEN** wird es aus dem vorhandenen Zustand angezeigt, ohne erneuten HTTP-Request
