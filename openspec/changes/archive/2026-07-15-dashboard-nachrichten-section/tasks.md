## 1. Datenlade-Logik

- [x] 1.1 In `DashboardPage.tsx` die ungelesenen Konversationen und Mitteilungen laden (`GET /api/chat/conversations`, `GET /api/chat/broadcasts`, `Promise.all`), clientseitig auf ungelesen filtern und auf max. 5 Einträge deckeln (neueste zuerst).
- [x] 1.2 `useChatEvents` abonnieren und bei `chat:new-message` / `chat:new-broadcast` / `chat:conversation-read` neu laden.

## 2. UI-Section

- [x] 2.1 Neue `Accordion`-Section „Nachrichten" mit passendem lucide-Icon (z.B. `MessageSquare`) ergänzen — Card-Optik/Struktur identisch zu den bestehenden vier Sections.
- [x] 2.2 Einträge als `DashboardRow` rendern: Konversation → `to="/chat"`, Mitteilung → `to="/chat?tab=broadcasts"`; Titel = Konversationsname bzw. Absender, Subtitle = Kurztext/Zeit.
- [x] 2.3 Leerzustand („Keine ungelesenen Nachrichten") und Fußzeilen-Link „Zum Chat →" (Muster wie bestehende Section-Footer).
- [x] 2.4 Nur `brand-*`-Tokens und `lucide-react`-Icons verwenden (keine Raw-Farben/Emojis).

## 3. Verifikation

- [x] 3.1 `pnpm -C web build` und `pnpm -C web lint` ohne Fehler.
- [x] 3.2 Bestehende Frontend-Tests laufen; Dashboard-Section rendert bei vorhandenen und bei null ungelesenen Einträgen korrekt (Rendertest wenn Test-Setup für `DashboardPage` vorhanden).

## Test-Anforderungen

| Route/Verhalten | Test | Erwartung |
|---|---|---|
| Kein neuer Backend-Endpunkt | — | Change ist frontend-only; keine neuen Go-Routen, kein Broadcast-Gate betroffen |
| Section mit Ungelesenem | Frontend-Rendertest (falls Setup vorhanden) | Ungelesene Konversationen/Mitteilungen erscheinen als `DashboardRow` mit korrekten `to`-Zielen |
| Section ohne Ungelesenes | Frontend-Rendertest (falls Setup vorhanden) | Leerzustand statt Einträge; „Zum Chat"-Link vorhanden |

**Garantierte Invariante:** Die Dashboard-Section liest ausschließlich (keine Mutation) und führt keinen neuen Backend-Endpunkt ein; ungelesene Nachrichten und Mitteilungen sind vom Dashboard aus in einem Klick erreichbar.
