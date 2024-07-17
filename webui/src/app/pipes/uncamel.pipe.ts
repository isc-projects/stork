import { Pipe, PipeTransform } from '@angular/core'
import { uncamelCase } from '../utils'

@Pipe({
    name: 'uncamel',
})
export class UncamelPipe implements PipeTransform {
    /**
     * Converts parameter names from camel case to long names.
     *
     * The words in the long names begin with upper case and are separated with
     * space characters. For example: 'cacheThreshold' becomes 'Cache Threshold'.
     *
     * It also handles several special cases. When the converted name begins with:
     * - ddns - it is converted to DDNS,
     * - pd - it is converted to PD,
     * - ip - it is converted to IP,
     * - underscore character - it is removed.
     *
     * @param key a name to be converted in camel case notation.
     * @returns converted name.
     */
    transform(value: string | null): string | null {
        if (value == null) {
            return value
        }
        return uncamelCase(value)
    }
}
