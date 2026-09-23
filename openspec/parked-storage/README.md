# parked-storage/

Zurückgestellte Proposals, die später wieder interessant werden könnten, aber aktuell nicht
aktiv verfolgt werden. Kein aktiver Change im Sinne des OpenSpec-Workflows — liegen hier zur
Wiedervorlage statt gelöscht zu werden. Bewusst **außerhalb** von `openspec/changes/`: die
OpenSpec-CLI behandelt jedes Verzeichnis direkt unter `changes/` als Change-Kandidaten: mit
diesem README als eigener Datei am Wurzelverzeichnis griff die CLI-Heuristik für „Namespace-
Ordner um verschachtelte Changes" nicht, und `openspec validate --all` meldete einen Fehler
(„No deltas found") statt die verschachtelten Proposals einfach zu ignorieren.

Enthält aktuell: `chat-delete-fuer-mich/`, `opensource-1-pii-cleanup/`,
`opensource-2-entbranding/`, `opensource-3-dokumentation/`, `opensource-4-contribution-infra/`, `ical-feed-folgefaelle/` (Notiz, kein Proposal).
