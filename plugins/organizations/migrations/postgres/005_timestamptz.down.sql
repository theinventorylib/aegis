-- Revert organization-plugin timestamps to RFC3339 TEXT.
ALTER TABLE organization
    ALTER COLUMN created_at TYPE TEXT USING created_at::text,
    ALTER COLUMN updated_at TYPE TEXT USING updated_at::text;

ALTER TABLE members
    ALTER COLUMN created_at TYPE TEXT USING created_at::text,
    ALTER COLUMN updated_at TYPE TEXT USING updated_at::text;

ALTER TABLE team
    ALTER COLUMN created_at TYPE TEXT USING created_at::text,
    ALTER COLUMN updated_at TYPE TEXT USING updated_at::text;

ALTER TABLE team_member
    ALTER COLUMN created_at TYPE TEXT USING created_at::text,
    ALTER COLUMN updated_at TYPE TEXT USING updated_at::text;

ALTER TABLE invitation
    ALTER COLUMN expires_at TYPE TEXT USING expires_at::text,
    ALTER COLUMN created_at TYPE TEXT USING created_at::text,
    ALTER COLUMN updated_at TYPE TEXT USING updated_at::text;
