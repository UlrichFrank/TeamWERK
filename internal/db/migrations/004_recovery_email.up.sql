-- Persistente Eltern-/Wiederherstellungs-E-Mail für Konten ohne eigene
-- Login-E-Mail (Kinderaccounts). Reine Ziel-Adresse für Passwort-Mails —
-- KEIN Unique-Index, NIE Login-/Forgot-Password-Lookup-Key.
ALTER TABLE users ADD COLUMN recovery_email TEXT;

-- email_change_tokens trägt nun einen Diskriminator: 'email' = klassischer
-- Erwachsenen-Flow (unverändert), 'recovery_email' = zweistufiger Kinder-Flow.
-- stage steuert den zweistufigen Flow: NULL = einstufig (Erwachsene),
-- 'auth' = Bestätigung an alte Adresse ausstehend, 'verify' = an neue Adresse.
ALTER TABLE email_change_tokens ADD COLUMN field TEXT NOT NULL DEFAULT 'email';
ALTER TABLE email_change_tokens ADD COLUMN stage TEXT;

-- Backfill bestehender Kinderkonten aus dem Beitrittsantrag, soweit eindeutig
-- über Name zuordenbar (genau ein passender approved Kinderantrag).
UPDATE users SET recovery_email = (
    SELECT mr.parent_email
    FROM membership_requests mr
    JOIN members m ON m.user_id = users.id
    WHERE mr.is_child = 1 AND mr.status = 'approved'
      AND mr.first_name = m.first_name AND mr.last_name = m.last_name
      AND mr.parent_email IS NOT NULL AND mr.parent_email <> ''
)
WHERE users.email IS NULL
  AND users.recovery_email IS NULL
  AND (
    SELECT COUNT(*)
    FROM membership_requests mr2
    JOIN members m2 ON m2.user_id = users.id
    WHERE mr2.is_child = 1 AND mr2.status = 'approved'
      AND mr2.first_name = m2.first_name AND mr2.last_name = m2.last_name
      AND mr2.parent_email IS NOT NULL AND mr2.parent_email <> ''
  ) = 1;
