create table backups
(
    id text not null
        constraint backups_pkey
            primary key,
    type text,
    status text,
    project text,
    strategy text,
    target_project text,
    target_region text,
    target_sink text,
    target_storage_class text,
    snapshot_lifetime_in_days smallint
        constraint backups_snapshot_lifetime_in_days_check
            check (snapshot_lifetime_in_days >= 0),
    snapshot_frequency_in_hours smallint
        constraint backups_snapshot_frequency_in_hours_check
            check (snapshot_frequency_in_hours >= 0),
    last_scheduled_timestamp timestamp,
    bigquery_dataset text,
    bigquery_table text,
    cloudstorage_include_path text,
    cloudstorage_exclude_path text,
    audit_created_timestamp timestamp default now(),
    audit_updated_timestamp timestamp,
    audit_deleted_timestamp timestamp,
    cloudstorage_bucket text,
    last_cleanup_timestamp timestamp default now() not null,
    bigquery_excluded_tables text,
    mirror_lifetime_in_days smallint
        constraint backups_mirror_lifetime_in_days_check
            check (mirror_lifetime_in_days >= 0),
    archive_ttm smallint
        constraint backups_archive_ttm_check
            check (archive_ttm >= 0)
);

create table jobs
(
    id text not null
        constraint jobs_pkey
            primary key,
    backup_id text
        constraint jobs_backup_id_fkey
            references backups,
    type text,
    status text not null,
    source text,
    bigquery_extract_job_id text,
    cloudstorage_transfer_job_id text,
    audit_created_timestamp timestamp default now(),
    audit_updated_timestamp timestamp,
    audit_deleted_timestamp timestamp
);

create table source_metadata
(
    id serial not null
        constraint source_metadata_pkey
            primary key,
    backup_id text
        constraint source_metadata_backup_id_fkey
            references backups,
    source text,
    source_checksum text,
    operation text default 'Add'::text not null,
    audit_created_timestamp timestamp default now() not null,
    audit_deleted_timestamp timestamp
);

create table source_metadata_jobs
(
    source_metadata_id integer not null
        constraint source_metadata_id_uq
            unique
        constraint source_metadata_jobs_source_metadata_id_fkey
            references source_metadata
            on update cascade on delete cascade,
    job_id text not null
        constraint job_id_uq
            unique
        constraint source_metadata_jobs_job_id_fkey
            references jobs
            on update cascade on delete cascade,
    constraint source_metadata_jobs_pkey
        primary key (source_metadata_id, job_id)
);

create table source_trashcan
(
    backup_id text
        constraint source_trashcan_backup_id_fkey
            references backups
        constraint source_trashcan_backup_id_check
            check (backup_id IS NOT NULL),
    source text
        constraint source_trashcan_source_check
            check (source IS NOT NULL),
    audit_created_timestamp timestamp
        constraint source_trashcan_audit_created_timestamp_check
            check (audit_created_timestamp IS NOT NULL),
    constraint source_trashcan_backup_id_source_key
        unique (backup_id, source)
);