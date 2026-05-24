CREATE TABLE members_new AS SELECT
    id, first_name, last_name, date_of_birth, member_number, pass_number,
    jersey_number, position, gender, status, user_id, club_function,
    street, zip, city, join_date, iban, account_holder,
    photo_path, photo_visible,
    dsgvo_verarbeitung, dsgvo_verarbeitung_date,
    dsgvo_weitergabe, dsgvo_weitergabe_date,
    sepa_mandat, sepa_mandat_date, sepa_mandat_path
FROM members;
DROP TABLE members;
ALTER TABLE members_new RENAME TO members;
