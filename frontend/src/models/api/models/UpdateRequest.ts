/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

export type UpdateRequest = {
    backup_id?: string;
    status?: string;
    mirror_ttl?: number;
    snapshot_ttl?: number;
    archive_ttm?: number;
    include_path?: Array<string>;
    exclude_path?: Array<string>;
    table?: Array<string>;
    excluded_tables?: Array<string>;
};

