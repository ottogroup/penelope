/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { BackupStatus } from './BackupStatus';
import type { RecoveryPointObjective } from './RecoveryPointObjective';
import type { RecoveryTimeObjective } from './RecoveryTimeObjective';
export type UpdateRequest = {
    backup_id?: string;
    description?: string;
    status?: BackupStatus;
    mirror_ttl?: number;
    snapshot_ttl?: number;
    archive_ttm?: number;
    include_path?: Array<string>;
    exclude_path?: Array<string>;
    table?: Array<string>;
    excluded_tables?: Array<string>;
    recovery_point_objective?: RecoveryPointObjective;
    recovery_time_objective?: RecoveryTimeObjective;
};

