ALTER TABLE backups
    ADD sink_is_immutable BOOLEAN DEFAULT FALSE NOT NULL;