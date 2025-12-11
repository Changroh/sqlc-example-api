ALTER TABLE notifications ALTER COLUMN last_error DROP NOT NULL;
ALTER TABLE notifications ALTER COLUMN last_error DROP DEFAULT;

