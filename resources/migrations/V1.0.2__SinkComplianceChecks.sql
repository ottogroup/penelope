CREATE TABLE sink_compliance_checks
(
    project_sink  TEXT    NOT NULL PRIMARY KEY,
    backup_only   BOOLEAN NOT NULL,
    single_writer BOOLEAN NOT NULL,
    last_checked  TIMESTAMP
);
