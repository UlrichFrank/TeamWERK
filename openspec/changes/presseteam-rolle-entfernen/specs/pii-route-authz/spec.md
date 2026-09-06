## REMOVED Requirements

### Requirement: duties — Spielbericht-Slot-Guard beschränkt auf Presseteam/Admin
**Reason**: Die System-Rolle `presseteam` entfällt ersatzlos; ein Guard, der nur noch
`admin` von allen anderen trennt, würde den Dienst für seine eigentliche Zielgruppe sperren.
**Migration**: Der Guard entfällt. Das Ziehen eines Spielbericht-Slots folgt dem regulären
Claim-Pfad — inklusive Proxy-Claim durch ein Elternteil.
