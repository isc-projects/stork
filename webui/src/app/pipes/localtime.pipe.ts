import { Pipe, PipeTransform } from '@angular/core'
import { datetimeToLocal, epochToLocal } from '../utils'

@Pipe({ name: 'localtime' })
export class LocaltimePipe implements PipeTransform {
    /**
     * Formats a given value as local date-time.
     *
     * If the specified value is a number, it is treated as epoch timestamp.
     * It can be an integer or a float. In the latter case, the fractional part
     * is treated as milliseconds. If the floating point number contains more than
     * 3 digits after the decimal point, the extra digits are ignored.
     * If the value is not a number, it is parsed to get date object.
     *
     * @param value Value to be formatted.
     * @returns Formatted date or stringified value.
     */
    transform(value: moment.MomentInput, milliseconds = false) {
        // If this is an integer we guess that it is an epoch time.
        if (typeof value === 'number') {
            return epochToLocal(value, milliseconds)
        }
        return datetimeToLocal(value, milliseconds)
    }
}
