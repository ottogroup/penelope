ALTER TABLE backups
    ADD trashcan_cleanup_status TEXT DEFAULT 'Noop';
ALTER TABLE backups
    ADD trashcan_cleanup_error_message TEXT DEFAULT NULL;