/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
 
export enum AvailabilityClass {
    /**
     * A1 Irrelevant - A recovery test SHOULD be conducted after changes to the backup process.
     */
    A1 = 'A1',
    /**
     * A2 Aimed - A recovery test SHOULD be conducted after changes to the backup process and at least once every year.
     */
    A2 = 'A2',
    /**
     * A3 Guaranteed - A recovery test MUST be conducted after changes to the backup process and at least once every six months.
     */
    A3 = 'A3',
    /**
     * A4 Resilient - A recovery test MUST be conducted after changes to the backup process and at least once every three months. A recovery test SHOULD also be conducted automatically after every backup (restore and verification).
     */
    A4 = 'A4',
}
