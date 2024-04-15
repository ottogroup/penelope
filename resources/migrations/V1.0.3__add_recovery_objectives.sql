ALTER TABLE backups ADD COLUMN recovery_point_objective INT NOT NULL DEFAULT 0;
ALTER TABLE backups ADD COLUMN recovery_time_objective INT NOT NULL DEFAULT 0;