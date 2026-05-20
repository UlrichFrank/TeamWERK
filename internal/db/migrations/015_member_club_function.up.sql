ALTER TABLE members ADD COLUMN club_function TEXT CHECK(club_function IN ('trainer','vorstand','vorstand_beisitzer'));
