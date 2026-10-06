ALTER TABLE backups
    ADD bigquery_use_native_table_snapshots boolean DEFAULT false NOT NULL;
