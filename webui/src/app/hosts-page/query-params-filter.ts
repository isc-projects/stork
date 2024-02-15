/**
 * Specifies the filter parameters for fetching hosts that may be specified
 * in the URL query parameters.
 */
export interface QueryParamsFilter {
    text?: string
    appId?: number
    subnetId?: number
    keaSubnetId?: number
    global?: boolean
    conflict?: boolean
    migrationError?: boolean
}

/**
 * Returns the keys of the boolean properties of the QueryParamsFilter.
 * @returns List of keys.
 */
export function getBooleanQueryParamsFilterKeys(): (keyof QueryParamsFilter)[] {
    return ['global', 'conflict', 'migrationError']
}

/**
 * Returns the keys of the numeric properties of the QueryParamsFilter.
 * @returns List of keys.
 */
export function getNumericQueryParamsFilterKeys(): (keyof QueryParamsFilter)[] {
    return ['appId', 'subnetId', 'keaSubnetId']
}
