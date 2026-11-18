-- Markets registered without coordinates (partner page / bot collection with
-- no address, or a product push that auto-creates its market) were stored at
-- a fixed placeholder point, which made every distance shown to the user
-- meaningless. has_location tells "these coordinates are real" apart from
-- "placeholder": distances are only computed for markets where it is true.
-- Defaults to true so seed/fixture rows with real coordinates stay valid.
ALTER TABLE markets ADD COLUMN has_location boolean NOT NULL DEFAULT true;

UPDATE markets SET has_location = false
WHERE latitude = -5.0892 AND longitude = -42.8016;
