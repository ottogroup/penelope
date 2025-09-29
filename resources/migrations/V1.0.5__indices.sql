CREATE INDEX jobs_backup_id
    ON jobs (backup_id);

CREATE INDEX source_metadata_backup_id
    ON source_metadata (backup_id);