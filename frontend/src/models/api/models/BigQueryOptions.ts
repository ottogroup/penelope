/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
export type BigQueryOptions = {
    dataset?: string;
    table?: Array<string>;
    excluded_tables?: Array<string>;
    /**
     * Create native BigQuery table snapshots instead of exporting to a GCS sink bucket. Only valid for strategy=Snapshot. Immutable after creation.
     */
    use_native_table_snapshots?: boolean;
};

