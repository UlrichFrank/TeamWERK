## ADDED Requirements

### Requirement: Konversationswechsel zeigt niemals Fremdinhalt

Beim Öffnen oder Wechseln einer Konversation SHALL der Chat-Bereich zu jedem Zeitpunkt
konsistent sein: Header und Nachrichtenliste MUST zur **selben** Konversation gehören. Es
darf keinen darstellbaren Zwischenzustand geben, in dem der Header die neu gewählte
Konversation zeigt, während die Nachrichtenliste noch Nachrichten der zuvor geöffneten
Konversation enthält.

Da die Nachrichten der Zielkonversation erst nach einem Netzwerk-Roundtrip vorliegen, MUST
die Nachrichtenliste beim Wechsel **geleert** werden, bevor der Abruf startet. Während des
Abrufs SHALL ein Ladezustand angezeigt werden, der von „Konversation ohne Nachrichten"
unterscheidbar ist. Der Ladezustand MUST auch dann enden, wenn der Abruf fehlschlägt.

Diese Invariante MUST unabhängig von Browser-Engine, Gerät und Netzwerklaufzeit gelten; sie
darf sich nicht darauf verlassen, dass der Abruf schneller ist als der nächste Frame.

Das Leeren betrifft ausschließlich den **Wechsel** der Konversation. Aktualisierungen einer
bereits geöffneten Konversation (eintreffende SSE-Nachricht, Reaktion, Austritt eines
Mitglieds, Nachladen älterer Nachrichten) MUST die Liste NICHT leeren und keinen Ladezustand
auslösen.

#### Scenario: Wechsel zwischen zwei Konversationen

- **GIVEN** ein User hat Konversation A geöffnet und deren Nachrichten sind sichtbar
- **WHEN** er in der Konversationsliste Konversation B auswählt und der Abruf der Nachrichten von B noch nicht abgeschlossen ist
- **THEN** zeigt der Header Konversation B
- **AND** es ist KEINE Nachricht aus Konversation A sichtbar
- **AND** ein Ladezustand ist sichtbar

#### Scenario: Nachrichten treffen ein

- **GIVEN** der Ladezustand aus dem vorherigen Szenario
- **WHEN** der Abruf der Nachrichten von B auflöst
- **THEN** verschwindet der Ladezustand
- **AND** die Nachrichten von B sind sichtbar und gemäß den Positionierungs-Anforderungen dieser Capability positioniert

#### Scenario: Abruf schlägt fehl

- **GIVEN** ein User wechselt zu Konversation B
- **WHEN** der Abruf der Nachrichten fehlschlägt
- **THEN** endet der Ladezustand
- **AND** es ist KEINE Nachricht aus der zuvor geöffneten Konversation sichtbar

#### Scenario: Eingehende Nachricht in der offenen Konversation leert nicht

- **GIVEN** ein User hat Konversation A geöffnet und liest im Verlauf
- **WHEN** über SSE eine neue Nachricht in A eintrifft
- **THEN** bleibt die bestehende Nachrichtenliste sichtbar
- **AND** es wird KEIN Ladezustand angezeigt
