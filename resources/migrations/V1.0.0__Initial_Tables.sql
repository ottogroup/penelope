
CREATE TABLE IF NOT EXISTS backups
(
  id                        TEXT PRIMARY KEY,
  type                      TEXT,
  status                    TEXT,
  project                   TEXT,
  strategy                  TEXT,

  target_project            TEXT,
  target_region             TEXT,
  target_sink               TEXT,
  target_storage_class      TEXT,

  snapshot_lifetime_in_days smallint CHECK (snapshot_lifetime_in_days >= 0),
  snapshot_frequency_in_hours         smallint CHECK (snapshot_frequency_in_hours >= 0),
  snapshot_last_scheduled       timestamp,

  bigquery_dataset          TEXT,
  bigquery_table            TEXT,

  cloudstorage_include_path TEXT,
  cloudstorage_exclude_path TEXT,

  audit_created_timestamp   timestamp DEFAULT NOW(),
  audit_updated_timestamp   timestamp,
  audit_deleted_timestamp   timestamp,
  UNIQUE (id)
);

CREATE TABLE IF NOT EXISTS jobs (
  id                           TEXT PRIMARY KEY,
  backup_id                    TEXT,
  type                         TEXT,
  status                       TEXT,
  source                       TEXT,
  source_checksum              TEXT,

  bigquery_extract_job_id      TEXT,
  cloudstorage_transfer_job_id TEXT,

  audit_created_timestamp      timestamp  DEFAULT NOW(),
  audit_updated_timestamp      timestamp,
  audit_deleted_timestamp      timestamp,
  UNIQUE (id),
  FOREIGN KEY (backup_id) REFERENCES backups (id)
);


CREATE TABLE IF NOT EXISTS source_metadata (
  id                SERIAL PRIMARY KEY,
  backup_id         TEXT,
  source            TEXT,
  source_checksum   TEXT,
  UNIQUE (id),
  UNIQUE (backup_id, source), /* we cannot put into backup the same table twice */
  FOREIGN KEY (backup_id) REFERENCES backups (id)
);
