## 1. Hook-Interface anpassen

- [ ] 1.1 `useVersionCheck.ts`: Rückgabetyp auf `{ updateAvailable: boolean; version: string | null }` ändern
- [ ] 1.2 `useVersionCheck.ts`: `version`-State hinzufügen, beim ersten `__version:`-Event setzen
- [ ] 1.3 `useVersionCheck.ts`: `onerror`-Handler entfernen (SSE reconnectet automatisch)

## 2. Konsumenten aktualisieren

- [ ] 2.1 `App.tsx`: Destructuring von `useVersionCheck()` auf `{ updateAvailable: sseUpdateAvailable }` anpassen
- [ ] 2.2 `AppShell.tsx`: `useVersionCheck()` aufrufen und `version` konsumieren

## 3. Version im Sidebar-Footer anzeigen

- [ ] 3.1 `AppShell.tsx`: Unterhalb des bestehenden Footer-Blocks (`px-4 py-4 border-t …`) einen zweiten Block mit `border-t border-brand-black/10` und `text-xs text-brand-black/40` einfügen, der `v <version>` anzeigt — nur wenn `version !== null`
