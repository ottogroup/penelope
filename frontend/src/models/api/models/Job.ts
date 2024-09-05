/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { JobStatus } from './JobStatus';
export type Job = {
    id?: string;
    backup_id?: string;
    foreign_job_id?: string;
    status?: JobStatus;
    source?: string;
    created?: string;
    updated?: string;
    deleted?: string;
};

