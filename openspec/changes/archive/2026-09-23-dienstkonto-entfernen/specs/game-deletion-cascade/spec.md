# Spec Delta

## REMOVED Requirements

### Requirement: Konto-Konsistenz bei Cascade-Delete
**Reason**: Das Dienstkonto (`duty_accounts`) wird entfernt; es gibt kein gespeichertes Stunden-Konto mehr, das nachgezogen werden müsste. Die Dienst-Bilanz rechnet live aus `duty_slots`/`duty_assignments` und ist nach dem Löschen eines Termins automatisch korrekt.
**Migration**: Keine — die Dienst-Bilanz (`dutyfairness`) liefert die geleisteten Stunden ohne gespeicherten Zwischenstand.
