import { Pipe, PipeTransform } from '@angular/core'
import { humanCount } from '../utils'

@Pipe({
    name: 'humanCount',
})
export class HumanCountPipe implements PipeTransform {
    /**
     * Formats the given number using the metric prefixes.
     * @param count Count to format, any object convertible to big integer.
     * @returns Formatted count. If the object is not numeric, returns @count
     *          as is.
     */
    transform(count: any): string {
        if (typeof count === 'string') {
            try {
                count = BigInt(count)
            } catch {
                // Cannot convert, keep it as is.
            }
        }

        return humanCount(count)
    }
}
