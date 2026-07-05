-- teams.typo3_category_uid: Verknüpfung zwischen einem TeamWERK-Team und der
-- sys_category-UID auf team-stuttgart.org, unter der die Spielberichte
-- gefiltert werden (spielbericht-typo3-publisher). NULL = keine Mapping —
-- Bericht wird veröffentlicht, aber nicht auf Team-Seite gefiltert.

ALTER TABLE teams ADD COLUMN typo3_category_uid INTEGER;
