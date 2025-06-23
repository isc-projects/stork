import { Pipe, PipeTransform } from '@angular/core'

/**
 * Transforms a daemon name to a nice name.
 *
 * @param value daemon name to transform.
 * @returns nice name.
 */
@Pipe({
    name: 'daemonNiceName',
})
export class DaemonNiceNamePipe implements PipeTransform {
    /**
     * Transforms a daemon name to a nice name.
     *
     * @param value daemon name to transform.
     * @returns nice name.
     */
    transform(value: string): string {
        switch (value) {
            case 'dhcp4':
                return 'DHCPv4'
            case 'dhcp6':
                return 'DHCPv6'
            case 'd2':
                return 'DDNS'
            case 'ca':
                return 'CA'
            case 'netconf':
                return 'NETCONF'
            case 'named':
                return 'named'
            case 'pdns':
                return 'powerdns_server'
            default:
                return value
        }
    }
}
