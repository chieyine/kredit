-- +goose Up
-- Phone identifiers were stored as dialled, so 08012345678 and +2348012345678
-- created two accounts for one person and split their trade history. This
-- canonicalises the stored form to E.164 and mirrors internal/auth.NormalizePhone
-- exactly, so the application and the database always agree on account identity.
--
-- In-flight OTP challenges are keyed by an HMAC of the identifier as it was
-- normalised when the challenge was issued. Challenges created before this
-- migration will no longer match and must be requested again; they expire in
-- ten minutes, so the window is bounded.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.normalize_phone(value TEXT) RETURNS TEXT
LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE
  trimmed TEXT;
  digits TEXT;
  dialled BOOLEAN;
BEGIN
  IF value IS NULL THEN
    RETURN NULL;
  END IF;
  trimmed := btrim(value);
  dialled := left(trimmed, 1) = '+';
  digits := regexp_replace(trimmed, '[^0-9]', '', 'g');
  IF digits = '' THEN
    RETURN '';
  END IF;
  IF length(digits) = 13 AND left(digits, 3) = '234' THEN
    RETURN '+' || digits;
  END IF;
  IF length(digits) = 11 AND left(digits, 1) = '0' THEN
    RETURN '+234' || substr(digits, 2);
  END IF;
  IF length(digits) = 10 AND NOT dialled THEN
    RETURN '+234' || digits;
  END IF;
  IF dialled THEN
    RETURN '+' || digits;
  END IF;
  RETURN digits;
END;
$$;
-- +goose StatementEnd

-- Fail closed. Two rows that canonicalise to the same number are two accounts
-- for one person: merging them automatically would silently move obligations
-- between identities, so an operator must resolve them before this runs.
-- +goose StatementBegin
DO $$
DECLARE
  collisions INTEGER;
BEGIN
  SELECT count(*) INTO collisions FROM (
    SELECT app.normalize_phone(normalized_phone)
    FROM app.users
    WHERE normalized_phone IS NOT NULL
    GROUP BY 1
    HAVING count(*) > 1
  ) duplicated;
  IF collisions > 0 THEN
    RAISE EXCEPTION 'canonicalising phone identifiers would merge % duplicate account group(s); resolve them before migrating', collisions;
  END IF;
END;
$$;
-- +goose StatementEnd

UPDATE app.users
SET normalized_phone = app.normalize_phone(normalized_phone)
WHERE normalized_phone IS NOT NULL
  AND normalized_phone IS DISTINCT FROM app.normalize_phone(normalized_phone);

ALTER TABLE app.users
  DROP CONSTRAINT IF EXISTS users_phone_canonical;
ALTER TABLE app.users
  ADD CONSTRAINT users_phone_canonical
  CHECK (normalized_phone IS NULL OR normalized_phone = app.normalize_phone(normalized_phone))
  NOT VALID;
ALTER TABLE app.users VALIDATE CONSTRAINT users_phone_canonical;

-- +goose Down
ALTER TABLE app.users DROP CONSTRAINT IF EXISTS users_phone_canonical;
DROP FUNCTION IF EXISTS app.normalize_phone(TEXT);
