/**
 * Specifies the filter parameters for fetching hosts that may be specified
 * either in the URL query parameters or programmatically.
 */
export interface HostsFilter {
    text?: string
    appId?: number
    subnetId?: number
    keaSubnetId?: number
    isGlobal?: boolean
    conflict?: boolean
    migrationError?: boolean
}

/**
 * Returns the keys of the boolean properties of the QueryParamsFilter.
 * @returns List of keys.
 */
export function getBooleanQueryParamsFilterKeys(): (keyof HostsFilter)[] {
    return [] // currently no boolean query params are needed
}

/**
 * Returns the keys of the boolean filters.
 * @returns List of keys.
 */
export function getBooleanFilterKeys(): (keyof HostsFilter)[] {
    return ['isGlobal', 'conflict', 'migrationError']
}

/**
 * Returns the keys of the numeric properties of the QueryParamsFilter.
 * These are to be used for navigation to prefiltered hosts list,
 * e.g. to list of hosts that belong to appId=1.
 * @returns List of keys.
 */
export function getNumericQueryParamsFilterKeys(): (keyof HostsFilter)[] {
    return ['appId']
}

/**
 * Returns the keys of all the numeric filters.
 * @returns List of keys.
 */
export function getNumericFilterKeys(): (keyof HostsFilter)[] {
    return ['appId', 'subnetId', 'keaSubnetId']
}
