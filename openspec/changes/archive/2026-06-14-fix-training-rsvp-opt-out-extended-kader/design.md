## Context

`training_sessions.rsvp_opt_out = 1` signalisiert, dass alle Mitglieder als „bestätigt" gelten, sofern sie nicht explizit absagen. Die Scan-Schleife in `GetAttendances` setzt bei fehlendem `training_responses`-Eintrag und `rsvpOptOut == 1` das `RSVPStatus`-Feld auf `"confirmed"` — ohne Rücksicht auf `is_extended`.

Seit dem Fix `fix-training-attendances-is-extended` kennt der Handler das `is_extended`-Flag. Die Korrektur ist eine einfache Guard-Erweiterung.

## Goals / Non-Goals

**Goals:**
- `rsvp_opt_out` Auto-Confirm gilt ausschliesslich für `is_extended = false`
- Erweiterte Kader-Mitglieder erhalten `rsvp_status: null` wenn keine explizite Rückmeldung vorliegt

**Non-Goals:**
- Kein Konfigurationsfeld „opt-out auch für erweiterten Kader" — das ist bewusst aus dem Scope (Option B bleibt zukünftiges Feature)
- Keine Frontend-Änderung

## Decisions

### Guard statt eigene Logik

Die Bedingung `rsvpOptOut == 1` wird um `&& !item.IsExtended` erweitert. Kein neuer Codepfad, keine neue Variable. Die Semantik ist vollständig durch den bestehenden `is_extended`-Wert ausgedrückt.

## Risks / Trade-offs

- **Kein Breaking Change**: Die Änderung betrifft nur den Fall `rsvp_opt_out = 1` + kein expliziter Response. Wer einen `training_responses`-Eintrag hat, ist unberührt.
- **Trainer-Sicht**: Erweiterte Kader-Mitglieder erscheinen in der Anwesenheitsliste ohne RSVP-Status (Strich). Das ist korrektes Verhalten — sie wurden nicht zur Anmeldung aufgefordert.
