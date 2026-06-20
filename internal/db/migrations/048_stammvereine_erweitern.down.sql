-- Entfernt die in 048 hinzugefügten Stammvereine. Schützt vor Datenverlust,
-- falls Mitglieder inzwischen darauf verweisen (FK-Verweis verhindert DELETE
-- nicht automatisch, da home_club_id ohne ON DELETE-Klausel ist — Down nur
-- ausführen, wenn 049 vorher down-migriert wurde).

DELETE FROM stammvereine WHERE name IN (
    'HSG Cannstatt/Münster/Max-Eyth-See',
    'HSG Oberer Neckar',
    'Hbi Weilimdorf/Feuerbach',
    'HSG Gablenberg-Gaisburg',
    'Sportvg Feuerbach',
    'HSV Zuffenhausen',
    'TuS Stuttgart',
    'TV Obertürkheim',
    'TSV Korntal',
    'SV Stuttgarter Kickers',
    'TV Fellbach',
    'TV Deizisau',
    'SG Asperg',
    'SG Hegensberg-Liebersbronn'
);
