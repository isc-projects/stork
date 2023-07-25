import { Pipe, PipeTransform } from '@angular/core'
import { datetimeToLocal, epochToLocal } from '../utils'

@Pipe({
    name: 'localtime',
})
export class LocaltimePipe implements PipeTransform {
    /**
     * Formats a given value as local date-time.
     * @param value If the value is integer, it is treated as epoch timestamp.
     *              Otherwise, it is parsed to get date object.
     * @returns Formatted date or stringified value.
     */
    transform(value: moment.MomentInput) {
        // If this is an integer we guess that it is an epoch time.
        if (Number.isInteger(value)) {
            return epochToLocal(value)
        }
        return datetimeToLocal(value)
    }
}
