import { ZoneInventoryState } from './backend'
import StatusEnum = ZoneInventoryState.StatusEnum

/**
 * Returns tooltip message for given ZoneInventoryState status.
 * @param status ZoneInventoryState status
 * @return tooltip message
 */
export function getTooltip(status: StatusEnum) {
    switch (status) {
        case 'busy':
            return 'Zone inventory on the agent is busy and cannot return zones at this time. Try again later.'
        case 'ok':
            return 'Stork server successfully fetched all zones from the DNS server.'
        case 'erred':
            return 'Error when communicating with a zone inventory on an agent.'
        case 'uninitialized':
            return 'Zone inventory on the agent was not initialized. Trying again or restarting the agent can help.'
        default:
            return null
    }
}

/**
 * Returns PrimeNG severity for given ZoneInventoryState status.
 * @param status ZoneInventoryState status
 * @return PrimeNG severity
 */
export function getSeverity(status: StatusEnum) {
    switch (status) {
        case 'ok':
            return 'success'
        case 'busy':
            return 'warn'
        case 'erred':
            return 'danger'
        case 'uninitialized':
            return 'secondary'
        default:
            return 'info'
    }
}
