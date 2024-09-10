import { Pipe, PipeTransform } from '@angular/core'

@Pipe({
    name: 'placeholder',
})
export class PlaceholderPipe implements PipeTransform {
    /**
     * Returns a placeholder if the provided string or array is empty or unspecified
     * (null or undefined).
     *
     * @param value a value.
     * @param unspecified a placeholder output when the value is null.
     * @param empty a placeholder output when the value is empty.
     * @returns A placeholder or a converted value to string.
     */
    transform(
        value: string | any[] | null,
        unspecified: string = '(not specified)',
        empty: string = '(empty)'
    ): string {
        if (value == null) {
            return unspecified
        } else if (value === '' || (Array.isArray(value) && value.length === 0)) {
            return empty
        } else {
            // Explicitly convert to a string because the actual type of
            // the value can be different.
            return value.toString()
        }
    }
}
