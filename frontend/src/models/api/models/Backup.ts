/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { AvailabilityClass } from './AvailabilityClass';
import type { BackupStatus } from './BackupStatus';
import type { BackupStrategy } from './BackupStrategy';
import type { BackupType } from './BackupType';
import type { BigQueryOptions } from './BigQueryOptions';
import type { GCSOptions } from './GCSOptions';
import type { Job } from './Job';
import type { MirrorOptions } from './MirrorOptions';
import type { RecoveryPointObjective } from './RecoveryPointObjective';
import type { RecoveryTimeObjective } from './RecoveryTimeObjective';
import type { SnapshotOptions } from './SnapshotOptions';
import type { TargetOptions } from './TargetOptions';
export type Backup = {
    id?: string;
    type?: BackupType;
    strategy?: BackupStrategy;
    project?: string;
    target?: TargetOptions;
    snapshot_options?: SnapshotOptions;
    mirror_options?: MirrorOptions;
    bigquery_options?: BigQueryOptions;
    gcs_options?: GCSOptions;
    status?: BackupStatus;
    sink?: string;
    sink_project?: string;
    created?: string;
    updated?: string;
    deleted?: string;
    jobs?: Array<Job>;
    jobs_total?: number;
    data_owner?: string;
    data_availability_class?: AvailabilityClass;
    recovery_point_objective?: RecoveryPointObjective;
    recovery_time_objective?: RecoveryTimeObjective;
};

