# parked-audit/

Proposals, die aus einem Audit hervorgegangen sind und später umgesetzt werden sollen.
Kein aktiver Change im Sinne des OpenSpec-Workflows (`openspec list` zeigt sie deshalb
bewusst nicht als offene Arbeit) — liegen hier zur Wiedervorlage. Bewusst **außerhalb** von
`openspec/changes/`: die OpenSpec-CLI behandelt jedes Verzeichnis direkt unter `changes/` als
Change-Kandidaten, und dieses README am Wurzelverzeichnis verhinderte, dass die CLI-Heuristik
für „Namespace-Ordner um verschachtelte Changes" griff — `openspec validate --all` meldete
stattdessen einen Fehler („No deltas found").

Enthält aktuell: `vault-keypair-strength/`.
