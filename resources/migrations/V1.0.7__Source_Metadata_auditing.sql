alter table source_metadata add operation TEXT default 'Add' not null;
alter table source_metadata add audit_created_timestamp timestamp DEFAULT NOW() not null;
alter table source_metadata add audit_deleted_timestamp timestamp;
alter table source_metadata drop constraint source_metadata_backup_id_source_key;


CREATE TABLE IF NOT EXISTS source_metadata_jobs
(
    source_metadata_id  int REFERENCES source_metadata (id) ON UPDATE CASCADE ON DELETE CASCADE,
    job_id TEXT REFERENCES jobs (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT source_metadata_jobs_pkey PRIMARY KEY (source_metadata_id, job_id),
    CONSTRAINT source_metadata_id_uq UNIQUE (source_metadata_id),
    CONSTRAINT job_id_uq UNIQUE (job_id)
);
