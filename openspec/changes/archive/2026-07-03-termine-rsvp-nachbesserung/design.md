## Context

Der Change `rsvp-defaults-per-rolle` führte zwei Rollen-Voreinstellungen (`rsvp_default_players`, `rsvp_default_extended` ∈ `confirmed|declined|none`) plus eine Konflikt-Sperre ein: `declined` durfte nicht mit `rsvp_require_reason=1` kombiniert werden (UI `disabled`/Tooltip + Backend-400 `invalid_rsvp_settings`). Der Change `termine-trainer-rsvp` machte Trainer auf der **Detailseite** RSVP-fähig (Default confirmed, kein Header-Zähler, keine Anwesenheit).

Zwei Praxis-Beobachtungen führen zu diesem Nachbesserungs-Change.

## Goals / Non-Goals

**Goals:**
- Die Konflikt-Sperre ersatzlos entfernen; `declined` und `rsvp_require_reason` sind frei kombinierbar (Backend + UI).
- Trainer können auf `/termine` (Kartenliste) zu-/absagen, aber **nur** bei Terminen von Teams, deren Trainer sie sind.
- Karte und Detailseite zeigen für Trainer denselben Default (`confirmed`).

**Non-Goals:**
- Serverseitige Erzwingung von `rsvp_require_reason` (bewusst weiter Frontend-only, unverändert).
- Trainer in Header-Zähler aufnehmen (bleibt spieler-orientiert).
- Neue Voreinstellungs-Spalte oder Route für Trainer.

## Decisions

### Konflikt-Sperre ist semantisch unnötig → ersatzlos entfernen

Die beiden Einstellungen wirken auf disjunkte Gruppen:

```
  rsvp_default_*='declined'          rsvp_require_reason=1
  ──────────────────────────         ─────────────────────────
  Mitglieder, die NICHT reagieren    Mitglieder, die AKTIV
  → virtuelle Absage                   „Absagen"/„Vielleicht" klicken
  → keine response-Zeile             → Grund-Eingabe wird Pflicht
  → nie ein Grund erfragt              (im openReasonModal)
            └────── überschneiden sich nie ──────┘
```

Eine virtuelle Default-Absage durchläuft den Grund-Pfad nie. Zudem erzwingt der Server `rsvp_require_reason` ohnehin nicht (die Respond-Handler prüfen keinen leeren Grund) — die 400-Sperre bewacht also eine Regel, die sonst nirgends greift. Damit ist auch „beide Rollen `declined` + `require_reason=1`" widerspruchsfrei.

- **Alternative**: schmaler Restschutz gegen den Fall „beide Rollen declined". **Abgelehnt**, weil auch dieser Fall stimmig ist (ein default-abgesagtes Mitglied kann aktiv absagen und dann einen Grund liefern).

### Trainer-Zugehörigkeit über den vorhandenen `my_rsvp`-Default abbilden (Option A)

Statt einer neuen pro-Termin-Flagge nutzen wir das bestehende `my_rsvp`-Feld: `ListSessions`/`ListMyGames` bekommen einen zusätzlichen `CASE`-Zweig analog zu `inRegularKader`/`inExtendedKader` — ist der aufrufende User Trainer des Team-Kaders (`kader_trainers`) und hat keine explizite Response, liefert die Query `my_rsvp='confirmed'`. Das Frontend zeigt die RSVP-Buttons genau dann, wenn `my_rsvp` nicht-null ist (Teilnahme-Signal).

- **Warum**: automatische, korrekte Team-Abgrenzung — ein Vorstand auf einem fremden Team-Termin bleibt `my_rsvp=null` → keine Buttons. Kein neues Feld, konsistent zur Detailseite. Priorität: explizite Response > Stammkader-Default > Erweitert-Default > **Trainer-confirmed** > null.
- **Alternative B** (explizites `is_own_trainer`-Feld): mehr API-Fläche, kein Mehrwert gegenüber A.
- **Alternative C** (nur Frontend, `!isTrainer` streichen ohne Backend): `my_rsvp` bliebe für Trainer null → keine sinnvolle Anzeige und keine saubere Abgrenzung. Abgelehnt.

### Button-Toggle-Logik der Karte

Die „Zusagen"-Aktion darf für Trainer nicht an `rsvp_default_players` gekoppelt sein (das ist die Spieler-Voreinstellung). Für Trainer verhält sich „Zusagen" wie ein einfaches Setzen auf `confirmed`; die bestehende Spieler-Toggle-Bedingung bleibt für Spieler erhalten.

## Risks / Trade-offs

- **[Reason-Erzwingung bleibt Frontend-only]** — Durch Entfernen der Sperre wird sichtbarer, dass `rsvp_require_reason` serverseitig nicht erzwungen wird. Das ist Bestandsverhalten und **nicht** Teil dieses Changes; nur dokumentiert.
- **[`my_rsvp`-Default für Trainer]** — Ein Trainer, der bewusst nichts anklickt, erscheint auf der Karte als „zugesagt". Das ist exakt die etablierte Trainer-Semantik der Detailseite (Opt-out) und damit konsistent gewollt.
- **[Query-Komplexität]** — Ein zusätzlicher `EXISTS`-Zweig pro Listen-Query. Performance unkritisch (wenige Kader pro User); durch Tests abgedeckt.
