-- Bild-Dimensionen (width/height) für Chat-/Broadcast-Bilder.
-- Der Upload-Handler probet ab jetzt per image.DecodeConfig (JPEG/PNG/GIF)
-- bzw. webp.DecodeConfig die Header-Dimensionen und schreibt sie mit.
-- Bestandszeilen bleiben NULL, bis internal/media/backfill.go sie beim
-- nächsten Serverstart nachträgt. Response-Felder mediaWidth/mediaHeight
-- werden omitempty gerendert, alte Bilder ohne Dims bleiben clientseitig
-- durch den bestehenden AuthImage-Probe kompatibel.

ALTER TABLE media ADD COLUMN width  INTEGER;
ALTER TABLE media ADD COLUMN height INTEGER;
