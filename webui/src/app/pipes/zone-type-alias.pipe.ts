import { Pipe, PipeTransform } from '@angular/core'

/**
 * Transforms 'master' to 'primary' and 'slave' to 'secondary' zone type.
 *
 * @param value zone type
 * @returns zone type using newer naming convention.
 */
@Pipe({
    name: 'zoneTypeAlias',
    standalone: true,
})
export class ZoneTypeAliasPipe implements PipeTransform {
    /**
     * Transforms 'master' to 'primary' and 'slave' to 'secondary' zone type.
     *
     * @param value zone type
     * @returns zone type using newer naming convention.
     */
    transform(value: string): string {
        switch (value) {
            case 'master':
                return 'primary'
            case 'slave':
                return 'secondary'
            default:
                return value
        }
    }
}
