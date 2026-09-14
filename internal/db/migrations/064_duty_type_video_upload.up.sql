-- Video-Dienst-basierter Upload: ein Diensttyp kann als "berechtigt zum
-- Video-Upload" markiert werden. Wer für ein Spiel eine Dienst-Zuweisung auf
-- so einen Diensttyp hat, darf für genau dieses Spiel ein Video hochladen
-- (internal/videos.CanUploadForGameViaDuty) — zusätzlich zur bestehenden
-- Trainer-/sportliche-Leitung-/Vorstand-/Admin-Berechtigung.
--
-- Rein additiv, kein Backfill: Default 0 IST das bisherige Verhalten.
ALTER TABLE duty_types ADD COLUMN grants_video_upload INTEGER NOT NULL DEFAULT 0
    CHECK(grants_video_upload IN (0,1));
