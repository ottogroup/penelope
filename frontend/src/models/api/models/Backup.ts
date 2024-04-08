/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { BackupStatus } from './BackupStatus';
import type { BackupStrategy } from './BackupStrategy';
import type { BackupType } from './BackupType';
import type { BigQueryOptions } from './BigQueryOptions';
import type { GCSOptions } from './GCSOptions';
import type { Job } from './Job';
import type { MirrorOptions } from './MirrorOptions';
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
};

