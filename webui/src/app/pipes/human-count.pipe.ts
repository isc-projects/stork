import { Pipe, PipeTransform } from '@angular/core'
import { humanCount } from '../utils'

@Pipe({
    name: 'humanCount',
})
export class HumanCountPipe implements PipeTransform {
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
