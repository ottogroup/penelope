alter table backups add cloudstorage_bucket text;
ALTER TABLE jobs ALTER COLUMN status SET NOT NULL;
