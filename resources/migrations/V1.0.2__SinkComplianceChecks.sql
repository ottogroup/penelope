CREATE TABLE sink_compliance_checks
(
    project_sink TEXT    NOT NULL PRIMARY KEY,
    compliant    BOOLEAN NOT NULL DEFAULT FALSE,
    reasons      TEXT,
    last_checked TIMESTAMP
);
