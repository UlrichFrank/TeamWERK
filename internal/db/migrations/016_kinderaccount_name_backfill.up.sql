-- Backfill der Namen für Kinderkonten, die vor dem Fix ohne first_name/last_name
-- angelegt wurden (approveChildRequest setzte die Spalten früher nicht).
--
-- Kinderkonten haben KEIN verknüpftes Mitglied (reine Korrespondenz), daher ist
-- der Name-über-members-Join aus Migration 004 hier nicht anwendbar. Die einzige
-- verlustfreie Namensquelle ist der Beitrittsantrag (membership_requests); der
-- login_name ist verlustbehaftet (Umlaut-Transliteration, Zeichen-Strip,
-- Kollisions-Suffix) und dient nur der Disambiguierung, nicht als Namensquelle.
--
-- Match: recovery_email == parent_email UND login_name == "<Vorname>.<Nachname>"
-- (case-insensitiv). Nur befüllen, wenn der Antrag EINDEUTIG zuordenbar ist
-- (genau ein passender approved Kinderantrag). Mehrdeutige oder wegen
-- Umlaut/Suffix nicht matchbare Konten bleiben unangetastet (kein Ratewerk).
UPDATE users SET
    first_name = (
        SELECT mr.first_name FROM membership_requests mr
        WHERE mr.is_child = 1 AND mr.status = 'approved'
          AND mr.parent_email IS NOT NULL AND mr.parent_email <> ''
          AND LOWER(mr.parent_email) = LOWER(users.recovery_email)
          AND LOWER(users.login_name) = LOWER(mr.first_name || '.' || mr.last_name)
    ),
    last_name = (
        SELECT mr.last_name FROM membership_requests mr
        WHERE mr.is_child = 1 AND mr.status = 'approved'
          AND mr.parent_email IS NOT NULL AND mr.parent_email <> ''
          AND LOWER(mr.parent_email) = LOWER(users.recovery_email)
          AND LOWER(users.login_name) = LOWER(mr.first_name || '.' || mr.last_name)
    )
WHERE users.can_login = 0
  AND users.email IS NULL
  AND users.login_name IS NOT NULL
  AND COALESCE(users.first_name, '') = ''
  AND (
    SELECT COUNT(*) FROM membership_requests mr2
    WHERE mr2.is_child = 1 AND mr2.status = 'approved'
      AND mr2.parent_email IS NOT NULL AND mr2.parent_email <> ''
      AND LOWER(mr2.parent_email) = LOWER(users.recovery_email)
      AND LOWER(users.login_name) = LOWER(mr2.first_name || '.' || mr2.last_name)
  ) = 1;
