import { Pipe, PipeTransform } from '@angular/core'
import { daemonNameToFriendlyName } from '../utils'

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
        return daemonNameToFriendlyName(value)
    }
}
