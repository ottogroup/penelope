/* generated using openapi-typescript-codegen -- do no edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */

import type { Role } from './Role';

export type UserResponse = {
    User?: {
        Email?: string;
    };
    RoleBindings?: Array<{
        Role?: Role;
        Project?: string;
    }>;
};

