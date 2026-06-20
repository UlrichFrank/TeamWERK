## 1. Backend — partnerTreffpunkt in Dashboard-Payload

- [x] 1.1 `internal/dashboard/handler.go`: `paarungEntry` (oder vergleichbare Struktur) um Feld `PartnerTreffpunkt string` mit JSON-Tag `partnerTreffpunkt` erweitern
- [x] 1.2 `queryCarpoolingConfirmed` (oder zuständige Methode): SQL-Subquery ergänzen, die je nach Bieter-/Sucher-Seite des Nutzers den Treffpunkt der Gegenseite aus `mitfahrgelegenheiten` zieht
- [x] 1.3 Kinder-Paarungen (`family_links`): gleiche Logik aus Sicht des Kindes (Kind-Seite = „eigene Seite")
- [x] 1.4 NULL/leerer Treffpunkt → leerer String im JSON

## 2. Backend — Tests

- [x] 2.1 `internal/dashboard/handler_test.go`: `TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_AsBieter` — eigene Bieter-Paarung, Sucher hat Treffpunkt → `partnerTreffpunkt` enthält Sucher-Wert
- [x] 2.2 `TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_AsSucher` — eigene Sucher-Paarung, Bieter hat Treffpunkt → `partnerTreffpunkt` enthält Bieter-Wert
- [x] 2.3 `TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_Empty` — Partner hat keinen Treffpunkt → leerer String
- [x] 2.4 `TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_KindAsBieter` — Eltern-User, Kind ist Bieter → Sucher-Treffpunkt im Payload
- [x] 2.5 Bestehender Happy-Path-Test (`partnerName`-Assertion) bleibt grün

## 3. Frontend — DashboardRow-Komponente

- [ ] 3.1 `web/src/pages/DashboardPage.tsx`: `DashboardRow`-Komponente mit Spaltenraster `w-10 | Icon w-4 | flex-1 min-w-0 | ArrowRight w-4` und Props `{ to, dateISO, icon, title, subtitle, badge? }` implementieren
- [ ] 3.2 Zwei-Zeilen-Inhalt: Zeile 1 `text-sm font-medium text-brand-text truncate` + optional `badge`; Zeile 2 `text-xs text-brand-text-muted truncate`
- [ ] 3.3 Tailwind-Klassen identisch zu bestehender `MeineTermineSection`-Zeile (visuelle Pixel-Treue)

## 4. Frontend — Sektionen umbauen

- [ ] 4.1 `DashboardData`-Type: `paarungen[]` um `partnerTreffpunkt: string` erweitern
- [ ] 4.2 `MeineTermineSection` auf `DashboardRow` refaktorieren (nur Umstrukturierung, kein Inhaltswechsel; `ExtendedBadge` als `badge`-Prop)
- [ ] 4.3 `MeineDiensteSection`: Gruppen-Header entfernen; pro `mySlots[i]` eine `DashboardRow` mit `dateISO = nextGame.date`, `icon = <Check>`, `title = s.dutyTypeName`, `subtitle = "{opponent} · {s.eventTime}"`
- [ ] 4.4 `MeineDiensteSection`-Fallback `mySlots.length === 0 && openSlotsCount > 0`: eine `DashboardRow` mit Info-Icon, `title = "N offene Dienste verfügbar"`, `subtitle = opponent`
- [ ] 4.5 `MeineDiensteSection`-Dienstkonto-Toggle bleibt unverändert unterhalb
- [ ] 4.6 `FahrgemeinschaftenSection`: flache Liste aus Zusagen + offenen Gesuchen, chronologisch sortiert (innerhalb gleichen Datums: Zusagen vor Gesuchen)
- [ ] 4.7 Fahrt-Zusage-Zeile: `icon = <Check className="text-brand-success">`, `title = p.partnerName`, `subtitle = partnerTreffpunkt ? "{opponent} · {partnerTreffpunkt}" : opponent`
- [ ] 4.8 Fahrt-Gesuch-Zeile: `icon = <Search className="text-brand-text-muted">`, `title = req.requesterName`, `subtitle = treffpunkt ? "{plaetze} Plätze · {treffpunkt}" : "{plaetze} Plätze · {gameTitle}"`
- [ ] 4.9 Footer-Link „Alle Mitfahrten →" bleibt
- [ ] 4.10 Empty-State-Texte und Section-Toggle-Logik unverändert

## 5. Reihenfolge der Sektionen

- [ ] 5.1 JSX-Reihenfolge in `DashboardPage` ändern: `Termine → Dienste → Fahrt → Team`
- [ ] 5.2 `openSections`-State-Defaults unverändert lassen

## 6. Verifikation

- [ ] 6.1 `make test` grün (inkl. neuer Backend-Tests)
- [ ] 6.2 `make lint` grün
- [ ] 6.3 `pnpm -C web typecheck` grün
- [ ] 6.4 Visuelle Verifikation am laufenden System: Spaltenraster richtet sich über die drei Sektionen einheitlich aus (Browser-Devtools-Inspektion der Spaltenbreiten)
- [ ] 6.5 Manueller Klick auf Termin/Dienst/Fahrt-Zeile führt zu erwartetem Ziel
- [ ] 6.6 `openspec validate dashboard-row-layout-vereinheitlichen` grün

## 7. Archivierung

- [ ] 7.1 Proposal nach erfolgreichem Deploy archivieren (`openspec archive`)
