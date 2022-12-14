import { Pipe, PipeTransform } from '@angular/core'
import { datetimeToLocal, epochToLocal } from '../utils'

@Pipe({
    name: 'localtime',
})
export class LocaltimePipe implements PipeTransform {
    transform(value: any, ...args: any[]): any {
        // If this is an integer we guess that it is an epoch time.
        if (Number.isInteger(value)) {
            return epochToLocal(value)
        }
        return datetimeToLocal(value)
    }
}
