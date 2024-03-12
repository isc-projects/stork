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
    return [] // currently no boolean query params are needed
}

/**
 * Returns the keys of the boolean filters.
 * @returns List of keys.
 */
export function getBooleanFilterKeys(): (keyof QueryParamsFilter)[] {
    return ['global', 'conflict', 'migrationError']
}

/**
 * Returns the keys of the numeric properties of the QueryParamsFilter.
 * These are to be used for navigation to prefiltered hosts list,
 * e.g. to list of hosts that belong to appId=1.
 * @returns List of keys.
 */
export function getNumericQueryParamsFilterKeys(): (keyof QueryParamsFilter)[] {
    return ['appId']
}

/**
 * Returns the keys of all the numeric filters.
 * @returns List of keys.
 */
export function getNumericFilterKeys(): (keyof QueryParamsFilter)[] {
    return ['appId', 'subnetId', 'keaSubnetId']
}
