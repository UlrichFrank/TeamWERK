## Why

Spieler und Elternteile sehen im Mitglieder-Tab aktuell eine leere Liste — die Seite ist für sie nutzlos. Gleichzeitig fehlt im Profil die Möglichkeit, verknüpfte Familienmitglieder (Kinder bzw. Elternteile) einzusehen.

## What Changes

- Der Mitglieder-Tab in der Navigation wird nur noch für `admin` und `trainer` angezeigt
- Im Profil erscheint eine neue Sektion „Meine Familie", die kontextabhängig zeigt:
  - Für `elternteil`: alle verknüpften Kinder (bereits im Backend vorhanden)
  - Für `spieler`: alle verknüpften Elternteile (neu im Backend)
- `GET /api/profile/me` wird um verknüpfte Elternteile für `spieler` erweitert

## Capabilities

### New Capabilities

- `familie-im-profil`: Anzeige verknüpfter Familienmitglieder (Kinder für elternteil, Elternteile für spieler) im Profil-Bereich

### Modified Capabilities

<!-- keine bestehenden Specs betroffen -->

## Impact

- `internal/members/handler.go`: `GetProfile` um Eltern-Abfrage für spieler ergänzen
- `web/src/pages/ProfilePage.tsx`: neue Sektion „Meine Familie"
- `web/src/components/AppShell.tsx`: Mitglieder-Nav-Eintrag per Rolle filtern
