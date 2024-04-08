/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { BackupStrategy } from './BackupStrategy';
import type { BackupType } from './BackupType';
import type { BigQueryOptions } from './BigQueryOptions';
import type { GCSOptions } from './GCSOptions';
import type { MirrorOptions } from './MirrorOptions';
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
};

