## 1. Backend — Guard in GetAttendances

- [x] 1.1 Bedingung `rsvpOptOut == 1` in der Scan-Schleife von `GetAttendances` um `&& !item.IsExtended` erweitern (`internal/trainings/handler.go`)

## 2. Test

- [x] 2.1 Test `TestGetAttendances_OptOut_NotAppliedToExtended`: Session mit `rsvp_opt_out=1` — primäres Kader-Mitglied ohne Response → `rsvp_status="confirmed"`; erweitertes Kader-Mitglied ohne Response → `rsvp_status=null`
