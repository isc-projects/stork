import { Pipe, PipeTransform } from '@angular/core'

/**
 * A pipe zeroing negative numbers.
 */
@Pipe({
    name: 'positive',
})
export class PositivePipe implements PipeTransform {
    /**
     * Transforms a number such that the negative number becomes 0. Other
     * numbers are returned without change.
     *
     * @param value value to be transformed.
     * @returns Zero when the number is negative, the input value otherwise.
     */
    transform(value: bigint | number | null): bigint | number | null {
        if (value == null) {
            return value
        }
        return value < 0 ? 0 : value
    }
}
