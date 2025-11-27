import { Table, TableLazyLoadEvent } from 'primeng/table'
import { FilterMetadata } from 'primeng/api/filtermetadata'

/**
 * Checks if given PrimeNG table filters contain any non-blank value.
 * @param filters PrimeNG table filters object
 * @param continueWhen callable that evaluates to boolean value; when evaluated to true, the filter for given filterKey is considered blank even if it has meaningful value
 * @return true if any non-blank filter was found; false otherwise
 */
export function hasFilter(
    filters: { [p: string]: FilterMetadata | FilterMetadata[] } = {},
    continueWhen: (filterKey: string) => boolean = () => false
): boolean {
    for (const [filterKey, filterMetadata] of Object.entries(filters)) {
        if (continueWhen(filterKey)) {
            continue
        }

        if (Array.isArray(filterMetadata)) {
            for (const filter of filterMetadata) {
                if (
                    (filter.matchMode != 'contains' && filter.value !== null) ||
                    (filter.matchMode == 'contains' && filter.value)
                ) {
                    return true
                }
            }
        } else if (filterMetadata) {
            if (
                (filterMetadata.matchMode != 'contains' && filterMetadata.value !== null) ||
                (filterMetadata.matchMode == 'contains' && filterMetadata.value)
            ) {
                return true
            }
        }
    }

    return false
}

/**
 * Checks if given PrimeNG table has filters that contain any non-blank value.
 * @param table PrimeNG table
 * @param continueWhen callable that evaluates to boolean value; when evaluated to true, the filter for given filterKey is considered blank even if it has meaningful value
 * @return true if any non-blank filter was found; false otherwise
 */
export function tableHasFilter(table: Table, continueWhen: (filterKey: string) => boolean = () => false): boolean {
    return hasFilter(table.filters, continueWhen)
}

/**
 * Parses string into boolean value. Returns boolean or null if it couldn't be parsed.
 * @param val
 */
export function parseBoolean(val: string): boolean | null {
    return val === 'true' ? true : val === 'false' ? false : null
}

/**
 * Returns PrimeNG table filters as queryParam object, which may be used for router navigation.
 * @param table PrimeNG table with filters
 * @return filters as queryParam object
 */
export function tableFiltersToQueryParams(table: Table) {
    const entries = Object.entries(table.filters).map((entry) => [entry[0], (<FilterMetadata>entry[1]).value])
    return Object.fromEntries(entries)
}

/**
 * Enumeration of sorting direction options with values used in Stork server backend.
 */
export enum SortDir {
    Asc = 1,
    Desc = 2,
}

/**
 * Function converting PrimeNG table sorting related metadata to REST API
 * sorting fields format.
 * @template TSortField type of possible sorting field values
 * @param event table lazy load event
 * @returns an array of sorting related fields in REST API format
 */
export function convertSortingFields<TSortField>(event: TableLazyLoadEvent): [TSortField, SortDir] {
    if (!event || !event.sortField) {
        return [null, null]
    }

    if (event.sortOrder === null || event.sortOrder === undefined) {
        return [<TSortField>event.sortField, null]
    }

    return [<TSortField>event.sortField, event.sortOrder === -1 ? SortDir.Desc : SortDir.Asc]
}
