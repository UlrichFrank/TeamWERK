## 1. Frontend-Filter

- [x] 1.1 `viewMine`-State (`useState(false)`) in `MitfahrgelegenheitenPage` hinzufügen
- [x] 1.2 `filteredGames`-Variable einführen: `viewMine ? games.filter(d => [...d.biete, ...d.suche].some(e => e.isOwn)) : games`
- [x] 1.3 `tabGames` und `countForTab` auf `filteredGames` statt `response.games` umstellen
- [x] 1.4 Toggle-Button-Gruppe „Alle | Meine" im Header-Bereich neben `<h1>` einfügen (analog `DutyPage.tsx:159-172`)
- [x] 1.5 Leer-Meldung im „Meine"-Modus prüfen: wenn `tabGames.length === 0 && viewMine`, passenden Hinweis anzeigen („Du bist bei keinem Spiel eingetragen.")
