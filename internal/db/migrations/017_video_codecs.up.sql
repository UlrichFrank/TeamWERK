-- 017_video_codecs: HLS-CODECS-Attribute pro Video für AirPlay/Cast-Kompatibilität.
-- Wird nach dem Transcode per ffprobe auf seg_001.ts der 720p-Rendition ermittelt
-- und in writeMasterManifest an das STREAM-INF gehängt. Ohne diese Signalisierung
-- baut tvOS den Video-Decoder nicht auf → AirPlay liefert nur Ton, kein Bild.
-- Format: "avc1.PPCCLL,mp4a.40.X" (H.264 Profile+Level + AAC Object-Type).
ALTER TABLE videos ADD COLUMN codecs TEXT;
