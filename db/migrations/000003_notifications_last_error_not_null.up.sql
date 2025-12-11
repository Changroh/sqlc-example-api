UPDATE notifications SET last_error = '' WHERE last_error IS NULL;
ALTER TABLE notifications ALTER COLUMN last_error SET DEFAULT '';
ALTER TABLE notifications ALTER COLUMN last_error SET NOT NULL;

