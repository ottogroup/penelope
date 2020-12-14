CREATE TABLE IF NOT EXISTS source_trashcan
(
    backup_id                    TEXT CHECK (backup_id IS NOT NULL),
    source                       TEXT CHECK (source IS NOT NULL),
    audit_created_timestamp      timestamp CHECK (audit_created_timestamp IS NOT NULL),
    FOREIGN KEY (backup_id) REFERENCES backups (id),
    UNIQUE (backup_id, source)
);


alter table backups add last_cleanup_timestamp timestamp DEFAULT NOW() not null;
alter table backups rename column last_scheduled to last_scheduled_timestamp;
