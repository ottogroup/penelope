/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { Backup } from '../models/Backup';
import type { BigQueryOptions } from '../models/BigQueryOptions';
import type { CreateRequest } from '../models/CreateRequest';
import type { GCSOptions } from '../models/GCSOptions';
import type { MirrorOptions } from '../models/MirrorOptions';
import type { RestoreResponse } from '../models/RestoreResponse';
import type { SnapshotOptions } from '../models/SnapshotOptions';
import type { SourceProject } from '../models/SourceProject';
import type { TargetOptions } from '../models/TargetOptions';
import type { UpdateRequest } from '../models/UpdateRequest';
import type { UserResponse } from '../models/UserResponse';

import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';

export class DefaultService {

    /**
     * Get current user
     * @returns UserResponse OK
     * @throws ApiError
     */
    public static getUsersMe(): CancelablePromise<UserResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/users/me',
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Get all backups
     * @param project Project ID
     * @returns any OK
     * @throws ApiError
     */
    public static getBackups(
        project?: string,
    ): CancelablePromise<{
        backups?: Array<Backup>;
    }> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/backups',
            query: {
                'project': project,
            },
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Create a new backup
     * @param requestBody
     * @returns Backup Created
     * @throws ApiError
     */
    public static postBackups(
        requestBody: CreateRequest,
    ): CancelablePromise<Backup> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/backups',
            body: requestBody,
            mediaType: 'application/json',
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Update a backup
     * @param requestBody
     * @returns Backup OK
     * @throws ApiError
     */
    public static patchBackups(
        requestBody: UpdateRequest,
    ): CancelablePromise<Backup> {
        return __request(OpenAPI, {
            method: 'PATCH',
            url: '/backups',
            body: requestBody,
            mediaType: 'application/json',
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Get a backup
     * @param backupId Backup ID
     * @param size Size of job page
     * @param page Page of job page
     * @returns Backup OK
     * @throws ApiError
     */
    public static getBackups1(
        backupId: string,
        size?: number,
        page?: number,
    ): CancelablePromise<Backup> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/backups/{backupId}',
            path: {
                'backupId': backupId,
            },
            query: {
                'size': size,
                'page': page,
            },
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Calculate backup costs
     * @param requestBody
     * @returns any OK
     * @throws ApiError
     */
    public static postBackupsCalculate(
        requestBody: {
            type?: string;
            strategy?: string;
            project?: string;
            target?: TargetOptions;
            snapshot_options?: SnapshotOptions;
            mirror_options?: MirrorOptions;
            bigquery_options?: BigQueryOptions;
            gcs_options?: GCSOptions;
        },
    ): CancelablePromise<{
        costs?: Array<{
            cost?: number;
            currency?: string;
            name?: string;
            period?: number;
            size_in_bytes?: number;
        }>;
    }> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/backups/calculate',
            body: requestBody,
            mediaType: 'application/json',
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Checks backup compliance level
     * @param requestBody
     * @returns any OK
     * @throws ApiError
     */
    public static postBackupsCompliance(
        requestBody: {
            type?: string;
            strategy?: string;
            project?: string;
            target?: TargetOptions;
            snapshot_options?: SnapshotOptions;
            mirror_options?: MirrorOptions;
            bigquery_options?: BigQueryOptions;
            gcs_options?: GCSOptions;
        },
    ): CancelablePromise<{
        checks?: Array<{
            field?: string;
            passed?: boolean;
            description?: string;
            details?: string;
        }>;
    }> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/backups/compliance',
            body: requestBody,
            mediaType: 'application/json',
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Restore a backup
     * @param backupId Backup ID
     * @param jobIdForTimestamp Job ID for timestamp
     * @returns RestoreResponse Restore response
     * @throws ApiError
     */
    public static getRestore(
        backupId: string,
        jobIdForTimestamp?: string,
    ): CancelablePromise<RestoreResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/restore/{backupId}',
            path: {
                'backupId': backupId,
            },
            query: {
                'jobIDForTimestamp': jobIdForTimestamp,
            },
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Get all available backup regions
     * @returns any OK
     * @throws ApiError
     */
    public static getConfigRegions(): CancelablePromise<{
        regions?: Array<string>;
    }> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/config/regions',
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Get all available backup storge classes
     * @returns any OK
     * @throws ApiError
     */
    public static getConfigStorageClasses(): CancelablePromise<{
        storage_classes?: Array<string>;
    }> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/config/storage_classes',
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Get all datasets
     * @param projectId Project ID
     * @returns any OK
     * @throws ApiError
     */
    public static getDatasets(
        projectId: string,
    ): CancelablePromise<{
        datasets?: Array<string>;
    }> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/datasets/{projectId}',
            path: {
                'projectId': projectId,
            },
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Get all buckets
     * @param projectId Project ID
     * @returns any OK
     * @throws ApiError
     */
    public static getBuckets(
        projectId: string,
    ): CancelablePromise<{
        buckets?: Array<string>;
    }> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/buckets/{projectId}',
            path: {
                'projectId': projectId,
            },
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Get source project
     * @param projectId Project ID
     * @returns any OK
     * @throws ApiError
     */
    public static getSourceProject(
        projectId: string,
    ): CancelablePromise<{
        sourceProject?: SourceProject;
    }> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/sourceProject/{projectId}',
            path: {
                'projectId': projectId,
            },
            errors: {
                400: `Bad Request`,
            },
        });
    }

    /**
     * Run a task
     * @param task Task name
     * @returns any Created
     * @throws ApiError
     */
    public static getTasks(
        task: string,
    ): CancelablePromise<any> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/tasks/{task}',
            path: {
                'task': task,
            },
            errors: {
                403: `Forbidden`,
            },
        });
    }

}
