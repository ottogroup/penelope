CREATE INDEX CONCURRENTLY idx_check_jobs_backup_source_status
    ON jobs (backup_id, source, status)
    WHERE audit_deleted_timestamp IS NULL;