/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
 
import type { BackupStrategy } from './BackupStrategy';
import type { BackupType } from './BackupType';
import type { BigQueryOptions } from './BigQueryOptions';
import type { GCSOptions } from './GCSOptions';
import type { MirrorOptions } from './MirrorOptions';
import type { RecoveryPointObjective } from './RecoveryPointObjective';
import type { RecoveryTimeObjective } from './RecoveryTimeObjective';
import type { SnapshotOptions } from './SnapshotOptions';
import type { TargetOptions } from './TargetOptions';
export type CreateRequest = {
    type?: BackupType;
    strategy?: BackupStrategy;
    project?: string;
    target?: TargetOptions;
    snapshot_options?: SnapshotOptions;
    mirror_options?: MirrorOptions;
    bigquery_options?: BigQueryOptions;
    gcs_options?: GCSOptions;
    recovery_point_objective?: RecoveryPointObjective;
    recovery_time_objective?: RecoveryTimeObjective;
};

