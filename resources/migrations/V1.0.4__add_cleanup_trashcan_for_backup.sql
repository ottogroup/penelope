ALTER TABLE backups
    ADD trashcan_cleanup_status TEXT DEFAULT 'Noop';
ALTER TABLE backups
    ADD trashcan_cleanup_error_message TEXT DEFAULT NULL;
ALTER TABLE backups
    ADD trashcan_cleanup_last_scheduled_timestamp TEXT DEFAULT NULL;