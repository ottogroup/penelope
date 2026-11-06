CREATE TABLE backup_maintenance_tasks
(
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    backup_id TEXT NOT NULL REFERENCES backups (id),
    task_name TEXT NOT NULL CHECK (btrim(task_name) <> ''),
    task_parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    audit_created_timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    audit_updated_timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    audit_finished_timestamp TIMESTAMPTZ,
    locked_by TEXT,
    lock_token UUID,
    lease_expires_at TIMESTAMPTZ,
    last_heartbeat_at TIMESTAMPTZ,
    CONSTRAINT backup_maintenance_tasks_lease_check CHECK (
        (locked_by IS NULL AND lock_token IS NULL
            AND lease_expires_at IS NULL AND last_heartbeat_at IS NULL)
        OR
        (locked_by IS NOT NULL AND btrim(locked_by) <> ''
            AND lock_token IS NOT NULL
            AND lease_expires_at IS NOT NULL AND last_heartbeat_at IS NOT NULL
            AND lease_expires_at > last_heartbeat_at
            AND audit_finished_timestamp IS NULL)
    )
);

CREATE UNIQUE INDEX backup_maintenance_tasks_unfinished_unique
    ON backup_maintenance_tasks (backup_id, task_name)
    WHERE audit_finished_timestamp IS NULL;

CREATE INDEX backup_maintenance_tasks_claim_idx
    ON backup_maintenance_tasks (lease_expires_at NULLS FIRST, id)
    WHERE audit_finished_timestamp IS NULL;

CREATE FUNCTION update_backup_maintenance_task_timestamp()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.audit_updated_timestamp := clock_timestamp();
    RETURN NEW;
END;
$$;

CREATE TRIGGER backup_maintenance_tasks_updated
    BEFORE UPDATE ON backup_maintenance_tasks
    FOR EACH ROW EXECUTE FUNCTION update_backup_maintenance_task_timestamp();
